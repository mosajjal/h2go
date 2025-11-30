# h2go

> A high-performance HTTP/2 proxy server and client library built in Go

> Note: this is heavily based on the awesome work of [ls0f](https://github.com/ls0f/cracker). Only copied here to remove some 3rd party libraries

h2go is a proxy solution that leverages HTTP/2 for enhanced performance and efficiency, supporting both HTTP and SOCKS5 proxy protocols. It can be used as both a **command-line tool** and a **Go library**.

## Features

- **Native HTTP/2 Support**: Uses HTTP/2 for both client-server communication with automatic fallback to HTTP/1.1
- **h2c Support**: HTTP/2 cleartext mode for non-TLS connections
- **Dual Protocol**: Supports both HTTP and SOCKS5 proxy protocols
- **Secure Communication**: Optional HTTPS/TLS support with custom certificates
- **HMAC Authentication**: Built-in HMAC-SHA1 authentication for secure connections
- **High Performance**: Optimized for concurrent connections and low latency
- **Library Support**: Clean interfaces and dependency injection for embedding in your own applications

```
+-----1------+             +-------2-------+          
| client app |  <=======>  |  local proxy  |  <#####
+------------+             +---------------+       #
                                                   #
                                                   #
                                                   # HTTP/2 over http[s]
                                                   #
                                                   #
+------4------+             +-------3------+       #
| target host |  <=======>  |http[s] server|  <#####
+-------------+             +--------------+         
```

## Performance

h2go leverages HTTP/2 features for improved performance:

- **Multiplexing**: Multiple requests over a single connection
- **Header Compression**: HPACK compression reduces overhead
- **Binary Protocol**: More efficient than HTTP/1.1 text-based protocol
- **Server Push**: Optional server-initiated data transfer

Run benchmarks to see performance metrics:
```bash
go test -bench=. -benchmem
```

# Install

Download the latest binaries from this [release page](https://github.com/mosajjal/h2go/releases).

Or build from source:
```bash
go install github.com/mosajjal/h2go/cmd/h2go@latest
```

# Command Line Usage

## Server side (Run on your vps or other application container platform)

Start an HTTP/2 server (h2c - cleartext HTTP/2):
```
./h2go server --addr :8080 --secret <password>
```

The server will automatically use HTTP/2 protocol (h2c for non-TLS connections).

## Client side (Run on your local pc)

Connect to the HTTP/2 proxy server:
```
./h2go client --raddr http://example.com:8080 --secret <password>
```

The client will automatically use HTTP/2 when connecting to the server.

## https

It is strongly recommended to enable HTTPS on the server side for production use. With HTTPS, the connection will use HTTP/2 over TLS (h2).

### Notice

If you have an SSL certificate, it's easy to enable HTTPS with HTTP/2:

```
./h2go server --addr :443 --secret <password> --https --cert /etc/cert.pem --key /etc/key.pem
```

Client connection with HTTPS (uses HTTP/2 over TLS):
```
./h2go client --raddr https://example.com --secret <password>
```

You can also generate self-signed SSL certificates using the gencert command:

```
./h2go gencert --domain example.com # you can also use an IP instead of a domain, or provide multiple --domain flags
```

Server with self-signed certificate (HTTP/2 over TLS):
```
./h2go server --addr :443 --secret <password> --https --cert /etc/self-signed-cert.pem --key /etc/self-ca-key.pem
```

Client with self-signed certificate (HTTP/2 over TLS):
```
./h2go client --raddr https://example.com --secret <password> --cert /etc/self-signed-cert.pem
```

# Library Usage

h2go can be used as a Go library for embedding proxy functionality in your applications. The library provides clean interfaces and uses the functional options pattern for flexible configuration.

## Interfaces

The library defines the following interfaces for dependency injection:

- `Connector`: For establishing proxy connections
- `ProxyHandler`: For handling proxy connections (extends Connector with Clean method)
- `Authenticator`: For request authentication
- `HTTPClient`: For making HTTP requests

## Client Example

Create a client that connects through the remote proxy:

```go
package main

import (
    "io"
    "log"
    "time"
    
    "github.com/mosajjal/h2go"
)

func main() {
    // Create a client with options
    client := h2go.NewClient(
        h2go.WithServerURL("http://proxy.example.com:8080"),
        h2go.WithSecret("my-secret"),
        h2go.WithInterval(time.Millisecond * 50), // Polling interval (0 = chunked transfer)
    )
    
    // Connect to a target through the proxy
    conn, err := client.Connect("target.example.com:443")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Use the connection for bidirectional communication
    _, err = conn.Write([]byte("Hello, World!"))
    if err != nil {
        log.Fatal(err)
    }
    
    buf := make([]byte, 1024)
    n, err := conn.Read(buf)
    if err != nil && err != io.EOF {
        log.Fatal(err)
    }
    log.Printf("Received: %s", buf[:n])
}
```

## Server Example

Create a proxy server:

```go
package main

import (
    "log"
    
    "github.com/mosajjal/h2go"
)

func main() {
    // Create a proxy server
    server := h2go.NewProxyServer(
        h2go.WithListenAddr(":8080"),
        h2go.WithServerSecret("my-secret"),
    )
    
    // Start the server (blocks)
    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

## Server with HTTPS

```go
package main

import (
    "log"
    
    "github.com/mosajjal/h2go"
)

func main() {
    server := h2go.NewProxyServer(
        h2go.WithListenAddr(":443"),
        h2go.WithServerSecret("my-secret"),
        h2go.WithHTTPS(true),
        h2go.WithTLSCert("/path/to/cert.pem"),
        h2go.WithTLSKey("/path/to/key.pem"),
    )
    
    if err := server.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

## Local Proxy Server Example

Create a local SOCKS5/HTTP proxy that forwards through the remote server:

```go
package main

import (
    "log"
    
    "github.com/mosajjal/h2go"
)

func main() {
    // Create a client that connects to the remote proxy
    client := h2go.NewClient(
        h2go.WithServerURL("http://proxy.example.com:8080"),
        h2go.WithSecret("my-secret"),
    )
    
    // Create a local server that forwards to the remote proxy
    localServer := h2go.NewLocalServer(
        h2go.WithLocalListenAddr("127.0.0.1:1080"),
        h2go.WithSocks5Handler(client),
        h2go.WithHTTPHandler(client),
    )
    
    // Start the local proxy (blocks)
    // Now configure your applications to use localhost:1080 as SOCKS5 or HTTP proxy
    if err := localServer.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

## Custom HTTP Client

Use a custom HTTP client with TLS certificate:

```go
package main

import (
    "log"
    
    "github.com/mosajjal/h2go"
)

func main() {
    // Create HTTP client with custom certificate
    httpClient, err := h2go.NewHTTPClientWithCert("/path/to/cert.pem", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the custom HTTP client
    client := h2go.NewClient(
        h2go.WithServerURL("https://proxy.example.com:443"),
        h2go.WithSecret("my-secret"),
        h2go.WithHTTPClient(httpClient),
    )
    
    conn, err := client.Connect("target.example.com:443")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Use the connection...
}
```

## Custom Authenticator

Implement a custom authenticator:

```go
package main

import (
    "github.com/mosajjal/h2go"
)

// CustomAuthenticator implements the Authenticator interface
type CustomAuthenticator struct {
    secret string
}

func (a *CustomAuthenticator) Sign(data string) string {
    // Your custom signing logic
    return data + "-signed-with-" + a.secret
}

func (a *CustomAuthenticator) Verify(data, signature string) bool {
    // Your custom verification logic
    return a.Sign(data) == signature
}

func main() {
    auth := &CustomAuthenticator{secret: "my-secret"}
    
    client := h2go.NewClient(
        h2go.WithServerURL("http://proxy.example.com:8080"),
        h2go.WithAuthenticator(auth),
    )
    
    // Use the client...
    _ = client
}
```

# Testing

Run all tests including HTTP/2 specific tests:
```bash
go test -v ./...
```

Run performance benchmarks:
```bash
go test -bench=. -benchmem
```

## HTTP/2 Protocol Details

- **Server mode**: Uses h2c (HTTP/2 cleartext) for plain HTTP and h2 (HTTP/2 over TLS) for HTTPS
- **Client mode**: Automatically negotiates HTTP/2 with ALPN when using TLS, falls back to HTTP/1.1 if needed
- **Multiplexing**: Multiple proxy connections can share the same HTTP/2 connection
- **Performance**: HTTP/2's binary framing and compression provide better performance than HTTP/1.1

## Backward Compatibility

h2go maintains backward compatibility with HTTP/1.1 clients and servers. The protocol negotiation happens automatically:
- TLS connections use ALPN to negotiate HTTP/2
- Non-TLS connections attempt h2c upgrade
- Fallback to HTTP/1.1 if HTTP/2 is not supported

The library also maintains backward compatibility with previous versions through type aliases (e.g., `Server` for `LocalServer`, `NewHttpProxy` for `NewProxyServer`).
