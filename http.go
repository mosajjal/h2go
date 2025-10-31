package h2go

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"fmt"
	"net"

	"errors"

	"sync"

	"io"

	"github.com/google/uuid"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	CONNECT    = "/connect"
	PING       = "/ping"
	PULL       = "/pull"
	PUSH       = "/push"
	DOWNLOAD   = "/download"
	CHUNK_PULL = "/chunk_pull"
	CHUNK_PUSH = "/chunk_push"
)
const (
	DATA_TYP  = "data"
	QUIT_TYP  = "quit"
	HEART_TYP = "heart"
)

const (
	timeout  = 10
	signTTL  = 10
	heartTTL = 60
)

const (
	version = "20170803"
)

type DevZero struct {
}

func (z DevZero) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}

	return len(b), nil
}

var bufPool = &sync.Pool{New: func() interface{} { return make([]byte, 1024*8) }}

type httpProxy struct {
	addr     string
	secret   string
	proxyMap map[string]*proxyConn
	sync.Mutex
	https  bool
	logger *slog.Logger
}

func NewHttpProxy(logger *slog.Logger, addr, secret string, https bool) *httpProxy {
	if logger == nil {
		logger = DefaultLogger()
	}
	return &httpProxy{addr: addr,
		secret:   secret,
		proxyMap: make(map[string]*proxyConn),
		https:    https,
		logger:   logger,
	}
}

func (hp *httpProxy) handler() {
	http.HandleFunc(CONNECT, hp.connect)
	http.HandleFunc(PULL, hp.pull)
	http.HandleFunc(PUSH, hp.push)
	http.HandleFunc(PING, hp.ping)
	http.HandleFunc(CHUNK_PULL, hp.chunkPull)
	http.HandleFunc(CHUNK_PUSH, hp.chunkPush)
}

func (hp *httpProxy) ListenHTTPS(cert, key string) {
	hp.handler()
	hp.logger.Info("starting the https/http2 server",
		"addr", hp.addr)
	
	// Create HTTP/2 server with TLS
	server := &http.Server{
		Addr:    hp.addr,
		Handler: nil,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2
		},
	}
	
	// Configure HTTP/2
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		hp.logger.Error("error configuring http2", "msg", err)
		return
	}
	
	hp.logger.Error("error", "msg", server.ListenAndServeTLS(cert, key))
}

func (hp *httpProxy) Listen() {
	hp.handler()
	hp.logger.Info("starting the http/http2 server (h2c)",
		"addr", hp.addr)
	
	// Create HTTP/2 server without TLS (h2c - HTTP/2 cleartext)
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:    hp.addr,
		Handler: h2c.NewHandler(http.DefaultServeMux, h2s),
	}
	
	hp.logger.Error("error", "msg", server.ListenAndServe())
}

func (hp *httpProxy) download(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Length", fmt.Sprintf("%d", 100<<20))
	io.CopyN(w, DevZero{}, 100<<20)
}

func (hp *httpProxy) verify(r *http.Request) error {
	ts := r.Header.Get("timestamp")
	if ts == "" {
		return errors.New("timestamp is empty")
	}
	sign := r.Header.Get("sign")
	tm, err := strconv.ParseInt(ts, 10, 0)
	if err != nil {
		return fmt.Errorf("timestamp invalid: %v", err)
	}
	now := time.Now().Unix()
	if now-tm > signTTL {
		return errors.New("timestamp expire")
	}
	if VerifyHMACSHA1(hp.secret, ts, sign) {
		return nil
	}
	return errors.New("sign invalid")
}

func (hp *httpProxy) before(w http.ResponseWriter, r *http.Request) error {
	err := hp.verify(r)
	if err != nil {
		hp.logger.Warn("error while verifying the request",
			"msg", err)
		WriteNotFoundError(w, "404")
	}
	return err
}

func (hp *httpProxy) ping(w http.ResponseWriter, r *http.Request) {
	hp.logger.Debug("ping",
		"remote", r.RemoteAddr)
	w.Header().Set("Version", version)
	w.Write([]byte("pong"))
	hp.logger.Debug("pong",
		"remote", r.RemoteAddr)
}

