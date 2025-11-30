package h2go

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Client represents an HTTP/2 proxy client that can establish connections
// through a remote proxy server. It implements the Connector interface.
type Client struct {
	serverURL     string
	secret        string
	interval      time.Duration
	logger        *slog.Logger
	httpClient    HTTPClient
	authenticator Authenticator
}

// Ensure Client implements the Connector and ProxyHandler interfaces.
var (
	_ Connector    = (*Client)(nil)
	_ ProxyHandler = (*Client)(nil)
)

// NewClient creates a new proxy client with the given options.
// The client uses HTTP/2 for communication with the proxy server.
//
// Example:
//
//	client := h2go.NewClient(
//	    h2go.WithServerURL("http://proxy.example.com:8080"),
//	    h2go.WithSecret("my-secret"),
//	)
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		logger: DefaultLogger(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Set default HTTP client if not provided
	if c.httpClient == nil {
		c.httpClient = newDefaultHTTPClient()
	}

	// Set default authenticator if not provided
	if c.authenticator == nil {
		c.authenticator = NewHMACAuthenticator(c.secret)
	}

	return c
}

// Connect establishes a connection to the specified address through the proxy server.
// The address should be in "host:port" format.
// Returns an io.ReadWriteCloser that can be used for bidirectional communication.
func (c *Client) Connect(addr string) (io.ReadWriteCloser, error) {
	serverURL := strings.TrimSuffix(c.serverURL, "/")

	conn := newClientConnection(
		serverURL,
		c.secret,
		c.interval,
		c.logger,
		c.httpClient,
		c.authenticator,
	)

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid address format: %s", addr)
	}
	host, port := parts[0], parts[1]

	uuid, err := conn.connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	conn.uuid = uuid

	if c.interval == 0 {
		if err := conn.pull(); err != nil {
			return nil, err
		}
	}

	conn.close = make(chan bool)
	go conn.alive()

	return conn, nil
}

// Clean performs any cleanup operations.
// Currently a no-op but defined to satisfy the ProxyHandler interface.
func (c *Client) Clean() {}

// ServerURL returns the configured server URL.
func (c *Client) ServerURL() string {
	return c.serverURL
}

// newDefaultHTTPClient creates a new HTTP client configured for HTTP/2.
func newDefaultHTTPClient() *http.Client {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"},
	}
	return &http.Client{Transport: configureHTTP2Transport(tlsConfig)}
}

// SetHTTPClient allows setting a custom HTTP client.
// This is useful for configuring TLS settings after client creation.
func (c *Client) SetHTTPClient(client HTTPClient) {
	c.httpClient = client
}
