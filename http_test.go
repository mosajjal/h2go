package h2go

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

var (
	lock            sync.Mutex
	serverHaveStart bool
	testP           *httpProxy
)

const (
	testAddr   = ":12245"
	testSecret = "12345"
)

func startProxyServer() {
	lock.Lock()
	defer lock.Unlock()
	if serverHaveStart {
		return
	}
	testP = NewHttpProxy(nil, testAddr, testSecret, false)
	go testP.Listen()
	time.Sleep(time.Millisecond * 100)
	serverHaveStart = true
}

func TestHandler_Connect(t *testing.T) {
	startProxyServer()

	res, err := http.Get(fmt.Sprintf("http://127.0.0.1%s%s", testAddr, CONNECT))
	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != 404 {
		t.Error("status code not equal 404")
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if string(body) != "404" {
		t.Error("body not equal 404")
	}
}
