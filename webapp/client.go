// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"slices"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/net/http/httptracing"
	"cloudeng.io/webapp/devtest"
)

// HTTPClientOption is used to configure an HTTP client.
type HTTPClientOption func(o *httpClientOptions)

// WithCustomCA configures the HTTP client to use the specified
// custom CA PEM data as a root CA.
func WithCustomCA(caPem string) HTTPClientOption {
	return func(o *httpClientOptions) {
		o.caPem = caPem
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
	caPem       string
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
	if caPem := options.caPem; caPem != "" {
		ctxlog.Logger(ctx).Warn("services.NewHTTPClient: using custom root CA pool containing", "ca", caPem)
		rootCAs, err := devtest.CertPoolForTesting(caPem)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain cert pool containing %v: %w", caPem, err)
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
