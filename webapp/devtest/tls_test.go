// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"cloudeng.io/webapp/devtest"
)

func verifyCert(t *testing.T, certFile, keyFile string, wantDNS []string, wantIPs []net.IP) {
	t.Helper()
	// Verify that the generated files can be loaded as a valid key pair.
	_, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("failed to load key pair: %v", err)
	}

	// Verify the contents of the certificate.
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("failed to read cert file: %v", err)
	}
	cert, err := x509.ParseCertificate(certPEM)
	if err != nil {
		// The PEM file can contain multiple blocks, the cert is the first one.
		block, _ := pem.Decode(certPEM)
		if block == nil {
			t.Fatalf("failed to decode PEM block")
		}
		cert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("failed to parse certificate: %v", err)
		}
	}

	if !reflect.DeepEqual(cert.DNSNames, wantDNS) {
		t.Errorf("got DNS names %v, want %v", cert.DNSNames, wantDNS)
	}

	if len(cert.IPAddresses) != len(wantIPs) {
		t.Fatalf("got %d IP addresses, want %d", len(cert.IPAddresses), len(wantIPs))
	}
	for i, ip := range cert.IPAddresses {
		if !ip.Equal(wantIPs[i]) {
			t.Errorf("got IP %v, want %v", ip, wantIPs[i])
		}
	}
}

func TestNewSelfSignedCert(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Test with default options (RSA key).
	t.Run("DefaultRSA", func(t *testing.T) {
		if err := devtest.NewSelfSignedCert(certFile, keyFile); err != nil {
			t.Fatalf("NewSelfSignedCert failed: %v", err)
		}
		verifyCert(t, certFile, keyFile, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	})

	// Test with a specified ECDSA private key.
	t.Run("WithECDSAPrivateKey", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate ECDSA key: %v", err)
		}
		if err := devtest.NewSelfSignedCert(certFile, keyFile, devtest.CertPrivateKey(priv)); err != nil {
			t.Fatalf("NewSelfSignedCert with ECDSA key failed: %v", err)
		}
		verifyCert(t, certFile, keyFile, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")})
	})

	// Test with custom DNS and IP options.
	t.Run("WithCustomHosts", func(t *testing.T) {
		dns := []string{"example.com", "www.example.com"}
		ips := []string{"192.168.1.1", "10.0.0.1"}
		netIPs := []net.IP{net.ParseIP(ips[0]), net.ParseIP(ips[1])}
		opts := []devtest.SelfSignedOption{
			devtest.CertDNSHosts(dns...),
			devtest.CertIPAddresses(ips...),
		}
		if err := devtest.NewSelfSignedCert(certFile, keyFile, opts...); err != nil {
			t.Fatalf("NewSelfSignedCert with custom hosts failed: %v", err)
		}
		verifyCert(t, certFile, keyFile, dns, netIPs)
	})
}

func TestNewSelfSignedCertUsingMkcert(t *testing.T) {
	if _, err := exec.LookPath("mkcert"); err != nil {
		t.Skip("mkcert not found in PATH, skipping test")
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	hosts := []string{"test.example.com", "127.0.0.1"}
	if err := devtest.NewSelfSignedCertUsingMkcert(certFile, keyFile, hosts...); err != nil {
		t.Fatalf("NewSelfSignedCertUsingMkcert failed: %v", err)
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Errorf("cert file was not created: %v", err)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Errorf("key file was not created: %v", err)
	}

	// Test error case with empty file paths.
	err := devtest.NewSelfSignedCertUsingMkcert("", "", hosts...)
	if err == nil {
		t.Error("expected an error for empty file paths, but got nil")
	}
}

func TestCertPoolForTesting(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// 1. Create a cert to use for the test.
	if err := devtest.NewSelfSignedCert(certFile, keyFile); err != nil {
		t.Fatalf("NewSelfSignedCert failed: %v", err)
	}

	// 2. Test creating a pool with the new cert.
	pool, err := devtest.CertPoolForTesting(certFile)
	if err != nil {
		t.Fatalf("CertPoolForTesting failed: %v", err)
	}
	if pool == nil {
		t.Fatal("CertPoolForTesting returned a nil pool")
	}

	// 3. Verify the pool can be used to validate the cert.
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	opts := x509.VerifyOptions{
		Roots: pool,
	}
	if _, err := cert.Verify(opts); err != nil {
		// Since it's a self-signed cert, we expect a specific error.
		// The important part is that the root was found in the pool.
		if _, ok := err.(x509.UnknownAuthorityError); !ok {
			t.Errorf("expected an UnknownAuthorityError, but got: %T %v", err, err)
		}
	}

	// 4. Test error cases.
	if _, err := devtest.CertPoolForTesting("non-existent-file.pem"); err == nil {
		t.Error("expected an error for non-existent file, but got nil")
	}
}
