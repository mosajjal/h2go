package h2go

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// handler is maintained for backward compatibility.
// Deprecated: Use Client instead.
type handler struct {
	Server   string
	Secret   string
	Interval time.Duration
	Logger   *slog.Logger
	client   *Client
}

// Ensure handler implements the ProxyHandler interface.
var _ ProxyHandler = (*handler)(nil)

// NewHandler creates a new client handler.
// Deprecated: Use NewClient instead.
func NewHandler(server, secret string, interval time.Duration, logger *slog.Logger) *handler {
	if logger == nil {
		logger = DefaultLogger()
	}

	// Initialize HTTP/2 client if not already done
	if hc.Transport == http.DefaultTransport {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2
		}
		hc = &http.Client{Transport: configureHTTP2Transport(tlsConfig)}
	}

	// Create a new Client with the same configuration
	client := NewClient(
		WithServerURL(server),
		WithSecret(secret),
		WithInterval(interval),
		WithLogger(logger),
		WithHTTPClient(hc),
	)

	return &handler{
		Server:   server,
		Secret:   secret,
		Interval: interval,
		Logger:   logger,
		client:   client,
	}
}

// Connect establishes a connection to the specified address through the proxy server.
func (h *handler) Connect(addr string) (io.ReadWriteCloser, error) {
	return h.client.Connect(addr)
}

// Clean performs any cleanup operations.
func (h *handler) Clean() {
	h.client.Clean()
}
