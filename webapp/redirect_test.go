// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp"
)

func TestRedirectToHTTPSHandlerFunc(t *testing.T) {
	acmeRedirectURL, err := url.Parse("http://acme-handler.example.com")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name               string
		reqURL             string
		tlsPort            string
		acmeRedirect       *url.URL
		expectedStatusCode int
		expectedLocation   string
		expectedLog        string
	}{
		{
			name:               "Redirect to specific port",
			reqURL:             "http://example.com/path/to/page",
			tlsPort:            "8443",
			acmeRedirect:       nil,
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/path/to/page",
			expectedLog:        `level=INFO msg="redirecting to https" redirect=https://example.com:8443/path/to/page`,
		},
		{
			name:               "Redirect to default port 443",
			reqURL:             "http://example.com/another/page",
			tlsPort:            "", // Default
			acmeRedirect:       nil,
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:443/another/page",
			expectedLog:        `level=INFO msg="redirecting to https" redirect=https://example.com:443/another/page`,
		},
		{
			name:               "ACME challenge redirect",
			reqURL:             "http://example.com/.well-known/acme-challenge/some-token",
			tlsPort:            "8443",
			acmeRedirect:       acmeRedirectURL,
			expectedStatusCode: http.StatusTemporaryRedirect,
			expectedLocation:   "http://acme-handler.example.com/.well-known/acme-challenge/some-token",
			expectedLog:        `level=INFO msg="redirecting acme challenge" redirect=http://acme-handler.example.com/.well-known/acme-challenge/some-token`,
		},
		{
			name:               "Standard redirect when ACME is configured",
			reqURL:             "http://example.com/not-an-acme-challenge",
			tlsPort:            "8443",
			acmeRedirect:       acmeRedirectURL,
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/not-an-acme-challenge",
			expectedLog:        `level=INFO msg="redirecting to https" redirect=https://example.com:8443/not-an-acme-challenge`,
		},
		{
			name:               "Request with host and port",
			reqURL:             "http://example.com:80/path",
			tlsPort:            "8443",
			acmeRedirect:       nil,
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/path",
			expectedLog:        `level=INFO msg="redirecting to https" redirect=https://example.com:8443/path`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))
			ctx := ctxlog.WithLogger(context.Background(), logger)

			req := httptest.NewRequest("GET", tc.reqURL, nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler := webapp.RedirectToHTTPSHandlerFunc(tc.tlsPort, tc.acmeRedirect)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatusCode)
			}

			if location := rr.Header().Get("Location"); location != tc.expectedLocation {
				t.Errorf("handler returned wrong redirect location: got %q want %q",
					location, tc.expectedLocation)
			}

			if got := strings.TrimSpace(logBuf.String()); !strings.Contains(got, tc.expectedLog) {
				t.Errorf("log output missing expected string:\n  got: %v\n want: %v", got, tc.expectedLog)
			}
		})
	}
}

func TestRedirectPort80(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel() // Ensure context is always canceled to avoid leaking goroutines.

	testCases := []struct {
		name             string
		httpsAddr        string
		acmeRedirectHost string
		expectErr        string
	}{
		{
			name:             "Valid ACME redirect",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "http://acme.example.com",
			expectErr:        "",
		},
		{
			name:             "Valid ACME redirect with port 80",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "http://acme.example.com:80",
			expectErr:        "",
		},
		{
			name:             "Invalid ACME redirect scheme",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "https://acme.example.com",
			expectErr:        "acme redirect must be http",
		},
		{
			name:             "Invalid ACME redirect port",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "http://acme.example.com:8080",
			expectErr:        "acme redirect must be to port 80",
		},
		{
			name:             "ACME redirect with path",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "http://acme.example.com/some/path",
			expectErr:        "acmeRedirect should not include a path",
		},
		{
			name:             "Invalid ACME redirect URL",
			httpsAddr:        "example.com:443",
			acmeRedirectHost: "http://[::1]:namedport",
			expectErr:        `invalid port`, // Simplified check
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// We test the validation logic of RedirectPort80, but do not actually
			// start the server, as it would require root privileges to bind to port 80.
			err := webapp.RedirectPort80(ctx, tc.httpsAddr, tc.acmeRedirectHost)

			if tc.expectErr == "" {
				if err != nil {
					// An error is expected here because we can't bind to port 80.
					// We are only testing the parameter validation part.
					// A successful parameter validation will lead to a listen error.
					if !strings.Contains(err.Error(), "listen tcp :80") {
						t.Errorf("unexpected error: got %v", err)
					}
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, but got nil", tc.expectErr)
				} else if !strings.Contains(err.Error(), tc.expectErr) {
					t.Errorf("expected error to contain %q, but got %q", tc.expectErr, err.Error())
				}
			}
		})
	}
}
