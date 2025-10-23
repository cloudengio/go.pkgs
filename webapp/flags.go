// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"fmt"

	"golang.org/x/crypto/acme/autocert"
)

// TLSCertFlags defines commonly used flags for obtaining TLS/SSL certificates.
// Certificates may be obtained in one of two ways: from a cache of
// certificates, or from local files.
type TLSCertFlags struct {
	CertFile string `subcmd:"tls-cert,,tls certificate file"`
	KeyFile  string `subcmd:"tls-key,,tls private key file"`
}

// Config returns a TLSCertConfig based on the supplied flags.
func (cl TLSCertFlags) Config() TLSCertConfig {
	return TLSCertConfig{
		CertFile: cl.CertFile,
		KeyFile:  cl.KeyFile,
	}
}

// TLSCertConfig defines configuration for TLS certificates obtained
// from local files.
type TLSCertConfig struct {
	CertFile string `yaml:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
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

// Config returns an HTTPServerConfig based on the supplied flags.
func (cl HTTPServerFlags) Config() HTTPServerConfig {
	return HTTPServerConfig{
		Address:  cl.Address,
		TLSCerts: cl.TLSCertFlags.Config(),
	}
}

// HTTPServerConfig defines configuration for an http server.
type HTTPServerConfig struct {
	Address  string        `yaml:"address,omitempty"`
	TLSCerts TLSCertConfig `yaml:"tls_certs,omitempty"`
}

// TLSConfigFromConfig creates a tls.Config based on the supplied config, which
// may require obtaining certificates directly from pem files or from a
// possibly remote certificate store using TLSConfigUsingCertStore. If
// a cert store is specified then the getStoreOpts function
// may be used to obtain additional options for the store. A cache
// is then created to from that store using the supplied cacheOpts.
func TLSConfigFromConfig(ctx context.Context, cl HTTPServerConfig) (*tls.Config, error) {
	return TLSConfigUsingCertFiles(cl.TLSCerts.CertFile, cl.TLSCerts.KeyFile)
}

// PreferredCipherSuites is the list of preferred cipher suites
// for tls.Config instances created by this package.
var PreferredCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
}

// PreferredCurves is the list of preferred elliptic curves
// for tls.Config instances created by this package.
var PreferredCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
}

// PreferredTLSMinVersion is the preferred minimum TLS version
// for tls.Config instances created by this package.
const PreferredTLSMinVersion = tls.VersionTLS13

// PreferredSignatureSchemes is the list of preferred signature schemes
// generally used for obtainint TLS certificates.
var PreferredSignatureSchemes = []tls.SignatureScheme{
	tls.ECDSAWithP256AndSHA256,
	tls.ECDSAWithP384AndSHA384,
	tls.ECDSAWithP521AndSHA512,
}

// TLSConfigUsingCertStore returns a tls.Config configured with the
// certificate obtained from the specified certificate store accessed
// via a CertServingCache created with the supplied options.
func TLSConfigUsingCertStore(ctx context.Context, store autocert.Cache, cacheOpts ...CertServingCacheOption) (*tls.Config, error) {
	return &tls.Config{
		GetCertificate:   NewCertServingCache(ctx, store, cacheOpts...).GetCertificate,
		MinVersion:       PreferredTLSMinVersion,
		CipherSuites:     PreferredCipherSuites,
		CurvePreferences: PreferredCurves,
	}, nil
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
		Certificates:     []tls.Certificate{cert},
		MinVersion:       PreferredTLSMinVersion,
		CipherSuites:     PreferredCipherSuites,
		CurvePreferences: PreferredCurves,
	}, nil
}
