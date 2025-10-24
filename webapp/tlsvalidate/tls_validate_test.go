// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package tlsvalidate_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp/tlsvalidate"
)

func newCert(t *testing.T, name string, isCA bool, san []string, ipSANs []net.IP, signer *x509.Certificate, signerKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().Unix()),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              san,
		IPAddresses:           ipSANs,
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	var parent *x509.Certificate
	var parentKey *rsa.PrivateKey
	if signer == nil {
		parent = template
		parentKey = privKey
	} else {
		parent = signer
		parentKey = signerKey
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &privKey.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatal(err)
	}

	return cert, privKey
}

func startTLSServer(t *testing.T, cert *x509.Certificate, key *rsa.PrivateKey, addr string) (string, func()) {
	t.Helper()
	serverCert := tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  key,
	}
	//nolint:gosec // G402 we want to test min version handling
	cfg := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS12,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{TLSConfig: cfg, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		_ = srv.ServeTLS(ln, "", "")
	}()
	return ln.Addr().String(), func() {
		srv.Shutdown(context.Background()) //nolint:errcheck
	}
}

func TestValidator(t *testing.T) {
	ctx := context.Background()

	// 1. Create certs
	rootCert, rootKey := newCert(t, "root.com", true, nil, nil, nil, nil)
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)

	leafCert, leafKey := newCert(t, "leaf.com", false, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, rootCert, rootKey)

	// 2. Start a server
	addr, cleanup := startTLSServer(t, leafCert, leafKey, "127.0.0.1:0")
	defer cleanup()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	_, cleanup6 := startTLSServer(t, leafCert, leafKey, net.JoinHostPort("::1", port))
	defer cleanup6()

	testCases := []struct {
		name     string
		opts     []tlsvalidate.Option
		host     string
		port     string
		errorMsg string
	}{
		{
			name: "valid cert",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
			},
			host: host,
			port: port,
		},
		{
			name: "valid cert with SAN",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
			},
			host: "localhost",
			port: port,
		},
		{
			name: "wrong root CAs",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(x509.NewCertPool()),
			},
			host:     host,
			port:     port,
			errorMsg: "certificate signed by unknown authority",
		},
		{
			name: "valid for not met",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithValidForAtLeast(2 * time.Hour),
			},
			host:     host,
			port:     port,
			errorMsg: "is less than the required",
		},
		{
			name: "issuer regex match",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithIssuerRegexps(regexp.MustCompile("CN=root.com")),
			},
			host: host,
			port: port,
		},
		{
			name: "issuer regex no match",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithIssuerRegexps(regexp.MustCompile("CN=wrong.com")),
			},
			host:     host,
			port:     port,
			errorMsg: "does not match any of the specified patterns",
		},
		{
			name: "min tls version met",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithTLSMinVersion(tls.VersionTLS12),
			},
			host: host,
			port: port,
		},
		{
			name: "min tls version not met",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithTLSMinVersion(tls.VersionTLS13),
			},
			host:     host,
			port:     port,
			errorMsg: "tls: protocol version not supported", // This comes from the handshake
		},
		{
			name: "expand dns",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithExpandDNSNames(true),
			},
			host: "localhost",
			port: port,
		},
		{
			name: "ipv4 only",
			opts: []tlsvalidate.Option{
				tlsvalidate.WithRootCAs(rootPool),
				tlsvalidate.WithExpandDNSNames(true),
				tlsvalidate.WithIPv4Only(true),
			},
			host: "localhost",
			port: port,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := tlsvalidate.NewValidator(tc.opts...)
			err := validator.Validate(ctx, tc.host, tc.port)
			if len(tc.errorMsg) > 0 {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
				if !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain %q, but got %q", tc.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckSerialNumbers(t *testing.T) {
	ctx := context.Background()

	// Serve different serial numbers for the same host using ipv4 and ipv6 addresses to expand
	// localhost to two different IPs.

	// Cert 1
	rootCert1, rootKey1 := newCert(t, "root1.com", true, nil, nil, nil, nil)
	leafCert1, leafKey1 := newCert(t, "leaf.com", false, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, rootCert1, rootKey1)
	addr1, cleanup1 := startTLSServer(t, leafCert1, leafKey1, "127.0.0.1:0")
	defer cleanup1()
	_, port1, _ := net.SplitHostPort(addr1)

	// Cert 2 (different serial)
	time.Sleep(time.Second)
	rootCert2, rootKey2 := newCert(t, "root2.com", true, nil, nil, nil, nil)
	leafCert2, leafKey2 := newCert(t, "leaf.com", false, []string{"localhost"}, []net.IP{net.ParseIP("::1")}, rootCert2, rootKey2)
	_, cleanup2 := startTLSServer(t, leafCert2, leafKey2, net.JoinHostPort("::1", port1))
	defer cleanup2()

	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert1)
	rootPool.AddCert(rootCert2)

	// This test is tricky because we need two servers with different certs for the same host.
	// We'll simulate this by validating 'localhost' which resolves to 127.0.0.1, but we'll
	// point to two different ports. This isn't a perfect test of the multi-IP case, but
	// it tests the serial number comparison logic.

	// First, validate against one server, should be fine.
	validator := tlsvalidate.NewValidator(
		tlsvalidate.WithRootCAs(rootPool),
		tlsvalidate.WithExpandDNSNames(true),
		tlsvalidate.WithCheckSerialNumbers(true),
	)
	err := validator.Validate(ctx, "localhost", port1)
	if err == nil {
		t.Fatalf("expected serial number check to fail on first server, but it passed")
	}
	if !strings.Contains(err.Error(), "mismatched serial numbers") {
		t.Errorf("expected mismatched serial numbers error, but got: %v", err)
	}

}
