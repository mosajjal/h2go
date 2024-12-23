package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

type CertConfig struct {
	Domains    []string `koanf:"domains"`
	KeyFile    string   `koanf:"keyfile"`
	CertFile   string   `koanf:"certfile"`
	ValidDays  int      `koanf:"validdays"`
	KeyBitSize int      `koanf:"keysize"`
}

func generateCerts(conf CertConfig) error {
	priv, err := rsa.GenerateKey(rand.Reader, conf.KeyBitSize)
	if err != nil {
		return err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(conf.ValidDays) * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: conf.Domains[0],
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, domain := range conf.Domains {
		if ip := net.ParseIP(domain); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, domain)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(conf.CertFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	keyOut, err := os.Create(conf.KeyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}
