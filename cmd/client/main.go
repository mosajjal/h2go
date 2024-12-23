package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	h2go "github.com/mosajjal/h2go"
)

var (
	GitTag    = "2000.01.01.release"
	BuildTime = "2000-01-01T00:00:00+0800"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

func main() {
	addr := flag.String("addr", "127.0.0.1:1080", "listen addr")
	raddr := flag.String("raddr", "", "remote http url(e.g, https://example.com)")
	secret := flag.String("secret", "", "secret key")
	version := flag.Bool("version", false, "version")
	interval := flag.Duration("interval", 0, "interval of pulling, 0 means use http chunked")
	cert := flag.String("cert", "", "cert file")
	flag.Parse()

	if *version {
		fmt.Printf("GitTag: %s \n", GitTag)
		fmt.Printf("BuildTime: %s \n", BuildTime)
		os.Exit(0)
	}
	if *cert != "" {
		h2go.Init(log, *cert)
	}
	s := h2go.Server{Addr: *addr, Logger: log}
	handler := &h2go.Handler{
		Server:   *raddr,
		Secret:   *secret,
		Interval: *interval,
	}
	s.HTTPHandler = handler
	s.Socks5Handler = handler
	log.Error("error",
		"msg", s.ListenAndServe())
}
