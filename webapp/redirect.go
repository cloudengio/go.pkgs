// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"cloudeng.io/logging/ctxlog"
)

// RedirectTarget is a function that given an http.Request returns
// the target URL for the redirect and the HTTP status code to use.
type RedirectTarget func(*http.Request) (string, int)

// Redirect defines a URL path prefix which will be redirected to
// the specified target.
type Redirect struct {
	Prefix string
	Target RedirectTarget
}

// redirectHandler is an http.Handler that will redirect requests
// based on the defined Redirects. If no redirect matches the
// defaultTarget is used. An empty Prefix or Target is treated as "/".
type redirectHandler struct {
	redirects []Redirect
}

// newRedirectHandler creates a RedirectHandler that will redirect
// requests based on the supplied redirects. If no redirect matches
// the defaultTarget is used. The redirects are sorted in order of
// decreasing prefix length so that more specific prefixes are matched
// first. If the defaultTarget is empty then no match results in
// a redirect to the same host as the request but to port 443.
func newRedirectHandler(redirects ...Redirect) http.Handler {
	rh := redirectHandler{
		redirects: slices.Clone(redirects),
	}
	slices.SortFunc(rh.redirects, func(a, b Redirect) int {
		return strings.Compare(b.Prefix, a.Prefix)
	})
	for i, r := range rh.redirects {
		if r.Prefix == "" {
			r.Prefix = "/"
		}
		if r.Target == nil {
			r.Target = func(*http.Request) (string, int) {
				return "/", http.StatusMovedPermanently
			}
		}
		rh.redirects[i] = r
	}
	return rh
}

func (rh redirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, redirect := range rh.redirects {
		if strings.HasPrefix(r.URL.Path, redirect.Prefix) {
			t, c := redirect.Target(r)
			http.Redirect(w, r, t, c)
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func challengeRewrite(host string, r *http.Request) string {
	nrl := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   r.URL.Path,
	}
	target := nrl.String()
	ctxlog.Info(r.Context(), "redirecting acme challenge", "redirect", target)
	return target
}

// RedirectAcmeHTTP01 returns a Redirect that will redirect
// ACME HTTP-01 challenges to the specified host.
func RedirectAcmeHTTP01(host string) Redirect {
	return Redirect{
		Prefix: "/.well-known/acme-challenge/",
		Target: func(r *http.Request) (string, int) {
			return challengeRewrite(host, r), http.StatusTemporaryRedirect
		},
	}
}

// RedirectToHTTPSPort returns a Redirect that will redirect
// to the specified address using https but with the following defaults:
// - if addr does not contain a host then the host from the request is used
// - if addr does not contain a port then port 443 is used.
func RedirectToHTTPSPort(addr string) Redirect {
	host, port := SplitHostPort(addr)
	if len(port) == 0 {
		port = "443"
	}
	return Redirect{
		Prefix: "/",
		Target: func(r *http.Request) (string, int) {
			h, _ := SplitHostPort(r.Host)
			if len(h) == 0 {
				h = host
			}
			u := r.URL
			u.Host = net.JoinHostPort(h, port)
			u.Scheme = "https"
			return u.String(), http.StatusMovedPermanently
		},
	}
}

func RedirectHandler(redirects ...Redirect) http.Handler {
	return newRedirectHandler(redirects...)
}

// RedirectPort80 starts an http.Server that will redirect port 80 to the
// specified supplied https port. If acmeRedirect is specified then
// acme HTTP-01 challenges will be redirected to that URL.
// The server will run in the background until the supplied context
// is canceled.
func RedirectPort80(ctx context.Context, redirects ...Redirect) error {
	rh := newRedirectHandler(redirects...)
	ln, srv, err := NewHTTPServer(ctx, ":80", rh)
	if err != nil {
		return err
	}
	go func() {
		if err := ServeWithShutdown(ctx, ln, srv, time.Minute); err != nil {
			ctxlog.Logger(ctx).Error("error from http redirect server", "addr", srv.Addr, "err", err.Error())
		}
	}()
	return nil
}
