// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/webapp"
	"golang.org/x/crypto/acme/autocert"
)

// mockCertStore is a mock implementation of the CertStore interface for testing.
type mockCertStore struct {
	mu       sync.Mutex
	certs    map[string][]byte
	storeErr error
	getHits  int
}

func newMockCertStore() *mockCertStore {
	return &mockCertStore{
		certs: make(map[string][]byte),
	}
}

func (s *mockCertStore) Get(_ context.Context, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storeErr != nil {
		return nil, s.storeErr
	}
	cert, ok := s.certs[name]
	if !ok {
		return nil, autocert.ErrCacheMiss
	}
	s.getHits++
	return cert, nil
}

func (s *mockCertStore) Put(_ context.Context, name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storeErr != nil {
		return s.storeErr
	}
	s.certs[name] = data
	return nil
}

func (s *mockCertStore) Delete(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.storeErr != nil {
		return s.storeErr
	}
	delete(s.certs, name)
	return nil
}

func (s *mockCertStore) GetHits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getHits
}

// generateTestCert generates a certificate chain (leaf, intermediate, root) and private key for a given domain.
// It returns the PEM encoded leaf private key + leaf certificate + intermediate certificate,
// and a root CA pool containing the root certificate.
func generateTestCert(t *testing.T, domain string, notBefore, notAfter time.Time) ([]byte, *x509.CertPool) {
	t.Helper()

	// 1. Root CA
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate root private key: %v", err)
	}
	rootTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-root-ca"},
		NotBefore:             notBefore,
		NotAfter:              notAfter.Add(time.Hour * 24 * 365), // Root cert valid for longer
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	rootCertBytes, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create root certificate: %v", err)
	}
	rootCert, err := x509.ParseCertificate(rootCertBytes)
	if err != nil {
		t.Fatalf("failed to parse root certificate: %v", err)
	}

	// 2. Intermediate CA
	intermediateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate intermediate private key: %v", err)
	}
	intermediateTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "test-intermediate-ca"},
		NotBefore:             notBefore,
		NotAfter:              notAfter.Add(time.Hour * 24 * 30), // Intermediate valid for less time
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	intermediateCertBytes, err := x509.CreateCertificate(rand.Reader, &intermediateTemplate, rootCert, &intermediateKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("failed to create intermediate certificate: %v", err)
	}
	intermediateCert, err := x509.ParseCertificate(intermediateCertBytes)
	if err != nil {
		t.Fatalf("failed to parse intermediate certificate: %v", err)
	}

	// 3. Leaf (Server) Certificate
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate leaf private key: %v", err)
	}
	leafTemplate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: domain},
		DNSNames:     []string{domain},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafCertBytes, err := x509.CreateCertificate(rand.Reader, &leafTemplate, intermediateCert, &leafKey.PublicKey, intermediateKey)
	if err != nil {
		t.Fatalf("failed to create leaf certificate: %v", err)
	}

	// 4. PEM encode and combine
	var certChainPEM, keyPEM bytes.Buffer

	// Leaf private key
	leafKeyBytes, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		t.Fatalf("failed to marshal leaf private key: %v", err)
	}
	if err := pem.Encode(&keyPEM, &pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyBytes}); err != nil {
		t.Fatalf("failed to encode leaf key to PEM: %v", err)
	}

	// Leaf certificate
	if err := pem.Encode(&certChainPEM, &pem.Block{Type: "CERTIFICATE", Bytes: leafCertBytes}); err != nil {
		t.Fatalf("failed to encode leaf cert to PEM: %v", err)
	}

	// Intermediate certificate
	if err := pem.Encode(&certChainPEM, &pem.Block{Type: "CERTIFICATE", Bytes: intermediateCertBytes}); err != nil {
		t.Fatalf("failed to encode intermediate cert to PEM: %v", err)
	}

	// 5. Create Root CA Pool
	rootCAs := x509.NewCertPool()
	var rootCertPEM bytes.Buffer
	if err := pem.Encode(&rootCertPEM, &pem.Block{Type: "CERTIFICATE", Bytes: rootCertBytes}); err != nil {
		t.Fatalf("failed to encode root cert to PEM: %v", err)
	}
	rootCAs.AppendCertsFromPEM(rootCertPEM.Bytes())

	return append(keyPEM.Bytes(), certChainPEM.Bytes()...), rootCAs
}

func TestCertServingCache_GetCertificate(t *testing.T) {
	ctx := t.Context()
	domain := "example.com"
	now := time.Now()
	certData, rootCAs := generateTestCert(t, domain, now.Add(-time.Hour), now.Add(time.Hour))

	store := newMockCertStore()
	if err := store.Put(ctx, domain, certData); err != nil {
		t.Fatal(err)
	}

	cache := webapp.NewCertServingCache(ctx, store, webapp.CertCacheRootCAs(rootCAs))

	// 1. First call: cache miss, should fetch from store.
	cert, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err != nil {
		t.Fatalf("first GetCertificate failed: %v", err)
	}
	if cert == nil {
		t.Fatal("first GetCertificate returned nil cert")
	}
	if hits := store.GetHits(); hits != 1 {
		t.Errorf("expected 1 store hit, got %d", hits)
	}

	// 2. Second call: cache hit, should not fetch from store.
	cert2, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err != nil {
		t.Fatalf("second GetCertificate failed: %v", err)
	}
	if cert2 == nil {
		t.Fatal("second GetCertificate returned nil cert")
	}
	if hits := store.GetHits(); hits != 1 {
		t.Errorf("expected 1 store hit after cache hit, got %d", hits)
	}
}

