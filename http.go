package h2go

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Endpoint paths for the proxy server.
const (
	CONNECT    = "/connect"
	PING       = "/ping"
	PULL       = "/pull"
	PUSH       = "/push"
	DOWNLOAD   = "/download"
	CHUNK_PULL = "/chunk_pull"
	CHUNK_PUSH = "/chunk_push"
)

// Message types for the proxy protocol.
const (
	DATA_TYP  = "data"
	QUIT_TYP  = "quit"
	HEART_TYP = "heart"
)

// Protocol constants.
const (
	timeout  = 10
	signTTL  = 10
	heartTTL = 60
)

const (
	version = "20170803"
)

// DevZero is an io.Reader that returns zeros.
type DevZero struct{}

// Read fills b with zeros and returns len(b).
func (z DevZero) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}
	return len(b), nil
}

var bufPool = &sync.Pool{New: func() interface{} { return make([]byte, 1024*8) }}

// ProxyServer is an HTTP/2 proxy server that handles proxy requests.
// It supports both HTTP and HTTPS modes.
type ProxyServer struct {
	addr          string
	secret        string
	proxyMap      map[string]*proxyConn
	mu            sync.Mutex
	https         bool
	logger        *slog.Logger
	authenticator Authenticator
	certPath      string
	keyPath       string
	mux           *http.ServeMux
}

// NewProxyServer creates a new proxy server with the given options.
//
// Example:
//
//	server := h2go.NewProxyServer(
//	    h2go.WithListenAddr(":8080"),
//	    h2go.WithServerSecret("my-secret"),
//	)
func NewProxyServer(opts ...ServerOption) *ProxyServer {
	s := &ProxyServer{
		proxyMap: make(map[string]*proxyConn),
		logger:   DefaultLogger(),
		mux:      http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set default authenticator if not provided
	if s.authenticator == nil {
		s.authenticator = NewHMACAuthenticator(s.secret)
	}

	return s
}

// ListenAndServe starts the proxy server.
// If HTTPS is enabled, it uses TLS; otherwise, it uses h2c (HTTP/2 cleartext).
func (s *ProxyServer) ListenAndServe() error {
	s.registerHandlers()

	if s.https {
		return s.listenHTTPS()
	}
	return s.listen()
}

func (s *ProxyServer) registerHandlers() {
	s.mux.HandleFunc(CONNECT, s.handleConnect)
	s.mux.HandleFunc(PULL, s.handlePull)
	s.mux.HandleFunc(PUSH, s.handlePush)
	s.mux.HandleFunc(PING, s.handlePing)
	s.mux.HandleFunc(CHUNK_PULL, s.handleChunkPull)
	s.mux.HandleFunc(CHUNK_PUSH, s.handleChunkPush)
}

func (s *ProxyServer) listenHTTPS() error {
	s.logger.Info("starting the https/http2 server",
		"addr", s.addr)

	// Create HTTP/2 server with TLS
	server := &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		},
	}

	// Configure HTTP/2
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		return fmt.Errorf("error configuring http2: %w", err)
	}

	return server.ListenAndServeTLS(s.certPath, s.keyPath)
}

func (s *ProxyServer) listen() error {
	s.logger.Info("starting the http/http2 server (h2c)",
		"addr", s.addr)

	// Create HTTP/2 server without TLS (h2c - HTTP/2 cleartext)
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:    s.addr,
		Handler: h2c.NewHandler(s.mux, h2s),
	}

	return server.ListenAndServe()
}

func (s *ProxyServer) verify(r *http.Request) error {
	ts := r.Header.Get("timestamp")
	if ts == "" {
		return errors.New("timestamp is empty")
	}
	sign := r.Header.Get("sign")
	tm, err := strconv.ParseInt(ts, 10, 0)
	if err != nil {
		return fmt.Errorf("timestamp invalid: %w", err)
	}
	now := time.Now().Unix()
	if now-tm > signTTL {
		return errors.New("timestamp expire")
	}
	if s.authenticator.Verify(ts, sign) {
		return nil
	}
	return errors.New("sign invalid")
}

func (s *ProxyServer) before(w http.ResponseWriter, r *http.Request) error {
	err := s.verify(r)
	if err != nil {
		s.logger.Warn("error while verifying the request",
			"msg", err)
		WriteNotFoundError(w, "404")
	}
	return err
}

func (s *ProxyServer) handlePing(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("ping",
		"remote", r.RemoteAddr)
	w.Header().Set("Version", version)
	w.Write([]byte("pong"))
	s.logger.Debug("pong",
		"remote", r.RemoteAddr)
}

