// Package h2go provides a high-performance HTTP/2 proxy server and client library.
//
// h2go is a proxy solution that leverages HTTP/2 for enhanced performance
// and efficiency, supporting both HTTP and SOCKS5 proxy protocols.
//
// # Library Usage
//
// The package can be used as a library to build custom proxy solutions.
// The main interfaces are:
//
//   - Connector: For establishing proxy connections
//   - Authenticator: For request authentication
//   - HTTPClient: For making HTTP requests
//
// # Client Example
//
//	// Create a client that connects through the proxy
//	client := h2go.NewClient(
//	    h2go.WithServerURL("http://proxy.example.com:8080"),
//	    h2go.WithSecret("my-secret"),
//	)
//
//	// Connect to a target through the proxy
//	conn, err := client.Connect("target.example.com:443")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conn.Close()
//
// # Server Example
//
//	// Create a proxy server
//	server := h2go.NewProxyServer(
//	    h2go.WithListenAddr(":8080"),
//	    h2go.WithServerSecret("my-secret"),
//	)
//
//	// Start the server
//	if err := server.ListenAndServe(); err != nil {
//	    log.Fatal(err)
//	}
package h2go

import (
	"io"
	"net/http"
)

// Connector defines the interface for establishing proxy connections.
// Implementations of this interface are responsible for connecting to
// remote hosts through the proxy server.
type Connector interface {
	// Connect establishes a connection to the specified address through
	// the proxy server. The address should be in "host:port" format.
	// The returned io.ReadWriteCloser can be used for bidirectional
	// communication with the remote host.
	Connect(addr string) (io.ReadWriteCloser, error)
}

// Authenticator defines the interface for request authentication.
// Implementations should provide methods for generating and verifying
// authentication signatures.
type Authenticator interface {
	// Sign generates an authentication signature for the given data.
	Sign(data string) string

	// Verify checks if the provided signature is valid for the given data.
	Verify(data, signature string) bool
}

// HTTPClient defines the interface for making HTTP requests.
// This allows for dependency injection of custom HTTP clients
// for testing or specialized transport requirements.
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// ProxyHandler defines the interface for handling proxy connections.
// This is used by the local proxy server to forward connections.
type ProxyHandler interface {
	// Connect establishes a connection to the specified address.
	Connect(addr string) (io.ReadWriteCloser, error)

	// Clean performs any cleanup operations.
	Clean()
}
