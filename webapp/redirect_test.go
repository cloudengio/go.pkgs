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
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp"
)

func TestRedirectHandler(t *testing.T) {
	acmeRedirectURL, err := url.Parse("http://acme-handler.example.com")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name               string
		reqURL             string
		redirects          []webapp.Redirect
		expectedStatusCode int
		expectedLocation   string
		expectedLog        string
	}{
		{
			name:   "Redirect to specific port",
			reqURL: "http://example.com/path/to/page",
			redirects: []webapp.Redirect{
				webapp.RedirectToHTTPSPort(":8443"),
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/path/to/page",
		},
		{
			name:   "Redirect to default port 443",
			reqURL: "http://example.com/another/page",
			redirects: []webapp.Redirect{
				webapp.RedirectToHTTPSPort(""),
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:443/another/page",
		},
		{
			name:   "ACME challenge redirect",
			reqURL: "http://example.com/.well-known/acme-challenge/some-token",
			redirects: []webapp.Redirect{
				webapp.RedirectAcmeHTTP01(acmeRedirectURL.Host),
				webapp.RedirectToHTTPSPort(":8443"),
			},
			expectedStatusCode: http.StatusTemporaryRedirect,
			expectedLocation:   "http://acme-handler.example.com/.well-known/acme-challenge/some-token",
			expectedLog:        `level=INFO msg="redirecting acme challenge" redirect=http://acme-handler.example.com/.well-known/acme-challenge/some-token`,
		},
		{
			name:   "Standard redirect when ACME is configured",
			reqURL: "http://example.com/not-an-acme-challenge",
			redirects: []webapp.Redirect{
				webapp.RedirectAcmeHTTP01(acmeRedirectURL.Host),
				webapp.RedirectToHTTPSPort(":8443"),
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/not-an-acme-challenge",
		},
		{
			name:   "Request with host and port",
			reqURL: "http://example.com:80/path",
			redirects: []webapp.Redirect{
				webapp.RedirectToHTTPSPort(":8443"),
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://example.com:8443/path",
		},
		{
			name:   "No matching redirect",
			reqURL: "http://example.com/no-match",
			redirects: []webapp.Redirect{
				{Prefix: "/foo"},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "More specific prefix wins",
			reqURL: "http://example.com/foo/bar",
			redirects: []webapp.Redirect{
				{
					Prefix: "/",
					Target: func(_ *http.Request) (string, int) {
						return "https://catchall.com", http.StatusMovedPermanently
					},
				},
				{
					Prefix: "/foo",
					Target: func(_ *http.Request) (string, int) {
						return "https://foospecific.com", http.StatusMovedPermanently
					},
				},
			},
			expectedStatusCode: http.StatusMovedPermanently,
			expectedLocation:   "https://foospecific.com",
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

			handler := webapp.RedirectHandler(tc.redirects...)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tc.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatusCode)
			}

			if location := rr.Header().Get("Location"); location != tc.expectedLocation {
				t.Errorf("handler returned wrong redirect location: got %q want %q",
					location, tc.expectedLocation)
			}

			if len(tc.expectedLog) > 0 {
				if got := strings.TrimSpace(logBuf.String()); !strings.Contains(got, tc.expectedLog) {
					t.Errorf("log output missing expected string:\n  got: %v\n want: %v", got, tc.expectedLog)
				}
			}
		})
	}
}

func TestRedirectPort80(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := webapp.RedirectPort80(ctx, webapp.RedirectToHTTPSPort(":8443"))
	if err != nil {
		// This may fail on systems where port 80 is privileged.
		// We can't reliably test this everywhere, so we just log it.
		// The important part is that the handler logic is tested above.
		t.Logf("failed to start redirect server on port 80 (this may be expected): %v", err)
		return
	}

	// Give the server a moment to start.
	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("http://127.0.0.1:80/test")
	if err != nil {
		t.Fatalf("failed to make request to redirect server: %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusMovedPermanently; got != want {
		t.Errorf("got status %v, want %v", got, want)
	}

	if got, want := resp.Header.Get("Location"), "https://127.0.0.1:8443/test"; got != want {
		t.Errorf("got location %q, want %q", got, want)
	}

	// The server shuts down when the context is canceled.
	cancel()
	// Give it a moment to shut down.
	time.Sleep(100 * time.Millisecond)

	// Verify it's no longer listening.
	_, err = client.Get("http://127.0.0.1:80/test")
	if err == nil {
		t.Fatal("server did not shut down as expected")
	}
}
