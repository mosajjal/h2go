package h2go

import (
	"fmt"
	"io"
	"testing"
	"time"
)

// BenchmarkProxyConnect measures the performance of establishing a connection
func BenchmarkProxyConnect(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, 0, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := h.Connect("localhost" + testAddr)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		conn.Close()
	}
}

// BenchmarkProxyConnectWithInterval measures connection performance with interval polling
func BenchmarkProxyConnectWithInterval(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := h.Connect("localhost" + testAddr)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		conn.Close()
	}
}

// BenchmarkProxyWriteRead measures throughput of write and read operations
func BenchmarkProxyWriteRead(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	data := []byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n")
	buf := make([]byte, 1024)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := conn.Write(data)
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
		
		_, err = conn.Read(buf)
		if err != nil && err != io.EOF {
			// Expected timeout in polling mode, not a failure
			continue
		}
	}
}

// BenchmarkProxyWriteSmall benchmarks small data writes
func BenchmarkProxyWriteSmall(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	data := make([]byte, 64) // 64 bytes
	
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := conn.Write(data)
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

// BenchmarkProxyWriteMedium benchmarks medium data writes
func BenchmarkProxyWriteMedium(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	data := make([]byte, 4096) // 4KB
	
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := conn.Write(data)
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

// BenchmarkProxyWriteLarge benchmarks large data writes
func BenchmarkProxyWriteLarge(b *testing.B) {
	startProxyServer()
	
	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	data := make([]byte, 65536) // 64KB
	
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := conn.Write(data)
		if err != nil {
			b.Fatalf("Failed to write: %v", err)
		}
	}
}

// BenchmarkConcurrentConnections benchmarks multiple concurrent connections
func BenchmarkConcurrentConnections(b *testing.B) {
	startProxyServer()
	
	b.RunParallel(func(pb *testing.PB) {
		h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
		for pb.Next() {
			conn, err := h.Connect("localhost" + testAddr)
			if err != nil {
				b.Fatalf("Failed to connect: %v", err)
			}
			
			// Send small request
			conn.Write([]byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
			
			// Read response
			buf := make([]byte, 1024)
			conn.Read(buf)
			
			conn.Close()
		}
	})
}

// BenchmarkHMACGeneration benchmarks HMAC signature generation
func BenchmarkHMACGeneration(b *testing.B) {
	secret := testSecret
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenHMACSHA1(secret, timestamp)
	}
}

// BenchmarkHMACVerification benchmarks HMAC signature verification
func BenchmarkHMACVerification(b *testing.B) {
	secret := testSecret
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signature := GenHMACSHA1(secret, timestamp)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyHMACSHA1(secret, timestamp, signature)
	}
}
