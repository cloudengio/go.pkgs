// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

type selfSignedCertOptions struct {
	ips  []string
	dns  []string
	key  crypto.PrivateKey
	orgs []string
}

type SelfSignedOption func(ssc *selfSignedCertOptions)

// CertDNSHosts specifies the set of dns host names to use in the
// generated certificate.
func CertDNSHosts(hosts ...string) SelfSignedOption {
	return func(ssc *selfSignedCertOptions) {
		ssc.dns = hosts
	}
}

// CertIPAddresses specifies the set of ip addresses to use in the
// generated certificate.
func CertIPAddresses(ips ...string) SelfSignedOption {
	return func(ssc *selfSignedCertOptions) {
		ssc.ips = ips
	}
}

// CertIPAddresses specifies that all local IPs be used in the generated
// certificate.
func CertAllIPAddresses() SelfSignedOption {
	return func(ssc *selfSignedCertOptions) {
		ssc.ips = allIPS()
	}
}

// CertPrivateKey specifies the private key to use for the certificate.
func CertPrivateKey(key crypto.PrivateKey) SelfSignedOption {
	return func(ssc *selfSignedCertOptions) {
		ssc.key = key
	}
}

// CertOrganizations specifies that the organization to be used in the generated
// certificate.
func CertOrganizations(orgs ...string) SelfSignedOption {
	return func(ssc *selfSignedCertOptions) {
		ssc.orgs = orgs
	}
}

func (opts *selfSignedCertOptions) setDefaults() error {
	if opts.key == nil {
		priv, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil
		}
		opts.key = priv
	}

	if len(opts.dns) == 0 {
		opts.dns = []string{"localhost"}
	}

	if len(opts.ips) == 0 {
		opts.ips = []string{"127.0.0.1"}
	}

	if len(opts.orgs) == 0 {
		opts.orgs = []string{"cloudeng llc"}
	}
	return nil
}

// NewSelfSignedCert creates a self signed certificate. Default values for
// the supported options are:
//   - an rsa 4096 bit private key will be geenrated and used.
//   - "localhost" and "127.0.0.1" are used for the DNS and IP addresses
//     used for the certificate.
//   - certificates are valid from time.Now() and for 5 days.
//   - the organization is 'cloudeng llc'.
func NewSelfSignedCert(certFile, keyFile string, options ...SelfSignedOption) error {
	opts := selfSignedCertOptions{}
	for _, fn := range options {
		fn(&opts)
	}
	if err := opts.setDefaults(); err != nil {
		return err
	}

	// based on crypto/tls/generate_cert.go

	// ECDSA, ED25519 and RSA subject keys should have the DigitalSignature
	// KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature

	// Only RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	if _, isRSA := opts.key.(*rsa.PrivateKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}
	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 5)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: opts.orgs,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(opts.key), opts.key)
	if err != nil {
		return fmt.Errorf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("Failed to open %v for writing: %v", certFile, err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("Failed to write data to %v: %v", certFile, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("Error closing %v: %v", certFile, err)
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Failed to open %v for writing: %v", keyFile, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(opts.key)
	if err != nil {
		return fmt.Errorf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("Failed to write data to %v: %v", keyFile, err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("Error closing %v: %v", keyFile, err)
	}
	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func allIPS() []string {
	var all []string
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			switch addr.(type) {
			case *net.IPNet:
			case *net.IPAddr:
			default:
				continue
			}
			all = append(all, addr.String())
		}
	}
	return all
}
