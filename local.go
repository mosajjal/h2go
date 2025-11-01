package h2go

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
)

var hc = &http.Client{Transport: http.DefaultTransport}

// configureHTTP2Transport creates and configures an HTTP/2 capable transport
func configureHTTP2Transport(tlsConfig *tls.Config) *http.Transport {
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	// Enable HTTP/2
	http2.ConfigureTransport(transport)
	return transport
}

func Init(logger *slog.Logger, cert string) {
	if logger == nil {
		logger = DefaultLogger()
	}
	
	// Create TLS config with HTTP/2 support
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2", "http/1.1"}, // Prefer HTTP/2
	}
	
	if f, err := os.Stat(cert); err == nil && !f.IsDir() {
		var CAPOOL *x509.CertPool
		CAPOOL, err := x509.SystemCertPool()
		if err != nil {
			logger.Warn("system cert pool err",
				"err", err)
			CAPOOL = x509.NewCertPool()
		}
		serverCert, err := os.ReadFile(cert)
		if err != nil {
			logger.Error("error while reading cert.pem",
				"err", err)
			return
		}
		CAPOOL.AppendCertsFromPEM(serverCert)
		tlsConfig.RootCAs = CAPOOL
		logger.Info("loaded certificate",
			"cert", cert)
	} else if err != nil {
		logger.Error("error reading cert file",
			"err", err)
	} else {
		logger.Error("cert file is a directory",
			"cert", cert)
	}
	
	hc = &http.Client{Transport: configureHTTP2Transport(tlsConfig)}
}

type localProxyConn struct {
	uuid     string
	server   string
	secret   string
	source   io.ReadCloser
	close    chan bool
	interval time.Duration
	dst      io.WriteCloser
	logger   *slog.Logger
}

func (c *localProxyConn) genSign(req *http.Request) {

	ts := fmt.Sprintf("%d", time.Now().Unix())
	req.Header.Set("UUID", c.uuid)
	req.Header.Set("timestamp", ts)
	req.Header.Set("sign", GenHMACSHA1(c.secret, ts))
}

func (c *localProxyConn) chunkPush(data []byte, typ string) error {
	if c.dst != nil {
		_, err := c.dst.Write(data)
		return err
	}
	wr, ww := io.Pipe()
	// If NewRequest is called with a context that has a cancel function,
	// it will get called when any of the functions within the goroutine
	// fail and return an error. This creates a race condition, hence
	// this function should be called with a background() context
	// moreover, if we return on an error here, the connection will
	// never be closed
	req, err := http.NewRequest("POST", c.server+PUSH, wr)
	if err != nil {
		c.logger.Warn("chunkPush",
			"err", err)
	}
	req.Header.Set("TYP", typ)
	req.Header.Set("Transfer-Encoding", "chunked")
	c.genSign(req)
	req.Header.Set("Content-Type", "image/jpeg")
	go func() (err error) {
		defer wr.Close()
		defer ww.Close()
		res, err := hc.Do(req)
		if err != nil {
			return err
		}
		c.logger.Debug("chunkPush",
			"status", res.StatusCode)
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		c.logger.Debug("chunkPush",
			"body", string(body))
		switch res.StatusCode {
		case HeadOK:
			return nil
		default:
			return fmt.Errorf("status code is %d, body is: %s", res.StatusCode, string(body))
		}
	}()

	c.dst = ww
	_, err = c.dst.Write(data)
	return err
}

func (c *localProxyConn) push(data []byte, typ string) error {
	buf := bytes.NewBuffer(data)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", c.server+PUSH, buf)
	if err != nil {
		return err
	}
	req.Header.Set("TYP", typ)
	c.genSign(req)
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "image/jpeg")
	c.logger.Debug("push",
		"uuid", c.uuid,
		"typ", typ,
		"server", c.server+PUSH)

	// if there's a QUIT packet is going to end a connection that doesn't have a UUID on the server side
	// it will cause some issues
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	c.logger.Debug("push",
		"status", res.StatusCode,
		"body", string(body))
	switch res.StatusCode {
	case HeadOK:
		return nil
	default:
		return fmt.Errorf("status code is %d, body is: %s", res.StatusCode, string(body))
	}
}

func (c *localProxyConn) connect(dstHost, dstPort string) (uuid string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", c.server+CONNECT, nil)
	if err != nil {
		return "", err
	}
	c.genSign(req)
	req.Header.Set("DSTHOST", dstHost)
	req.Header.Set("DSTPORT", dstPort)
	c.logger.Debug("connect",
		"server", c.server+CONNECT,
		"dstHost", dstHost,
		"dstPort", dstPort)
	res, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	res.Body.Close()
	if res.StatusCode != HeadOK {
		return "", fmt.Errorf("status code is %d, body is:%s", res.StatusCode, string(body))
	}
	return string(body), err

}

func (c *localProxyConn) pull() error {

	req, err := http.NewRequest("GET", c.server+PULL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Interval", fmt.Sprintf("%d", c.interval))
	c.genSign(req)
	if c.interval > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*timeout)
		defer cancel()
		req.WithContext(ctx)
	}
	c.logger.Debug("pull",
		"server", c.server+PULL,
		"uuid", c.uuid)
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != HeadOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		res.Body.Close()
		return fmt.Errorf("status code is %d, body is %s", res.StatusCode, string(body))
	}
	c.source = res.Body
	return nil
}

func (c *localProxyConn) Read(b []byte) (n int, err error) {

	if c.source == nil {
		if c.interval > 0 {
			if err = c.pull(); err != nil {
				c.logger.Debug("pull error",
					"err", err)
				return
			}
		} else {
			return 0, errors.New("pull http connection is not ready")
		}
	}
	n, err = c.source.Read(b)
	c.logger.Debug("read",
		"uuid", c.uuid,
		"err", err,
		"n", n,
		"b", b[:n])
	if err != nil {
		c.source.Close()
		c.source = nil
	}
	if err == io.EOF && c.interval > 0 {
		// c.logger.Debug("ignoreing eof",
		// 	"eof", "true")
		err = nil
	}
	return
}

func (c *localProxyConn) Write(b []byte) (int, error) {

	var err error
	if c.interval > 0 {
		err = c.push(b, DATA_TYP)
	} else {
		//err = c.push(b, DATA_TYP)
		// this chunkpush does not respect the timeout value
		c.logger.Debug("chunkPush",
			"b", b)
		err = c.chunkPush(b, DATA_TYP)
	}
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (c *localProxyConn) alive() {
	for {
		select {
		case <-c.close:
			return
		case <-time.After(time.Second * heartTTL / 2):
			if err := c.push([]byte("alive"), HEART_TYP); err != nil {
				return
			}
		}
	}
}

func (c *localProxyConn) quit() error {
	return c.push([]byte("quit"), QUIT_TYP)
}

func (c *localProxyConn) Close() error {
	c.logger.Debug("close",
		"uuid", c.uuid)
	close(c.close)
	return c.quit()
}