func (hp *httpProxy) pull(w http.ResponseWriter, r *http.Request) {
	if err := hp.before(w, r); err != nil {
		return
	}
	uuid := r.Header.Get("UUID")
	hp.Lock()
	pc, ok := hp.proxyMap[uuid]
	hp.Unlock()
	if !ok {
		hp.logger.Warn("the connection associated with this uuid does not exist",
			"uuid", uuid)
		WriteHTTPError(w, "uuid don't exist")
		return
	}
	if pc.IsClosed() {
		WriteHTTPError(w, "remote conn is closed")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	interval := r.Header.Get("Interval")
	if interval == "" {
		interval = "0"
	}
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)
	t, err := strconv.ParseInt(interval, 10, 0)
	if err != nil {
		hp.logger.Warn("error",
			"interval", interval,
			"msg", err)
	}
	if t > 0 {
		pc.remote.SetReadDeadline(time.Now().Add(time.Duration(t)))
		n, err := pc.remote.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
			} else {
				if err != io.EOF && !pc.IsClosed() {
					hp.logger.Error("error", "msg", err)
				}
				hp.logger.Debug("closing the remote conn",
					"uuid", uuid)
				pc.Close()
			}
		}

		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		hp.logger.Warn("error",
			"msg", "can't convert to http.Flusher")
	}
	w.Header().Set("Transfer-Encoding", "chunked")
	defer pc.Close()
	for {
		flusher.Flush()
		n, err := pc.remote.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			if err != io.EOF && !pc.IsClosed() {
				hp.logger.Error("error", "msg", err)
			}
			return
		}
	}
}

func (hp *httpProxy) push(w http.ResponseWriter, r *http.Request) {
	if err := hp.before(w, r); err != nil {
		return
	}
	uuid := r.Header.Get("UUID")
	hp.Lock()
	pc, ok := hp.proxyMap[uuid]
	hp.Unlock()
	if !ok {
		hp.logger.Warn("the connection associated with this uuid does not exist",
			"uuid", uuid)
		WriteHTTPError(w, "uuid don't exist")
		return
	}
	if pc.IsClosed() {
		WriteHTTPError(w, "remote conn is closed")
		return
	}

	typ := r.Header.Get("TYP")
	switch typ {
	default:
	case HEART_TYP:
		pc.Heart()
	case QUIT_TYP:
		hp.logger.Debug("closing the remote conn",
			"uuid", uuid)
		pc.Close()
	case DATA_TYP:
		_, err := io.Copy(pc.remote, r.Body)
		if err != nil && err != io.EOF {
			if !pc.IsClosed() {
				hp.logger.Error("error", "msg", err)
			}
			hp.logger.Debug("closing the remote conn",
				"uuid", uuid)
			pc.Close()
		}
	}

}

func (hp *httpProxy) connect(w http.ResponseWriter, r *http.Request) {

	if err := hp.before(w, r); err != nil {
		return
	}

	host := r.Header.Get("DSTHOST")
	port := r.Header.Get("DSTPORT")
	addr := fmt.Sprintf("%s:%s", host, port)
	remote, err := net.DialTimeout("tcp", addr, time.Second*timeout)
	if err != nil {
		WriteHTTPError(w, fmt.Sprintf("connect %s %v", addr, err))
		return
	}
	hp.logger.Info("connect success", "addr", addr)
	proxyID := uuid.New().String()
	pc := newProxyConn(remote, proxyID)
	hp.Lock()
	hp.proxyMap[proxyID] = pc
	hp.Unlock()

	go func() {
		pc.Do()
		hp.Lock()
		delete(hp.proxyMap, proxyID)
		hp.Unlock()
		hp.logger.Info("disconnect", "addr", addr)
	}()
	WriteHTTPOK(w, proxyID)
}

func (hp *httpProxy) chunkPush(w http.ResponseWriter, r *http.Request) {
	if err := hp.before(w, r); err != nil {
		return
	}
	chunk := bufPool.Get().([]byte)
	defer bufPool.Put(chunk)
	for {
		n, err := r.Body.Read(chunk)
		if n > 0 {
			// unpack chunk
		}
		if err != nil {
			hp.logger.Error("error while reading chunks", "msg", err)
			break
		}
	}
}

func (hp *httpProxy) chunkPull(w http.ResponseWriter, r *http.Request) {
	if err := hp.before(w, r); err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")
	flusher, _ := w.(http.Flusher)
	flusher.Flush()
	buf := make([]byte, 10)
	for {
		_, err := w.Write(buf)
		if err != nil {
			hp.logger.Error("error while flushing buffer", "msg", err)
			break
		}
		flusher.Flush()
	}
}
