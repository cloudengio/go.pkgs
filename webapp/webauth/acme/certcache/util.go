// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package certcache

import (
	"context"
	"crypto/tls"
	"net"

	"cloudeng.io/webapp"
	"golang.org/x/crypto/acme/autocert"
)

// WrapHostPolicyNoPort wraps an existing autocert.HostPolicy to strip
// any port information from the host before passing it to the existing
// policy. This is required when running in a test environment where
// well-known/hardwired ports (80, 443) are not used.
func WrapHostPolicyNoPort(existing autocert.HostPolicy) autocert.HostPolicy {
	return func(ctx context.Context, host string) error {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
		if err := existing(ctx, host); err != nil {
			return err
		}
		return nil
	}
}

// RefreshCertificate attempts to refresh the certificate for the specified
// host using the provided autocert.Manager by simulating a TLS ClientHello
// for the specified host. It prefers to use the PreferredCipherSuites and
// PreferredSignatureSchemes defined in webapp package to force the use
// of ECDSA certificates rather than RSA.
func RefreshCertificate(_ context.Context, mgr *autocert.Manager, host string) (*tls.Certificate, error) {
	hello := tls.ClientHelloInfo{
		ServerName:       host,
		CipherSuites:     webapp.PreferredCipherSuites,
		SignatureSchemes: webapp.PreferredSignatureSchemes,
	}
	return mgr.GetCertificate(&hello)
}
