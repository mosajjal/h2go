package h2go

import (
	"log/slog"
	"time"
)

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithServerURL sets the remote proxy server URL.
// The URL should include the scheme (http:// or https://).
func WithServerURL(url string) ClientOption {
	return func(c *Client) {
		c.serverURL = url
	}
}

// WithSecret sets the shared secret for authentication.
func WithSecret(secret string) ClientOption {
	return func(c *Client) {
		c.secret = secret
	}
}

// WithInterval sets the polling interval for data retrieval.
// A value of 0 means use HTTP chunked transfer encoding.
func WithInterval(interval time.Duration) ClientOption {
	return func(c *Client) {
		c.interval = interval
	}
}

// WithLogger sets a custom logger for the client.
// If nil is provided, the default logger will be used.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithHTTPClient sets a custom HTTP client for making requests.
// This is useful for testing or when custom transport settings are needed.
func WithHTTPClient(client HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithAuthenticator sets a custom authenticator for request signing.
func WithAuthenticator(auth Authenticator) ClientOption {
	return func(c *Client) {
		c.authenticator = auth
	}
}

// ServerOption is a function that configures a ProxyServer.
type ServerOption func(*ProxyServer)

// WithListenAddr sets the address for the server to listen on.
func WithListenAddr(addr string) ServerOption {
	return func(s *ProxyServer) {
		s.addr = addr
	}
}

// WithServerSecret sets the shared secret for authentication.
func WithServerSecret(secret string) ServerOption {
	return func(s *ProxyServer) {
		s.secret = secret
	}
}

// WithHTTPS enables HTTPS mode for the server.
func WithHTTPS(enabled bool) ServerOption {
	return func(s *ProxyServer) {
		s.https = enabled
	}
}

// WithTLSCert sets the path to the TLS certificate file.
func WithTLSCert(certPath string) ServerOption {
	return func(s *ProxyServer) {
		s.certPath = certPath
	}
}

// WithTLSKey sets the path to the TLS private key file.
func WithTLSKey(keyPath string) ServerOption {
	return func(s *ProxyServer) {
		s.keyPath = keyPath
	}
}

// WithServerLogger sets a custom logger for the server.
// If nil is provided, the default logger will be used.
func WithServerLogger(logger *slog.Logger) ServerOption {
	return func(s *ProxyServer) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// WithServerAuthenticator sets a custom authenticator for request verification.
func WithServerAuthenticator(auth Authenticator) ServerOption {
	return func(s *ProxyServer) {
		s.authenticator = auth
	}
}

// LocalServerOption is a function that configures a LocalServer.
type LocalServerOption func(*LocalServer)

// WithLocalListenAddr sets the address for the local proxy to listen on.
func WithLocalListenAddr(addr string) LocalServerOption {
	return func(s *LocalServer) {
		s.Addr = addr
	}
}

// WithSocks5Handler sets the handler for SOCKS5 proxy requests.
func WithSocks5Handler(handler ProxyHandler) LocalServerOption {
	return func(s *LocalServer) {
		s.Socks5Handler = handler
	}
}

// WithHTTPHandler sets the handler for HTTP proxy requests.
func WithHTTPHandler(handler ProxyHandler) LocalServerOption {
	return func(s *LocalServer) {
		s.HTTPHandler = handler
	}
}

// WithDisableSocks5 disables SOCKS5 proxy support.
func WithDisableSocks5(disabled bool) LocalServerOption {
	return func(s *LocalServer) {
		s.DisableSocks5 = disabled
	}
}

// WithDisableHTTP disables HTTP proxy support.
func WithDisableHTTP(disabled bool) LocalServerOption {
	return func(s *LocalServer) {
		s.DisableHTTP = disabled
	}
}

// WithDisableHTTPConnect disables HTTP CONNECT method support.
func WithDisableHTTPConnect(disabled bool) LocalServerOption {
	return func(s *LocalServer) {
		s.DisableHTTPCONNECT = disabled
	}
}

// WithLocalLogger sets a custom logger for the local server.
// If nil is provided, the default logger will be used.
func WithLocalLogger(logger *slog.Logger) LocalServerOption {
	return func(s *LocalServer) {
		if logger != nil {
			s.Logger = logger
		}
	}
}
