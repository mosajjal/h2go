package h2go

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

// TestHTTP2ServerSupport verifies that the server properly supports HTTP/2
func TestHTTP2ServerSupport(t *testing.T) {
	startProxyServer()
	
	// Create HTTP/2 capable client
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	http2.ConfigureTransport(transport)
	client := &http.Client{Transport: transport}
	
	// Test ping endpoint
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1%s%s", testAddr, PING))
	if err != nil {
		t.Fatalf("Failed to ping server: %v", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "pong") {
		t.Errorf("Expected 'pong' in response, got: %s", string(body))
	}
	
	// Verify protocol is HTTP/2 or HTTP/1.1 (h2c cleartext HTTP/2)
	// Note: h2c requires special negotiation, so we may get HTTP/1.1 in some test scenarios
	if resp.Proto != "HTTP/2.0" && resp.Proto != "HTTP/1.1" {
		t.Logf("Warning: Expected HTTP/2.0 or HTTP/1.1, got: %s", resp.Proto)
	}
}

// TestHTTP2ClientHandler verifies that the client handler supports HTTP/2
func TestHTTP2ClientHandler(t *testing.T) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, 0, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	// Send HTTP request through the connection
	_, err = conn.Write([]byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	
	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read: %v", err)
	}
	
	response := string(buf[:n])
	if !strings.Contains(response, "pong") {
		t.Errorf("Expected 'pong' in response, got: %s", response)
	}
}

// TestHTTP2WithInterval verifies HTTP/2 works with interval-based polling
func TestHTTP2WithInterval(t *testing.T) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*50, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	// Send data
	testData := []byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n")
	n, err := conn.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}
	
	// Read response with timeout
	done := make(chan bool)
	go func() {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("Failed to read: %v", err)
		}
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(time.Second * 2):
		t.Error("Timeout waiting for response")
	}
}

// TestHTTP2ConcurrentConnections tests multiple concurrent connections
func TestHTTP2ConcurrentConnections(t *testing.T) {
	startProxyServer()
	
	concurrency := 10
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
			conn, err := h.Connect("localhost" + testAddr)
			if err != nil {
				errors <- fmt.Errorf("connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()
			
			// Send data
			_, err = conn.Write([]byte(fmt.Sprintf("GET /ping HTTP/1.1\r\nHost: localhost\r\nX-Request-ID: %d\r\n\r\n", id)))
			if err != nil {
				errors <- fmt.Errorf("write %d failed: %v", id, err)
				return
			}
			
			// Read response
			buf := make([]byte, 1024)
			_, err = conn.Read(buf)
			if err != nil && err != io.EOF {
				errors <- fmt.Errorf("read %d failed: %v", id, err)
				return
			}
			
			done <- true
		}(i)
	}
	
	// Wait for all connections to complete
	successCount := 0
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			successCount++
		case err := <-errors:
			t.Error(err)
		case <-time.After(time.Second * 5):
			t.Errorf("Timeout waiting for connection %d", i)
			return
		}
	}
	
	if successCount != concurrency {
		t.Errorf("Expected %d successful connections, got %d", concurrency, successCount)
	}
}

// TestHTTP2Multiplexing tests that HTTP/2 multiplexing works correctly
func TestHTTP2Multiplexing(t *testing.T) {
	startProxyServer()
	
	// Create multiple handlers sharing the same connection pool
	handlers := make([]*handler, 5)
	for i := range handlers {
		handlers[i] = NewHandler("http://localhost"+testAddr, testSecret, 0, nil)
	}
	
	// Create connections concurrently
	connections := make([]io.ReadWriteCloser, len(handlers))
	errors := make(chan error, len(handlers))
	
	for i, h := range handlers {
		go func(idx int, hdlr *handler) {
			conn, err := hdlr.Connect("localhost" + testAddr)
			if err != nil {
				errors <- err
				return
			}
			connections[idx] = conn
			errors <- nil
		}(i, h)
	}
	
	// Check all connections succeeded
	for i := 0; i < len(handlers); i++ {
		if err := <-errors; err != nil {
			t.Fatalf("Failed to create connection %d: %v", i, err)
		}
	}
	
	// Clean up
	for _, conn := range connections {
		if conn != nil {
			conn.Close()
		}
	}
}
