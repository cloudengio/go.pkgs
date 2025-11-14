// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
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
	certStore autocert.Cache
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
func NewCertServingCache(ctx context.Context, certStore autocert.Cache, opts ...CertServingCacheOption) *CertServingCache {
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
		return nil, fmt.Errorf("server name %q is not a qualified domain name", name)
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

	// New cert file loaded.
	privPEM, _, certsPEM := ParsePEM(data)
	if len(certsPEM) == 0 {
		return nil, fmt.Errorf("no certificates found for %v", name)
	}
	if len(privPEM) == 0 {
		return nil, fmt.Errorf("no private key found for %v", name)
	}

	// Verify cert chain.
	certs, err := parseCertsPEM(certsPEM)
	if err != nil {
		return nil, err
	}
	opts := x509.VerifyOptions{
		DNSName:     name,
		Roots:       m.rootCAs,
		CurrentTime: now,
	}
	_, err = verifyCertChainOpts(certs, opts)
	if err != nil {
		return nil, err
	}

	if certs[0].IsCA {
		return nil, fmt.Errorf("leaf certificate is a CA cert for %v", name)
	}

	// Prepare tls.Certificate
	cert := pem.EncodeToMemory(certsPEM[0])
	priv := pem.EncodeToMemory(privPEM[0])
	// Load tls.Certificate
	tlscert, err := tls.X509KeyPair(cert, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to load x509 key pair for %v: %w", name, err)
	}

	m.put(name, &tlscert, m.ttl)
	return &tlscert, nil
}
