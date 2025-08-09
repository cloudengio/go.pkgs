// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/idna"
)

type entry struct {
	cert   *tls.Certificate
	expiry time.Time
}

// CertServingCache implements an in-memory cache of TLS/SSL certificates
// loaded from a backing store. Validation of the certificates is
// performed on loading rather than every use. It provides a GetCertificate
// method that can be used by tls.Config.
// A TTL (default of 6 hours) is used so that the in-memory cache will
// reload certificates from the store on a periodic basis (with some jitter)
// to allow for certificates to be refreshed.
type CertServingCache struct {
	ctx       context.Context
	certStore CertStore
	ttl       time.Duration
	rootCAs   *x509.CertPool
	nowFunc   func() time.Time
	cacheMu   sync.Mutex
	cache     map[string]entry
}

// CertServingCacheOption represents options to NewCertServingCache.
type CertServingCacheOption func(*CertServingCache)

// CertCacheRootCAs sets the rootCAs to be used when verifying the validity
// of the certificate loaded from the back store.
func CertCacheRootCAs(rootCAs *x509.CertPool) CertServingCacheOption {
	return func(cs *CertServingCache) {
		cs.rootCAs = rootCAs
	}
}

// CertCacheTTL sets the in-memory TTL beyond which cache entries are
// refreshed. This is generally only required for testing purposes.
func CertCacheTTL(ttl time.Duration) CertServingCacheOption {
	return func(cs *CertServingCache) {
		cs.ttl = ttl
	}
}

// CertCacheNowFunc sets the function used to obtain the current time.
// This is generally only required for testing purposes.
func CertCacheNowFunc(fn func() time.Time) CertServingCacheOption {
	return func(cs *CertServingCache) {
		cs.nowFunc = fn
	}
}

// NewCertServingCache returns a new instance of CertServingCache that
// uses the supplied CertStore. The supplied context is cached and used by
// the GetCertificate method, this allows for credentials etc to be passed
// to the CertStore.Get method called by GetCertificate via the context.
func NewCertServingCache(ctx context.Context, certStore CertStore, opts ...CertServingCacheOption) *CertServingCache {
	sc := &CertServingCache{
		ctx:       ctx,
		cache:     map[string]entry{},
		certStore: certStore,
		nowFunc:   time.Now,
		ttl:       time.Hour * 6,
	}
	for _, fn := range opts {
		fn(sc)
	}
	return sc
}

func (m *CertServingCache) get(name string, when time.Time) *tls.Certificate {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	if entry, ok := m.cache[name]; ok {
		if entry.expiry.After(when) {
			return entry.cert
		}
	}
	return nil
}

func (m *CertServingCache) put(name string, cert *tls.Certificate, expiration time.Duration) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.cache[name] = entry{
		cert:   cert,
		expiry: m.nowFunc().Add(expiration),
	}
}

// GetCertificate can be assigned to tls.Config.GetCertificate.
func (m *CertServingCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := hello.ServerName
	if name == "" {
		return nil, fmt.Errorf("missing server name")
	}
	if !strings.Contains(strings.Trim(name, "."), ".") {
		return nil, fmt.Errorf("server name component count invalid")
	}
	name, err := idna.Lookup.ToASCII(name)
	if err != nil {
		return nil, fmt.Errorf("server name contains invalid character")
	}

	now := m.nowFunc()
	if cert := m.get(name, now); cert != nil {
		return cert, nil
	}

	data, err := m.certStore.Get(m.ctx, name)
	if err != nil {
		return nil, err
	}

	// Private key portion.
	priv, pub := pem.Decode(data)
	if priv == nil || !strings.Contains(priv.Type, "PRIVATE") {
		return nil, fmt.Errorf("no private key for %v", name)
	}
	privKey, err := parsePrivateKey(priv.Bytes)
	if err != nil {
		return nil, err
	}

	// Public portion, iterate over all certs in the chain.
	var pubDER [][]byte
	var pubDERBytes []byte
	for len(pub) > 0 {
		var b *pem.Block
		b, pub = pem.Decode(pub)
		if b == nil {
			break
		}
		pubDER = append(pubDER, b.Bytes)
		pubDERBytes = append(pubDERBytes, b.Bytes...)
	}
	if len(pub) > 0 {
		return nil, fmt.Errorf("corrupt/spurious certs for %v", name)
	}
	x509Certs, err := x509.ParseCertificates(pubDERBytes)
	if err != nil || len(x509Certs) == 0 {
		return nil, fmt.Errorf("no public key/certs found for %v", name)
	}
	leaf, err := m.verifyLeafCert(name, now, x509Certs)
	if err != nil {
		return nil, err
	}
	tlscert := &tls.Certificate{
		Certificate: pubDER,
		PrivateKey:  privKey,
		Leaf:        leaf,
	}
	m.put(name, tlscert, m.ttl)
	return tlscert, nil
}

func (m *CertServingCache) verifyLeafCert(name string, now time.Time, x509Certs []*x509.Certificate) (*x509.Certificate, error) {
	leaf := x509Certs[0]
	intermediates := x509.NewCertPool()
	for _, ic := range x509Certs[1:] {
		intermediates.AddCert(ic)
	}
	_, err := leaf.Verify(x509.VerifyOptions{
		DNSName:       name,
		Intermediates: intermediates,
		CurrentTime:   now,
		Roots:         m.rootCAs,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid leaf cert %v: %v", name, err)
	}
	return leaf, nil
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
//
// Inspired by parsePrivateKey in crypto/tls/tls.go.
func parsePrivateKey(der []byte) (crypto.Signer, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, nil
		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("failed to parse private key")
}
