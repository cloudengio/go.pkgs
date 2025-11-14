// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest

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
	"os/exec"
	"time"
)

// NewSelfSignedCertUsingMkcert uses mkcert (https://github.com/FiloSottile/mkcert) to
// create certificates. If mkcert --install has been run then these certificates will
// be trusted by the browser and other local applications.
func NewSelfSignedCertUsingMkcert(certFile, keyFile string, hosts ...string) error {
	if len(certFile) == 0 || len(keyFile) == 0 {
		return fmt.Errorf("both the crt and key files must be specified")
	}
	if len(hosts) == 0 {
		return fmt.Errorf("at least one host must be specified")
	}
	args := []string{"--cert-file", certFile, "--key-file", keyFile}
	args = append(args, hosts...)
	out, err := exec.Command("mkcert", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create certificates: %v\nOutput: %s", err, out)
	}
	return nil
}

type selfSignedCertOptions struct {
	ips  []string
	dns  []string
	key  crypto.PrivateKey
	orgs []string
}

// SelfSignedOption represents an option to NewSelfSignedCert.
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
//   - an rsa 4096 bit private key will be generated and used.
//   - "localhost" and "127.0.0.1" are used for the DNS and IP addresses
//     included in the certificate.
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
		return fmt.Errorf("failed to generate serial number: %v", err)
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
		DNSNames:              opts.dns,
		IPAddresses:           make([]net.IP, len(opts.ips)),
	}
	for i, ip := range opts.ips {
		template.IPAddresses[i] = net.ParseIP(ip)
		if template.IPAddresses[i] == nil {
			return fmt.Errorf("failed to parse IP address %q", ip)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(opts.key), opts.key)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to open %v for writing: %v", certFile, err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write data to %v: %v", certFile, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("error closing %v: %v", certFile, err)
	}

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %v for writing: %v", keyFile, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(opts.key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write data to %v: %v", keyFile, err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing %v: %v", keyFile, err)
	}
	return nil
}

func publicKey(priv any) any {
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

// CertPoolForTesting returns a new x509.CertPool containing the certs
// in the specified pem files. If no pem files are specified nil it
// will return the system cert pool.
// It is intended for testing purposes only.
func CertPoolForTesting(pemFiles ...string) (*x509.CertPool, error) {
	if len(pemFiles) == 0 {
		return x509.SystemCertPool()
	}
	rootCAs := x509.NewCertPool()
	for _, pemFile := range pemFiles {
		if len(pemFile) == 0 {
			continue
		}
		certs, err := os.ReadFile(pemFile)
		if err != nil {
			return nil, fmt.Errorf("failed to append %q to RootCAs: %v", pemFile, err)
		}

		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			return nil, fmt.Errorf("no certs appended")
		}
	}
	return rootCAs, nil
}
