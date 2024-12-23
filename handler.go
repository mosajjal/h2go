package h2go

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

type Handler struct {
	Server   string
	Secret   string
	Interval time.Duration
}

func (h *Handler) Connect(addr string) (io.ReadWriteCloser, error) {
	if strings.HasSuffix(h.Server, "/") {
		h.Server = h.Server[:len(h.Server)-1]
	}
	conn := &localProxyConn{server: h.Server, secret: h.Secret, interval: h.Interval}
	host := strings.Split(addr, ":")[0]
	port := strings.Split(addr, ":")[1]
	uuid, err := conn.connect(host, port)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("connect %s %v", addr, err))
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

func (h *Handler) Clean() {}
