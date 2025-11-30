package h2go

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
)

// SOCKS5 address types.
const (
	typeIPv4 = 1 // type is ipv4 address
	typeDm   = 3 // type is domain address
	typeIPv6 = 4 // type is ipv6 address
)

// Common errors for proxy handling.
var (
	ErrNotSupportedProtocol = errors.New("protocol not supported")
	ErrNotSupportedNow      = errors.New("not supported now")
	ErrAuthExtraData        = errors.New("socks authentication get extra data")
	ErrCommand              = errors.New("socks command not supported")
	ErrAddrType             = errors.New("socks addr type not supported")
	ErrVersion              = errors.New("socks version not supported")
	ErrReqExtraData         = errors.New("socks request get extra data")
)

// Legacy error variables for backward compatibility.
// Deprecated: Use the exported error variables instead.
var (
	errNotSupportProtocol = ErrNotSupportedProtocol
	errNotSupportNow      = ErrNotSupportedNow
	errAuthExtraData      = ErrAuthExtraData
	errCmd                = ErrCommand
	errAddrType           = ErrAddrType
	errVer                = ErrVersion
	errReqExtraData       = ErrReqExtraData
)

type reqReader struct {
	b []byte
	r io.Reader
}

func (r *reqReader) Read(p []byte) (n int, err error) {
	if len(r.b) == 0 {
		return r.r.Read(p)
	}
	n = copy(p, r.b)
	r.b = r.b[n:]

	return
}

// LocalServer is a local SOCKS5/HTTP proxy server that forwards connections
// through a remote proxy server. It accepts connections from local applications
// and forwards them to the configured proxy handlers.
type LocalServer struct {
	// Addr is the local address to listen on.
	Addr string

	// Socks5Handler handles SOCKS5 proxy requests.
	Socks5Handler ProxyHandler

	// HTTPHandler handles HTTP proxy requests.
	HTTPHandler ProxyHandler

	// DisableSocks5 disables SOCKS5 proxy support.
	DisableSocks5 bool

	// DisableHTTP disables HTTP proxy support.
	DisableHTTP bool

	// DisableHTTPCONNECT disables HTTP CONNECT method support.
	DisableHTTPCONNECT bool

	// Logger is the logger for the server.
	Logger *slog.Logger
}

