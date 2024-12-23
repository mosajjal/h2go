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

	addr := flag.String("addr", "", "listen addr")
	secret := flag.String("secret", "", "secret")
	version := flag.Bool("version", false, "version")
	https := flag.Bool("https", false, "https")
	cert := flag.String("cert", "", "cert file")
	key := flag.String("key", "", "private key file")
	flag.Parse()
	if *version {
		fmt.Printf("GitTag: %s \n", GitTag)
		fmt.Printf("BuildTime: %s \n", BuildTime)
		os.Exit(0)
	}
	p := h2go.NewHttpProxy(log, *addr, *secret, *https)
	if *https {
		f, err := os.Stat(*cert)
		if err != nil {
			log.Error("error", "msg", (err))
			return
		}
		if f.IsDir() {
			log.Error("error", "msg", ("cert should be file"))
			return
		}
		f, err = os.Stat(*key)
		if err != nil {
			log.Error("error", "msg", (err))
			return
		}
		if f.IsDir() {
			log.Error("error", "msg", ("key should be file"))
			return
		}
		p.ListenHTTPS(*cert, *key)
	} else {
		p.Listen()
	}

}