func TestCertServingCache_Expiry(t *testing.T) {
	ctx := t.Context()
	domain := "example.com"
	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	certData, rootCAs := generateTestCert(t, domain, startTime.Add(-time.Hour), startTime.Add(2*time.Hour))

	store := newMockCertStore()
	if err := store.Put(ctx, domain, certData); err != nil {
		t.Fatal(err)
	}

	// Use a mock time function to control expiry.
	mockTime := startTime
	nowFunc := func() time.Time { return mockTime }

	cache := webapp.NewCertServingCache(ctx, store,
		webapp.CertCacheRootCAs(rootCAs),
		webapp.CertCacheTTL(time.Hour),
		webapp.CertCacheNowFunc(nowFunc),
	)

	// 1. First call to populate cache.
	if _, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain}); err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if hits := store.GetHits(); hits != 1 {
		t.Errorf("expected 1 store hit, got %d", hits)
	}

	// 2. Advance time, but not past TTL. Should be a cache hit.
	mockTime = mockTime.Add(30 * time.Minute)
	if _, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain}); err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if hits := store.GetHits(); hits != 1 {
		t.Errorf("expected 1 store hit after 30 mins, got %d", hits)
	}

	// 3. Advance time past TTL. Should be a cache miss, fetching from store again.
	mockTime = mockTime.Add(40 * time.Minute) // Total advance is 70 mins > 1 hour TTL
	if _, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain}); err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if hits := store.GetHits(); hits != 2 {
		t.Errorf("expected 2 store hits after TTL expiry, got %d", hits)
	}
}

func TestCertServingCache_CacheMiss(t *testing.T) {
	ctx := t.Context()
	domain := "not-in-cache.com"
	store := newMockCertStore()
	cache := webapp.NewCertServingCache(ctx, store)

	// Request a certificate that is not in the cache or the store.
	// This is a cache miss, and the store will also return a "not found" error.
	_, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err == nil {
		t.Fatal("expected an error for a certificate that is not in the store, but got nil")
	}

	if !errors.Is(err, autocert.ErrCacheMiss) {
		t.Errorf("expected error to be %v, but got %v", autocert.ErrCacheMiss, err)
	}

	if hits := store.GetHits(); hits != 0 {
		t.Errorf("expected 0 store hits for a cache miss, got %d", hits)
	}

}

func TestCertServingCache_Errors(t *testing.T) {
	ctx := t.Context()
	store := newMockCertStore()

	startTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	certData, _ := generateTestCert(t, "store-error.com", startTime.Add(-time.Hour), startTime.Add(2*time.Hour))

	if err := store.Put(ctx, "store-error.com", certData); err != nil {
		t.Fatal(err)
	}

	idx := bytes.Index(certData, []byte("-----BEGIN CERTIFICATE-----"))
	if idx == -1 {
		t.Fatal("failed to find certificate in generated cert data")
	}
	noKey := certData[idx:]

	testCases := []struct {
		name       string
		serverName string
		storeData  []byte
		storeErr   error
		rootCAs    *x509.CertPool
		wantErr    string
	}{
		{
			name:       "missing server name",
			serverName: "",
			wantErr:    "missing server name",
		},
		{
			name:       "invalid server name",
			serverName: "localhost",
			wantErr:    "server name \"localhost\" is not a qualified domain name",
		},
		{
			name:       "store error",
			serverName: "store-error.com",
			storeErr:   errors.New("store unavailable"),
			wantErr:    "store unavailable",
		},
		{
			name:       "no private key",
			serverName: "no-key.com",
			storeData:  noKey,
			wantErr:    "no private key",
		},
		{
			name:       "expired cert",
			serverName: "expired.com",
			storeData: func() []byte {
				now := time.Now()
				data, _ := generateTestCert(t, "expired.com", now.Add(-2*time.Hour), now.Add(-time.Hour))
				return data
			}(),
			rootCAs: x509.NewCertPool(), // No roots, will fail verification
			wantErr: "has expired or is not yet valid",
		},
		{
			name:       "too early cert",
			serverName: "too-early.com",
			storeData: func() []byte {
				now := time.Now()
				data, _ := generateTestCert(t, "too-early.com", now.Add(time.Hour), now.Add(2*time.Hour))
				return data
			}(),
			rootCAs: x509.NewCertPool(), // No roots, will fail verification
			wantErr: "has expired or is not yet valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store.mu.Lock()
			if tc.storeData != nil {
				store.certs[tc.serverName] = tc.storeData
			}
			store.storeErr = tc.storeErr
			store.mu.Unlock()

			cache := webapp.NewCertServingCache(ctx, store)
			if tc.rootCAs != nil {
				// Create a new cache for tests that need specific roots.
				cache = webapp.NewCertServingCache(ctx, store, webapp.CertCacheRootCAs(tc.rootCAs))
			}

			_, err := cache.GetCertificate(&tls.ClientHelloInfo{ServerName: tc.serverName})
			if err == nil {
				t.Fatalf("expected error containing %q, but got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error to contain %q, but got %q", tc.wantErr, err.Error())
			}
		})
	}
}
