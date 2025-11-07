// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"slices"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/net/http/httptracing"
)

// HTTPClientOption is used to configure an HTTP client.
type HTTPClientOption func(o *httpClientOptions)

// WithCustomCAPEMFile configures the HTTP client to use the specified
// custom CA PEM data as a root CA.
func WithCustomCAPEMFile(caPEMFile string) HTTPClientOption {
	return func(o *httpClientOptions) {
		o.caPEMFile = caPEMFile
	}
}

// WithCustomCAPool configures the HTTP client to use the specified
// custom CA pool. It takes precedence over WithCustomCAPEMFile.
func WithCustomCAPool(caPool *x509.CertPool) HTTPClientOption {
	return func(o *httpClientOptions) {
		o.caPool = caPool
	}
}

// WithTracingTransport configures the HTTP client to use a tracing
// round tripper with the specified options.
func WithTracingTransport(to ...httptracing.TraceRoundtripOption) HTTPClientOption {
	return func(o *httpClientOptions) {
		o.tracingOpts = slices.Clone(to)
	}
}

type httpClientOptions struct {
	caPEMFile   string
	caPool      *x509.CertPool
	tracingOpts []httptracing.TraceRoundtripOption
}

// NewHTTPClient creates a new HTTP client configured according to the specified options.
func NewHTTPClient(ctx context.Context, opts ...HTTPClientOption) (*http.Client, error) {
	options := &httpClientOptions{}
	for _, opt := range opts {
		opt(options)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			CipherSuites: PreferredCipherSuites,
		}}
	if options.caPool != nil {
		ctxlog.Logger(ctx).Warn("services.NewHTTPClient: using custom root CA pool")
		transport.TLSClientConfig.RootCAs = options.caPool
	} else if caPEMFile := options.caPEMFile; caPEMFile != "" {
		ctxlog.Logger(ctx).Warn("services.NewHTTPClient: using custom root CA pool containing", "ca", caPEMFile)
		rootCAs, err := certPool(caPEMFile)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain cert pool containing %v: %w", caPEMFile, err)
		}
		transport.TLSClientConfig.RootCAs = rootCAs
	}

	httpClient := &http.Client{
		Transport: transport,
	}
	if len(options.tracingOpts) > 0 {
		trt := httptracing.NewTracingRoundTripper(httpClient.Transport, options.tracingOpts...)
		httpClient.Transport = trt
	}
	return httpClient, nil
}

func certPool(pemFile string) (*x509.CertPool, error) {
	rootCAs := x509.NewCertPool()
	certs, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file %q: %w", pemFile, err)
	}
	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		return nil, fmt.Errorf("no certs appended from %q", pemFile)
	}
	return rootCAs, nil
}
