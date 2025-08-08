// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// TLSCertStoreFlags defines commonly used flags for specifying a TLS/SSL
// certificate store. This is generally used in conjunction with
// TLSConfigFromFlags for apps that simply want to use stored certificates.
// Apps that manage/obtain/renew certificates may use them directly.
type TLSCertStoreFlags struct {
	CertStoreType  string `subcmd:"tls-cert-store-type,,'the type of the certificate store to use for retrieving tls certificates, use --tls-list-stores to see the currently available types'"`
	CertStore      string `subcmd:"tls-cert-store,,'name/address of the certificate cache to use for retrieving tls certificates, the interpreation of this depends on the tls-cert-store-type flag'"`
	ListStoreTypes bool   `subcmd:"tls-list-stores,,list the available types of tls certificate store"`
}

// TLSCertFlags defines commonly used flags for obtaining TLS/SSL certificates.
// Certificates may be obtained in one of two ways: from a cache of
// certificates, or from local files.
type TLSCertFlags struct {
	TLSCertStoreFlags
	CertificateFile string `subcmd:"tls-cert,,ssl certificate file"`
	KeyFile         string `subcmd:"tls-key,,ssl private key file"`
}

// HTTPAcmeFlags defines flags for running an http server that will
// handle acme challenges, currently it allows for
// redirecting them to a server that implements the acme client protocol.
type HTTPAcmeFlags struct {
	AcmeRedirectTarget string `subcmd:"acme-redirect-target,,host implementing acme client that this http server will redirect acme challenges to"`
}

// HTTPServerFlags defines commonly used flags for running an http server.
// TLS certificates may be retrieved either from a local cert and key file
// as specified by tls-cert and tls-key; this is generally used for testing
// or when the domain certificates are available only as files.
// The altnerative, preferred for production, source for TLS certificates
// is from a cache as specified by tls-cert-cache-type and tls-cert-cache-name.
// The cache may be on local disk, or preferably in some shared service such
// as Amazon's Secrets Service.
type HTTPServerFlags struct {
	Address string `subcmd:"https,:8080,address to run https web server on"`
	TLSCertFlags
}

// TLSConfigFromFlags creates a tls.Config based on the supplied flags, which
// may require obtaining certificates directly from pem files or from a
// possibly remote certificate store using TLSConfigUsingCertStore. If
// a cert store is specified then the getStoreOpts function
// may be used to obtain additional options for the store. A cache
// is then created to from that store using the supplied cacheOpts.
func TLSConfigFromFlags(ctx context.Context, cl HTTPServerFlags, getStoreOpts func() (opts []any, err error), cacheOpts ...CertServingCacheOption) (*tls.Config, error) {
	if cl.ListStoreTypes {
		return nil, errors.New(strings.Join(RegisteredCertStores(), "\n"))
	}
	useStore := len(cl.CertStoreType) > 0 || len(cl.CertStore) > 0
	useFiles := len(cl.CertificateFile) > 0 || len(cl.KeyFile) > 0
	if useStore && useFiles {
		return nil, fmt.Errorf("can't use both a certificate cache and certificate files")
	}
	if useStore {
		storeOpts := []any{}
		if getStoreOpts != nil {
			var err error
			storeOpts, err = getStoreOpts()
			if err != nil {
				return nil, fmt.Errorf("failed to obtain store options: %v", err)
			}
		}
		store, err := NewCertStore(ctx, cl.CertStoreType, cl.CertStore, storeOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create cert store: %v", err)
		}
		return TLSConfigUsingCertStore(ctx, store, cacheOpts...)
	}
	return TLSConfigUsingCertFiles(cl.CertificateFile, cl.KeyFile)
}

// TLSConfigUsingCertStore returns a tls.Config configured with the
// certificate obtained from the specified certificate store accessed
// via a CertServingCache created with the supplied options.
func TLSConfigUsingCertStore(ctx context.Context, store CertStore, cacheOpts ...CertServingCacheOption) (*tls.Config, error) {
	return &tls.Config{
		GetCertificate: NewCertServingCache(ctx, store, cacheOpts...).GetCertificate,
		MinVersion:     tls.VersionTLS13,
	}, nil
}

func NewCertStore(ctx context.Context, typ, name string, storeOpts ...any) (CertStore, error) {
	factory, err := getFactory(typ)
	if err != nil {
		return nil, err
	}
	store, err := factory.New(ctx, name, storeOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache instance: %v %v: %v", typ, name, err)
	}
	return store, nil
}

// TLSConfigUsingCertFiles returns a tls.Config configured with the
// certificate read from the supplied files.
func TLSConfigUsingCertFiles(certFile, keyFile string) (*tls.Config, error) {
	if len(certFile) == 0 || len(keyFile) == 0 {
		return nil, fmt.Errorf("both the crt and key files must be specified")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// CertStore represents a store for TLS certificates.
type CertStore interface {
	Get(ctx context.Context, name string) ([]byte, error)
	Put(ctx context.Context, name string, data []byte) error
}

// CertStoreFactory is the interface that must be implemented to register
// a new CertStore type with this package so that it may accessed via
// the TLSCertStoreFlags command line flags.
type CertStoreFactory interface {
	Type() string
	Describe() string
	New(ctx context.Context, name string, opts ...any) (CertStore, error)
}

var (
	storesMu sync.Mutex
	stores   = map[string]CertStoreFactory{}
)

func getFactory(typ string) (CertStoreFactory, error) {
	storesMu.Lock()
	defer storesMu.Unlock()
	factory := stores[typ]
	if factory == nil {
		return nil, fmt.Errorf("unsupported cert store type: %v", typ)
	}
	return factory, nil
}

// RegisterCertStoreFactory makes the supplied CertStoreFactory available
// for use via the TLSCertStoreFlags command line flags.
func RegisterCertStoreFactory(cache CertStoreFactory) {
	storesMu.Lock()
	defer storesMu.Unlock()
	stores[cache.Type()] = cache
}

// RegisteredCertStores returns the list of currently registered certificate
// stores.
func RegisteredCertStores() []string {
	storesMu.Lock()
	defer storesMu.Unlock()
	names := make([]string, 0, len(stores))
	for k, c := range stores {
		names = append(names, fmt.Sprintf("%s: %s", k, c.Describe()))
	}
	return names
}