func (s *ProxyServer) handlePull(w http.ResponseWriter, r *http.Request) {
	if err := s.before(w, r); err != nil {
		return
	}
	uuid := r.Header.Get("UUID")
	s.mu.Lock()
	pc, ok := s.proxyMap[uuid]
	s.mu.Unlock()
	if !ok {
		s.logger.Warn("the connection associated with this uuid does not exist",
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
		s.logger.Warn("error",
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
					s.logger.Error("error", "msg", err)
				}
				s.logger.Debug("closing the remote conn",
					"uuid", uuid)
				pc.Close()
			}
		}

		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logger.Warn("error",
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
				s.logger.Error("error", "msg", err)
			}
			return
		}
	}
}

func (s *ProxyServer) handlePush(w http.ResponseWriter, r *http.Request) {
	if err := s.before(w, r); err != nil {
		return
	}
	uuid := r.Header.Get("UUID")
	s.mu.Lock()
	pc, ok := s.proxyMap[uuid]
	s.mu.Unlock()
	if !ok {
		s.logger.Warn("the connection associated with this uuid does not exist",
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
		s.logger.Debug("closing the remote conn",
			"uuid", uuid)
		pc.Close()
	case DATA_TYP:
		_, err := io.Copy(pc.remote, r.Body)
		if err != nil && err != io.EOF {
			if !pc.IsClosed() {
				s.logger.Error("error", "msg", err)
			}
			s.logger.Debug("closing the remote conn",
				"uuid", uuid)
			pc.Close()
		}
	}
}

func (s *ProxyServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	if err := s.before(w, r); err != nil {
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
	s.logger.Info("connect success", "addr", addr)
	proxyID := uuid.New().String()
	pc := newProxyConn(remote, proxyID)
	s.mu.Lock()
	s.proxyMap[proxyID] = pc
	s.mu.Unlock()

	go func() {
		pc.Do()
		s.mu.Lock()
		delete(s.proxyMap, proxyID)
		s.mu.Unlock()
		s.logger.Info("disconnect", "addr", addr)
	}()
	WriteHTTPOK(w, proxyID)
}

func (s *ProxyServer) handleChunkPush(w http.ResponseWriter, r *http.Request) {
	if err := s.before(w, r); err != nil {
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
			s.logger.Error("error while reading chunks", "msg", err)
			break
		}
	}
}

func (s *ProxyServer) handleChunkPull(w http.ResponseWriter, r *http.Request) {
	if err := s.before(w, r); err != nil {
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
			s.logger.Error("error while flushing buffer", "msg", err)
			break
		}
		flusher.Flush()
	}
}

// download handles download requests.
func (s *ProxyServer) download(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Length", fmt.Sprintf("%d", 100<<20))
	io.CopyN(w, DevZero{}, 100<<20)
}

// Legacy types and functions for backward compatibility

// httpProxy is an alias for ProxyServer for backward compatibility.
// Deprecated: Use ProxyServer instead.
type httpProxy = ProxyServer

// NewHttpProxy creates a new proxy server for backward compatibility.
// Deprecated: Use NewProxyServer instead.
func NewHttpProxy(logger *slog.Logger, addr, secret string, https bool) *httpProxy {
	return NewProxyServer(
		WithListenAddr(addr),
		WithServerSecret(secret),
		WithHTTPS(https),
		WithServerLogger(logger),
	)
}

// ListenHTTPS starts the server in HTTPS mode.
// Deprecated: Use ListenAndServe with WithHTTPS option instead.
func (s *ProxyServer) ListenHTTPS(cert, key string) {
	s.certPath = cert
	s.keyPath = key
	s.https = true
	s.registerHandlersLegacy()
	s.logger.Info("starting the https/http2 server",
		"addr", s.addr)

	// Create HTTP/2 server with TLS
	server := &http.Server{
		Addr:    s.addr,
		Handler: nil,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		},
	}

	// Configure HTTP/2
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		s.logger.Error("error configuring http2", "msg", err)
		return
	}

	s.logger.Error("error", "msg", server.ListenAndServeTLS(cert, key))
}

// Listen starts the server in HTTP mode (h2c).
// Deprecated: Use ListenAndServe instead.
func (s *ProxyServer) Listen() {
	s.registerHandlersLegacy()
	s.logger.Info("starting the http/http2 server (h2c)",
		"addr", s.addr)

	// Create HTTP/2 server without TLS (h2c - HTTP/2 cleartext)
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:    s.addr,
		Handler: h2c.NewHandler(http.DefaultServeMux, h2s),
	}

	s.logger.Error("error", "msg", server.ListenAndServe())
}

// registerHandlersLegacy registers handlers on http.DefaultServeMux for backward compatibility.
func (s *ProxyServer) registerHandlersLegacy() {
	http.HandleFunc(CONNECT, s.handleConnect)
	http.HandleFunc(PULL, s.handlePull)
	http.HandleFunc(PUSH, s.handlePush)
	http.HandleFunc(PING, s.handlePing)
	http.HandleFunc(CHUNK_PULL, s.handleChunkPull)
	http.HandleFunc(CHUNK_PUSH, s.handleChunkPush)
}
