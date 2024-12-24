package h2go

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestProxyConn(t *testing.T) {
	startProxyServer()

	h := NewHandler("http://localhost"+testAddr, testSecret, 0, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Error(err)
	}
	conn.Write([]byte("GET /ping HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	go func() {
		time.Sleep(time.Millisecond * 100)
		conn.Close()
	}()
	body, err := io.ReadAll(conn)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(string(body), "pong") {
		t.Error("expected pong")
	}
}

func TestProxyConn2(t *testing.T) {
	startProxyServer()

	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Error(err)
	}
	conn.Write([]byte("GET /connect HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	go func() {
		time.Sleep(time.Millisecond * 100)
		conn.Close()
	}()
	body, err := io.ReadAll(conn)
	// we expect an error here
	if err == nil {
		t.Error("got no error, expected 404")
	}
	if !strings.Contains(string(body), "404") {
		t.Error("expected 404 in body")
	}
}

// func TestProxyConn3(t *testing.T) {
// 	startProxyServer()

// 	h := NewHandler("http://localhost"+testAddr, testSecret, 0, nil)
// 	conn, err := h.Connect("localhost" + testAddr)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	p, ok := conn.(*localProxyConn)
// 	if !ok {
// 		t.Error("not ok")
// 	}
// 	// wrong uuid
// 	p.uuid = ""
// 	n, err := p.Write([]byte("GET /connect HTTP/1.1\r\nHost: localhost\r\n\r\n"))
// 	log.Printf("%d bytes written\n", n)

// 	time.Sleep(2 * time.Second)
// 	p.Close()

// 	// we expect an error here
// 	if err == nil {
// 		t.Error("got no error, expected 404")
// 	}
// 	if !strings.Contains(err.Error(), "uuid don't exist") {
// 		t.Error("\"uuid don't exist\" should've been the error")
// 	}

// }

func TestProxyConn4(t *testing.T) {
	startProxyServer()

	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Error(err)
	}
	p, ok := conn.(*localProxyConn)
	if !ok {
		t.Error("not ok")
	}
	// wrong uuid
	p.uuid = ""
	body, err := io.ReadAll(conn)
	// we expect an error here
	if err == nil {
		t.Error("got no error, expected 404")
	}
	if !strings.Contains(err.Error(), "uuid don't exist") {
		t.Error("\"uuid don't exist\" should've been the error")
	}
	conn.Close()
	// body should be empty
	if len(body) != 0 {
		t.Error("body should be empty")
	}
}

func TestProxyConn5(t *testing.T) {
	startProxyServer()

	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Error(err)
	}
	p, ok := conn.(*localProxyConn)
	if !ok {
		t.Error("not ok")
	}
	remote, ok := testP.proxyMap[p.uuid]
	if !ok {
		t.Error("not ok")
	}
	if remote.IsClosed() {
		t.Error("is closed")
	}
	conn.Close()
}

func TestProxyConn6(t *testing.T) {
	startProxyServer()

	h := NewHandler("http://localhost"+testAddr, testSecret, time.Millisecond*20, nil)
	conn, err := h.Connect("localhost" + testAddr)
	if err != nil {
		t.Error(err)
	}
	p, ok := conn.(*localProxyConn)
	if !ok {
		t.Error("not ok")
	}
	remote, ok := testP.proxyMap[p.uuid]
	if !ok {
		t.Error("not ok")
	}
	conn.Close()
	if !remote.IsClosed() {
		t.Error("not closed")
	}
}
