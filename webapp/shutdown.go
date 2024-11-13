// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// ServeWithShutdown runs srv.ListenAndServe in background and then
// waits for the context to be canceled. It will then attempt to shutdown
// the web server within the specified grace period.
func ServeWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error {
	return serveWithShutdown(ctx, srv, grace, func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %s\n", err)
		}
	})
}

// ServeTLSWithShutdown is like ServeWithShutdown except for a TLS server.
// Note that any TLS options must be configured prior to calling this
// function via the TLSConfig field in http.Server.
func ServeTLSWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error {
	return serveWithShutdown(ctx, srv, grace, func() {
		if err := srv.ServeTLS(ln, "", ""); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serveTLS: %s\n", err)
		}
	})
}

func serveWithShutdown(ctx context.Context, srv *http.Server, grace time.Duration, fn func()) error {
	go fn()

	<-ctx.Done()
	log.Printf("stopping..... %v\n", srv.Addr)
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown afer %s: %v", grace, err)
	}
	return nil
}

// NewHTTPServer returns a new *http.Server and a listener whose address defaults
// to ":http".
func NewHTTPServer(addr string, handler http.Handler) (net.Listener, *http.Server, error) {
	return newServer(addr, ":http", handler, nil)

}

// NewTLSServer returns a new *http.Server and a listener whose address defaults
// to ":https".
func NewTLSServer(addr string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error) {
	return newServer(addr, ":https", handler, cfg)
}

func newServer(addr, def string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error) {
	if len(addr) == 0 {
		addr = def
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	return ln, &http.Server{Addr: addr, Handler: handler, TLSConfig: cfg, ReadHeaderTimeout: time.Minute}, nil
}
