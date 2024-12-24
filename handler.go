package h2go

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

type handler struct {
	Server   string
	Secret   string
	Interval time.Duration
	Logger   *slog.Logger
}

// NewHandler creates a new client handler
func NewHandler(server, secret string, interval time.Duration, logger *slog.Logger) *handler {
	if logger == nil {
		logger = DefaultLogger()
	}
	return &handler{Server: server, Secret: secret, Interval: interval, Logger: logger}
}

func (h *handler) Connect(addr string) (io.ReadWriteCloser, error) {
	if strings.HasSuffix(h.Server, "/") {
		h.Server = h.Server[:len(h.Server)-1]
	}
	conn := &localProxyConn{server: h.Server, secret: h.Secret, interval: h.Interval, logger: h.Logger}
	host := strings.Split(addr, ":")[0]
	port := strings.Split(addr, ":")[1]
	uuid, err := conn.connect(host, port)
	if err != nil {
		return nil, fmt.Errorf("connect %s %v", addr, err)
	}
	conn.uuid = uuid
	if h.Interval == 0 {
		err = conn.pull()
		if err != nil {
			return nil, err
		}
	}
	conn.close = make(chan bool)
	go conn.alive()
	return conn, nil
}

func (h *handler) Clean() {}
