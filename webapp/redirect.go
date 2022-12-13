// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RedirectToHTTPSHandlerFunc is a http.HandlerFunc that will redirect
// to the specified port but using https as the scheme. Install it on
// port 80 to redirect all http requests to https on tlsPort. tlsPort
// defaults to 443. If acmeRedirect is specified then acme HTTP-01 challenges
// will be redirected to that URL.
func RedirectToHTTPSHandlerFunc(tlsPort string, acmeRedirectHost *url.URL) http.HandlerFunc {
	if len(tlsPort) == 0 {
		tlsPort = "443"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := strings.Replace(r.URL.Path, "\n", "", -1)
		escaped = strings.Replace(escaped, "\r", "", -1)
		log.Printf("http %v: %v: %v", r.Host, escaped, acmeRedirectHost != nil)
		if acmeRedirectHost != nil && strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			nrl := url.URL{
				Scheme: "http",
				Host:   acmeRedirectHost.Host,
				Path:   r.URL.Path,
			}
			log.Printf("acme redirect to %s", nrl.String())
			// redirect to our login servers.
			http.Redirect(w, r, nrl.String(), http.StatusTemporaryRedirect)
			return
		}
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		u := r.URL
		u.Host = net.JoinHostPort(host, tlsPort)
		u.Scheme = "https"
		http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
	})
}

// RedirectPort80 starts an http.Server that will redirect port 80 to the
// specified supplied https port. If acmeRedirect is specified then
// acme HTTP-01 challenges will be redirected to that URL.
func RedirectPort80(ctx context.Context, httpsAddr string, acmeRedirectHost string) error {
	_, tlsPort, _ := net.SplitHostPort(httpsAddr)
	var au *url.URL
	var err error
	if len(acmeRedirectHost) > 0 {
		au, err = url.Parse(acmeRedirectHost)
		if err != nil {
			return err
		}
		if au.Scheme != "http" {
			return fmt.Errorf("acme redirect must be http")
		}
		_, port, err := net.SplitHostPort(au.Host)
		if err == nil {
			if port != "80" {
				return fmt.Errorf("acme redirect must be to port 80")
			}
		}
		if len(au.Path) > 0 {
			return fmt.Errorf("acmeRedirect should not include a path")
		}
	}

	redirect := RedirectToHTTPSHandlerFunc(tlsPort, au)
	ln, srv, err := NewHTTPServer(":80", redirect)
	if err != nil {
		return err
	}
	go func() {
		err := ServeWithShutdown(ctx, ln, srv, time.Minute)
		if err != nil {
			log.Printf("failed to shutdown http redirect to tls port %s", tlsPort)
		}
	}()
	return nil
}
