package h2go

import (
	"net"
	"sync"
	"time"
)

// proxyConn represents a proxy connection to a remote host.
// It manages the lifecycle of the connection including heartbeat handling.
type proxyConn struct {
	remote    net.Conn
	uuid      string
	close     chan struct{}
	heart     chan struct{}
	mu        sync.Mutex
	hasClosed bool
}

// newProxyConn creates a new proxy connection.
func newProxyConn(remote net.Conn, uuid string) *proxyConn {
	return &proxyConn{remote: remote, uuid: uuid,
		close: make(chan struct{}),
		heart: make(chan struct{}),
	}
}

// Close closes the proxy connection.
func (pc *proxyConn) Close() {
	pc.mu.Lock()
	pc.hasClosed = true
	pc.mu.Unlock()
	select {
	case pc.close <- struct{}{}:
	default:
	}
}

// IsClosed returns whether the connection is closed.
func (pc *proxyConn) IsClosed() bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.hasClosed
}

// Heart sends a heartbeat signal to keep the connection alive.
func (pc *proxyConn) Heart() {
	select {
	case pc.heart <- struct{}{}:
	default:
	}
}

// Do runs the connection lifecycle, waiting for close or heartbeat timeout.
func (pc *proxyConn) Do() {
	defer pc.remote.Close()

	for {
		select {
		case <-time.After(time.Second * heartTTL):
			return
		case <-pc.close:
			return
		case <-pc.heart:
			continue
		}
	}
}
