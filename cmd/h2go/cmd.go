/*
Package main implements the h2go command-line tool which can run in either
client or server mode. It uses the koanf library for configuration management
and supports configuration via command-line flags and environment variables.

The client mode sets up an HTTP/2 client that connects to a remote server
and can optionally use a certificate for secure communication.
it exposes a combo http and socks5 proxy server to localhost

The server mode sets up an HTTP/2 server that can optionally use HTTPS for
secure communication and acts as a proxy server.

The configuration can be provided via command-line flags or environment
variables prefixed with "H2GO_". The configuration is unmarshaled into a
Config struct.
*/
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/mosajjal/h2go"
	"github.com/spf13/pflag"
)

var (
	version = "v0-UNKNOWN"
	commit  = "NOT PROVIDED"
)

var log = h2go.DefaultLogger()

// Config holds all configuration
type Config struct {
	Version  bool          `koanf:"version"`
	Addr     string        `koanf:"addr"`
	Secret   string        `koanf:"secret"`
	Cert     string        `koanf:"cert"`
	RAddr    string        `koanf:"raddr"`
	Interval time.Duration `koanf:"interval"`
	HTTPS    bool          `koanf:"https"`
	Key      string        `koanf:"key"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: h2go <client|server|gencert> [flags]")
		os.Exit(1)
	}

	mode := os.Args[1]
	if mode != "client" && mode != "server" && mode != "gencert" {
		fmt.Println("First argument must be either 'client', 'server' or 'gencert'")
		os.Exit(1)
	}

	k := koanf.New(".")
	flags := pflag.NewFlagSet("config", pflag.ContinueOnError)

	// Define flags based on mode
	switch mode {
	case "client":
		flags.Bool("version", false, "version")
		flags.String("addr", "127.0.0.1:1080", "listen addr")
		flags.String("secret", "", "secret key")
		flags.String("cert", "", "cert file")
		flags.String("raddr", "", "remote http url(e.g, https://example.com)")
		flags.Duration("interval", 0, "interval of pulling, 0 means use http chunked")
	case "server":
		flags.Bool("version", false, "version")
		flags.String("addr", "", "listen addr")
		flags.String("secret", "", "secret key")
		flags.String("cert", "", "cert file")
		flags.Bool("https", false, "enable https")
		flags.String("key", "", "private key file")
	case "gencert":
		flags.StringArray("domain", []string{}, "domain or IP address. can be multiple")
		flags.String("keyfile", "key.pem", "output private key file")
		flags.String("certfile", "cert.pem", "output certificate file")
		flags.Int("validdays", 390, "certificate validity in days")
		flags.Int("keysize", 2048, "RSA key size in bits")
	}

	if err := flags.Parse(os.Args[2:]); err != nil {
		log.Error("error parsing flags", "err", err)
		os.Exit(1)
	}

	// Load environment variables
	if err := k.Load(env.Provider("H2GO_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "H2GO_")), "_", ".", -1)
	}), nil); err != nil {
		log.Error("error loading env", "err", err)
		os.Exit(1)
	}

	// Load flags
	if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		log.Error("error loading flags", "err", err)
		os.Exit(1)
	}

	var conf Config
	if err := k.Unmarshal("", &conf); err != nil {
		log.Error("error unmarshaling config", "err", err)
		os.Exit(1)
	}

	if conf.Version {
		fmt.Printf("h2go %s (%s)\n", version, commit)
		os.Exit(0)
	}

	switch mode {
	case "client":
		runClient(conf)
	case "server":
		runServer(conf)
	case "gencert":
		certConf := CertConfig{
			Domains:    k.Strings("domain"),
			KeyFile:    k.String("keyfile"),
			CertFile:   k.String("certfile"),
			ValidDays:  k.Int("validdays"),
			KeyBitSize: k.Int("keysize"),
		}
		if len(certConf.Domains) == 0 {
			log.Error("domain is required")
			os.Exit(1)
		}
		if err := generateCerts(certConf); err != nil {
			log.Error("failed to generate certificates", "err", err)
			os.Exit(1)
		}
		log.Info("certificates generated successfully",
			"cert", certConf.CertFile,
			"key", certConf.KeyFile)
	}
}

func runClient(conf Config) {
	if conf.Cert != "" {
		h2go.Init(log, conf.Cert)
	}

	s := h2go.Server{
		Addr:   conf.Addr,
		Logger: log,
	}

	handler := h2go.NewHandler(conf.RAddr, conf.Secret, conf.Interval, log)

	s.HTTPHandler = handler
	s.Socks5Handler = handler
	log.Error("error", "msg", s.ListenAndServe())
}

func runServer(conf Config) {
	p := h2go.NewHttpProxy(log, conf.Addr, conf.Secret, conf.HTTPS)

	if conf.HTTPS {
		for _, file := range []string{conf.Cert, conf.Key} {
			f, err := os.Stat(file)
			if err != nil {
				log.Error("error", "msg", err)
				return
			}
			if f.IsDir() {
				log.Error("error", "msg", fmt.Sprintf("%s should be file", file))
				return
			}
		}
		p.ListenHTTPS(conf.Cert, conf.Key)
	} else {
		p.Listen()
	}
}