// NewLocalServer creates a new local proxy server with the given options.
//
// Example:
//
//	client := h2go.NewClient(
//	    h2go.WithServerURL("http://proxy.example.com:8080"),
//	    h2go.WithSecret("my-secret"),
//	)
//
//	server := h2go.NewLocalServer(
//	    h2go.WithLocalListenAddr("127.0.0.1:1080"),
//	    h2go.WithSocks5Handler(client),
//	    h2go.WithHTTPHandler(client),
//	)
func NewLocalServer(opts ...LocalServerOption) *LocalServer {
	s := &LocalServer{
		Logger: DefaultLogger(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *LocalServer) handleConn(conn net.Conn) (err error) {

	defer conn.Close()
	var (
		conn2 io.ReadWriteCloser
		n     int
	)

	buf := make([]byte, 258)
	n, err = io.ReadAtLeast(conn, buf, 2)
	if err != nil {
		return err
	}
	if buf[0] == 0x05 {
		if s.DisableSocks5 || (s.Socks5Handler == nil) {
			return ErrNotSupportedProtocol
		}
		nmethod := int(buf[1])
		msgLen := nmethod + 2
		if n == msgLen {
			// common case
		} else if n < msgLen {
			if _, err = io.ReadFull(conn, buf[n:msgLen]); err != nil {
				return
			}
		} else {
			return ErrAuthExtraData
		}
		// send confirmation: version 5, no authentication required
		if _, err = conn.Write([]byte{0x05, 0x00}); err != nil {
			return
		}

		buf := make([]byte, 263)
		if n, err = io.ReadAtLeast(conn, buf, 5); err != nil {
			return
		}
		if buf[0] != 0x05 {
			return ErrVersion
		}
		if buf[1] != 0x01 {
			return ErrCommand
		}
		reqLen := -1
		var (
			addr string
			host string
		)
		switch buf[3] {
		case typeIPv4:
			reqLen = net.IPv4len + 6
		case typeIPv6:
			reqLen = net.IPv6len + 6
		case typeDm:
			reqLen = int(buf[4]) + 7
		default:
			return ErrAddrType
		}
		if n == reqLen {
			// common case, do nothing
		} else if n < reqLen { // rare case
			if _, err = io.ReadFull(conn, buf[n:reqLen]); err != nil {
				return
			}
		} else {
			return ErrReqExtraData
		}
		switch buf[3] {
		case typeIPv4:
			host = net.IP(buf[4 : 4+net.IPv4len]).String()
		case typeIPv6:
			host = net.IP(buf[4 : 4+net.IPv6len]).String()
		case typeDm:
			host = string(buf[5 : 5+buf[4]])
		}
		port := binary.BigEndian.Uint16(buf[reqLen-2 : reqLen])
		addr = net.JoinHostPort(host, strconv.Itoa(int(port)))
		s.Logger.Info("socks5",
			"addr", addr)
		conn2, err = s.Socks5Handler.Connect(addr)
		if err != nil {
			return
		}
		conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
		s.Logger.Info("socks5",
			"local", conn.RemoteAddr().String(),
			"remote", addr)

		defer s.Socks5Handler.Clean()
	} else {
		if s.DisableHTTP || (s.HTTPHandler == nil) {
			return ErrNotSupportedProtocol
		}

		req, err := http.ReadRequest(bufio.NewReader(&reqReader{b: buf[:n], r: conn}))
		if err != nil {
			return err
		}
		s.Logger.Info("http",
			"method", req.Method,
			"remote", conn.RemoteAddr().String(),
			"host", req.Host,
			"proto", req.Proto)

		if req.Method == "CONNECT" && s.DisableHTTPCONNECT {
			conn.Write([]byte("HTTP/1.1 502 Connection refused\r\n\r\n"))
			return ErrNotSupportedProtocol
		}

		if s.Logger.Enabled(context.Background(), slog.LevelDebug) {
			dump, _ := httputil.DumpRequest(req, false)
			s.Logger.Debug("http", "dump", string(dump))
		}

		if req.Method == "PRI" && req.ProtoMajor == 2 {
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return ErrNotSupportedNow
		}
		addr := req.Host
		if !strings.Contains(addr, ":") {
			addr += ":80"
		}
		conn2, err = s.HTTPHandler.Connect(addr)
		if err != nil {
			return err
		}
		if req.Method == "CONNECT" {
			conn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		} else {
			// bug here
			req.Header.Del("Proxy-Connection")
			req.Header.Set("Connection", "Keep-Alive")
			req.Write(conn2)
		}
		s.Logger.Info("http",
			"local", conn.RemoteAddr().String(),
			"remote", addr)
		defer s.HTTPHandler.Clean()
	}
	defer conn2.Close()
	return s.transport(conn, conn2)
}

func (s *LocalServer) transport(conn1 io.ReadWriter, conn2 io.ReadWriter) (err error) {
	errChan := make(chan error, 2)

	go func() {
		_, err := io.Copy(conn1, conn2)
		if err != nil {
			s.Logger.Error("copy", "msg", err)
		}
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(conn2, conn1)
		if err != nil {
			s.Logger.Error("copy", "msg", err)
		}
		errChan <- err
	}()
	err = <-errChan
	return
}

// ListenAndServe starts the local proxy server.
// It accepts both SOCKS5 and HTTP proxy connections.
func (s *LocalServer) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	if s.Logger == nil {
		s.Logger = DefaultLogger()
	}
	s.Logger.Info("socks5/http proxy started",
		"addr", l.Addr().(*net.TCPAddr).String())
	for {
		if conn, err := l.Accept(); err == nil {
			go func() {
				if err := s.handleConn(conn); err != nil {
					s.Logger.Error("handle conn",
						"from", conn.RemoteAddr().String(),
						"msg", err)
				}
			}()
		} else {
			s.Logger.Error("accept", "msg", err)
		}

	}
}

// Server is an alias for LocalServer for backward compatibility.
// Deprecated: Use LocalServer instead.
type Server = LocalServer
