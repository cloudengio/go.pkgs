// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"cloudeng.io/logging/ctxlog"
)

// ServeWithShutdown runs srv.ListenAndServe in background and then
// waits for the context to be canceled. It will then attempt to shutdown
// the web server within the specified grace period.
func ServeWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error {
	return serveWithShutdown(ctx, srv, grace, func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			ctxlog.Logger(ctx).Error("server error", "err", err.Error())
		}
	})
}

// ServeTLSWithShutdown is like ServeWithShutdown except for a TLS server.
// Note that any TLS options must be configured prior to calling this
// function via the TLSConfig field in http.Server.
func ServeTLSWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error {
	if srv.TLSConfig == nil {
		return fmt.Errorf("ServeTLSWithShutdown requires a non-nil TLSConfig in the http.Server")
	}
	return serveWithShutdown(ctx, srv, grace, func() {
		if err := srv.ServeTLS(ln, "", ""); err != nil && err != http.ErrServerClosed {
			ctxlog.Logger(ctx).Error("serveTLS error", "err", err.Error())
		}
	})
}

func serveWithShutdown(ctx context.Context, srv *http.Server, grace time.Duration, fn func()) error {
	go fn()

	<-ctx.Done()
	ctxlog.Logger(ctx).Info("shutting down server", "addr", srv.Addr)
	ctx, cancel := context.WithTimeout(ctx, grace)
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
