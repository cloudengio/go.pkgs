// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file/localfs"
	"cloudeng.io/webapp"
)

// Helper function to create a self-signed certificate
func newCert(t *testing.T, name string, isCA bool, signer *x509.Certificate, signerKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{name},
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

func TestVerifyCertChain(t *testing.T) {
	// 1. Create a root CA
	rootCert, rootKey := newCert(t, "root.com", true, nil, nil)
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)

	// 2. Create an intermediate CA signed by the root
	intCert, intKey := newCert(t, "intermediate.com", true, rootCert, rootKey)

	// 3. Create a leaf certificate signed by the intermediate
	leafCert, _ := newCert(t, "leaf.com", false, intCert, intKey)

	// 4. Create a standalone leaf certificate (will fail verification)
	standaloneCert, _ := newCert(t, "standalone.com", false, nil, nil)

	testCases := []struct {
		name        string
		certs       []*x509.Certificate
		roots       *x509.CertPool
		dnsName     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid chain",
			certs:       []*x509.Certificate{leafCert, intCert},
			roots:       rootPool,
			dnsName:     "leaf.com",
			expectError: false,
		},
		{
			name:        "chain with root included",
			certs:       []*x509.Certificate{leafCert, intCert, rootCert},
			roots:       rootPool,
			dnsName:     "leaf.com",
			expectError: false,
		},
		{
			name:        "invalid chain - missing intermediate",
			certs:       []*x509.Certificate{leafCert},
			roots:       rootPool,
			dnsName:     "leaf.com",
			expectError: true,
			errorMsg:    "certificate signed by unknown authority",
		},
		{
			name:        "invalid chain - wrong root",
			certs:       []*x509.Certificate{leafCert, intCert},
			roots:       x509.NewCertPool(), // Empty pool
			dnsName:     "leaf.com",
			expectError: true,
			errorMsg:    "certificate signed by unknown authority",
		},
		{
			name:        "standalone cert",
			certs:       []*x509.Certificate{standaloneCert},
			roots:       rootPool,
			dnsName:     "standalone.com",
			expectError: true,
			errorMsg:    "certificate signed by unknown authority",
		},
		{
			name:        "no certs",
			certs:       []*x509.Certificate{},
			roots:       rootPool,
			dnsName:     "any.com",
			expectError: true,
			errorMsg:    "no certificates supplied",
		},
		{
			name:        "no leaf cert",
			certs:       []*x509.Certificate{intCert, rootCert},
			roots:       rootPool,
			dnsName:     "any.com",
			expectError: true,
			errorMsg:    "no leaf certificate found",
		},
		{
			name:        "multiple leaf certs",
			certs:       []*x509.Certificate{leafCert, standaloneCert, intCert},
			roots:       rootPool,
			dnsName:     "any.com",
			expectError: true,
			errorMsg:    "expected exactly one leaf certificate",
		},
		{
			name:        "wrong dns name",
			certs:       []*x509.Certificate{leafCert, intCert},
			roots:       rootPool,
			dnsName:     "wrong.com",
			expectError: true,
			errorMsg:    `x509: certificate is valid for leaf.com, not wrong.com`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chains, err := webapp.VerifyCertChain(tc.dnsName, tc.certs, tc.roots)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
				if len(tc.errorMsg) > 0 && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain %q, but got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(chains) == 0 {
					t.Errorf("expected at least one verified chain")
				}
			}
		})
	}
}

func TestReadAndParsePEM(t *testing.T) {
	ctx := context.Background()
	fs := localfs.New()
	tmpDir := t.TempDir()

	// Create some certs
	rootCert, rootKey := newCert(t, "root.com", true, nil, nil)
	intCert, _ := newCert(t, "intermediate.com", true, rootCert, rootKey)

	// Create a PEM file with multiple certs
	pemFile := filepath.Join(tmpDir, "certs.pem")
	f, err := os.Create(pemFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: rootCert.Raw}); err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: intCert.Raw}); err != nil {
		t.Fatal(err)
	}
	// Add a non-cert block to ensure it's skipped
	if err := pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("dummy")}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Create an empty PEM file
	emptyFile := filepath.Join(tmpDir, "empty.pem")
	if err := os.WriteFile(emptyFile, []byte{}, 0600); err != nil {
		t.Fatal(err)
	}

	// Create a file with no certs
	noCertsFile := filepath.Join(tmpDir, "nocerts.pem")
	if err := os.WriteFile(noCertsFile, []byte("not a pem file"), 0600); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name        string
		pemFile     string
		expectCerts int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid pem file",
			pemFile:     pemFile,
			expectCerts: 2,
			expectError: false,
		},
		{
			name:        "non-existent file",
			pemFile:     "non-existent.pem",
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name:        "empty file",
			pemFile:     emptyFile,
			expectError: true,
			errorMsg:    "no certificates found",
		},
		{
			name:        "file with no certs",
			pemFile:     noCertsFile,
			expectError: true,
			errorMsg:    "no certificates found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			certs, err := webapp.ReadAndParseCertsPEM(ctx, fs, tc.pemFile)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
				if len(tc.errorMsg) > 0 && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error to contain %q, but got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(certs) != tc.expectCerts {
					t.Errorf("expected %d certs, but got %d", tc.expectCerts, len(certs))
				}
			}
		})
	}
}
