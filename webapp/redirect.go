// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"
)

// RedirectToHTTPSHandlerFunc is a http.HandlerFunc that will redirect
// to the specified port but using https as the scheme. Install it on
// port 80 to redirect all http requests to https on tlsPort. tlsPort
// defaults to 443.
func RedirectToHTTPSHandlerFunc(tlsPort string) http.HandlerFunc {
	if len(tlsPort) == 0 {
		tlsPort = "443"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// RedirectPort80 starts an http.Server that will redirect port 80 to
// the specified supplied https port.
func RedirectPort80(ctx context.Context, httpsAddr string) error {
	_, tlsPort, _ := net.SplitHostPort(httpsAddr)
	redirect := RedirectToHTTPSHandlerFunc(tlsPort)
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
