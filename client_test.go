package h2go

import (
	"io"
	"strings"
	"testing"
	"time"
)

// TestNewClient verifies that NewClient creates a properly configured client.
func TestNewClient(t *testing.T) {
	startProxyServer()

	client := NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
		WithInterval(time.Millisecond*20),
	)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.ServerURL() != "http://localhost"+testAddr {
		t.Errorf("ServerURL() = %v, want %v", client.ServerURL(), "http://localhost"+testAddr)
	}
}

// TestClientConnect verifies that Client.Connect works correctly.
func TestClientConnect(t *testing.T) {
	startProxyServer()

	client := NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
		WithInterval(0),
	)

	conn, err := client.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer conn.Close()

	// Send HTTP request
	_, err = conn.Write([]byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read response
	go func() {
		time.Sleep(time.Millisecond * 100)
		conn.Close()
	}()

	body, err := io.ReadAll(conn)
	if err != nil {
		t.Errorf("ReadAll() error = %v", err)
	}

	if !strings.Contains(string(body), "pong") {
		t.Errorf("expected pong in response, got: %s", string(body))
	}
}

// TestClientConnectWithInterval verifies that Client works with interval polling.
func TestClientConnectWithInterval(t *testing.T) {
	startProxyServer()

	client := NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
		WithInterval(time.Millisecond*20),
	)

	conn, err := client.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	// Send HTTP request
	_, err = conn.Write([]byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Read response with timeout
	done := make(chan bool)
	go func() {
		buf := make([]byte, 1024)
		_, _ = conn.Read(buf)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second * 2):
		t.Error("Timeout waiting for response")
	}

	conn.Close()
}

// TestProxyServerOptions verifies that NewProxyServer configures properly.
func TestProxyServerOptions(t *testing.T) {
	server := NewProxyServer(
		WithListenAddr(":18080"),
		WithServerSecret("test-secret"),
		WithHTTPS(false),
	)

	if server == nil {
		t.Fatal("NewProxyServer returned nil")
	}

	if server.addr != ":18080" {
		t.Errorf("addr = %v, want %v", server.addr, ":18080")
	}

	if server.secret != "test-secret" {
		t.Errorf("secret = %v, want %v", server.secret, "test-secret")
	}

	if server.https != false {
		t.Errorf("https = %v, want %v", server.https, false)
	}
}

// TestLocalServerOptions verifies that NewLocalServer configures properly.
func TestLocalServerOptions(t *testing.T) {
	startProxyServer()

	client := NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
	)

	server := NewLocalServer(
		WithLocalListenAddr("127.0.0.1:19080"),
		WithSocks5Handler(client),
		WithHTTPHandler(client),
		WithDisableSocks5(false),
		WithDisableHTTP(false),
	)

	if server == nil {
		t.Fatal("NewLocalServer returned nil")
	}

	if server.Addr != "127.0.0.1:19080" {
		t.Errorf("Addr = %v, want %v", server.Addr, "127.0.0.1:19080")
	}

	if server.DisableSocks5 != false {
		t.Errorf("DisableSocks5 = %v, want %v", server.DisableSocks5, false)
	}
}

// TestHMACAuthenticator verifies that HMACAuthenticator works correctly.
func TestHMACAuthenticator(t *testing.T) {
	auth := NewHMACAuthenticator("test-secret")

	data := "test-data"
	signature := auth.Sign(data)

	if !auth.Verify(data, signature) {
		t.Error("Verify() returned false for valid signature")
	}

	if auth.Verify(data, "invalid-signature") {
		t.Error("Verify() returned true for invalid signature")
	}

	if auth.Verify("different-data", signature) {
		t.Error("Verify() returned true for different data")
	}
}

// TestConnectorInterface verifies that Client implements Connector.
func TestConnectorInterface(t *testing.T) {
	startProxyServer()

	var connector Connector = NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
		WithInterval(0),
	)

	conn, err := connector.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	conn.Close()
}

// TestProxyHandlerInterface verifies that Client implements ProxyHandler.
func TestProxyHandlerInterface(t *testing.T) {
	startProxyServer()

	var handler ProxyHandler = NewClient(
		WithServerURL("http://localhost"+testAddr),
		WithSecret(testSecret),
		WithInterval(0),
	)

	conn, err := handler.Connect("localhost" + testAddr)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	conn.Close()

	handler.Clean() // Should not panic
}

// TestAuthenticatorInterface verifies that HMACAuthenticator implements Authenticator.
func TestAuthenticatorInterface(t *testing.T) {
	var auth Authenticator = NewHMACAuthenticator("test-secret")

	data := "test-data"
	signature := auth.Sign(data)

	if !auth.Verify(data, signature) {
		t.Error("Verify() returned false for valid signature")
	}
}
