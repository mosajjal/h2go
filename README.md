# h2go

> A high-performance HTTP/2 proxy server and client built in Go

> Note: this is heavily based on the awesome work of [ls0f](https://github.com/ls0f/cracker). Only copied here to remove some 3rd party libraries

h2go is a proxy solution that leverages HTTP/2 for enhanced performance and efficiency, supporting both HTTP and SOCKS5 proxy protocols.

## Features

- **Native HTTP/2 Support**: Uses HTTP/2 for both client-server communication with automatic fallback to HTTP/1.1
- **h2c Support**: HTTP/2 cleartext mode for non-TLS connections
- **Dual Protocol**: Supports both HTTP and SOCKS5 proxy protocols
- **Secure Communication**: Optional HTTPS/TLS support with custom certificates
- **HMAC Authentication**: Built-in HMAC-SHA1 authentication for secure connections
- **High Performance**: Optimized for concurrent connections and low latency

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

# Usage

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

## Testing

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


