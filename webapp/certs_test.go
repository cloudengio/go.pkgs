// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"cloudeng.io/file/localfs"
)

func newCert(t *testing.T, isCA bool, signer *ecdsa.PrivateKey) ([]byte, *x509.Certificate) {
	t.Helper()
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &signer.PublicKey, signer)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	return derBytes, cert
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	return privateKey
}

func TestParsePEM(t *testing.T) {
	key := newKey(t)
	_, leafCert := newCert(t, false, key)
	_, caCert := newCert(t, true, key)

	var pemData bytes.Buffer
	if err := pem.Encode(&pemData, &pem.Block{Type: "CERTIFICATE", Bytes: leafCert.Raw}); err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(&pemData, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw}); err != nil {
		t.Fatal(err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(&pemData, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatal(err)
	}

	privateKeys, publicKeys, certs := ParsePEM(pemData.Bytes())
	if got, want := len(privateKeys), 1; got != want {
		t.Errorf("got %v, want %v private keys", got, want)
	}
	if got, want := len(publicKeys), 0; got != want {
		t.Errorf("got %v, want %v public keys", got, want)
	}
	if got, want := len(certs), 2; got != want {
		t.Errorf("got %v, want %v certs", got, want)
	}

	der, cert, err := FindLeafPEM(certs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(der, leafCert.Raw) {
		t.Errorf("leaf der does not match")
	}
	if !cert.Equal(leafCert) {
		t.Errorf("leaf cert does not match")
	}
}

func TestReadAndParsePrivateKeyPEM(t *testing.T) {
	ctx := context.Background()
	fs := localfs.New()
	tmpdir := t.TempDir()

	// Test with EC key
	ecKey := newKey(t)
	keyBytes, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	pemFile := fs.Join(tmpdir, "eckey.pem")
	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}
	pemData := pem.EncodeToMemory(pemBlock)
	if err := fs.WriteFileCtx(ctx, pemFile, pemData, 0600); err != nil {
		t.Fatal(err)
	}

	signer, err := ReadAndParsePrivateKeyPEM(ctx, fs, pemFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := signer.(*ecdsa.PrivateKey); !ok {
		t.Errorf("expected ecdsa.PrivateKey, got %T", signer)
	}

	// Test with RSA key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keyBytes = x509.MarshalPKCS1PrivateKey(rsaKey)
	pemFile = fs.Join(tmpdir, "rsakey.pem")
	pemBlock = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}
	pemData = pem.EncodeToMemory(pemBlock)
	if err := fs.WriteFileCtx(ctx, pemFile, pemData, 0600); err != nil {
		t.Fatal(err)
	}

	signer, err = ReadAndParsePrivateKeyPEM(ctx, fs, pemFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := signer.(*rsa.PrivateKey); !ok {
		t.Errorf("expected rsa.PrivateKey, got %T", signer)
	}
}
