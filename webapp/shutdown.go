// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"errors"
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
	return serveWithShutdown(ctx, srv, grace, func() error {
		return srv.Serve(ln)
	})
}

// ServeTLSWithShutdown is like ServeWithShutdown except for a TLS server.
// Note that any TLS options must be configured prior to calling this
// function via the TLSConfig field in http.Server.
func ServeTLSWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error {
	if srv.TLSConfig == nil {
		return fmt.Errorf("ServeTLSWithShutdown requires a non-nil TLSConfig in the http.Server")
	}
	return serveWithShutdown(ctx, srv, grace, func() error {
		return srv.ServeTLS(ln, "", "")
	})
}

func serveWithShutdown(ctx context.Context, srv *http.Server, grace time.Duration, fn func() error) error {
	serveErrCh := make(chan error, 1)
	go func() {
		err := fn()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- err
			return
		}
		serveErrCh <- nil
		close(serveErrCh)
	}()

	select {
	case err := <-serveErrCh:
		if err != nil {
			return fmt.Errorf("server %v, unexpected error %w", srv.Addr, err)
		}
		return nil
	case <-ctx.Done():
		ctxlog.Logger(ctx).Info("server being shut down", "addr", srv.Addr, "grace", grace)
	}

	// Use a new context tree for the shutdown, since the original
	// was only intended to signal starting the shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server running on %v, shutdown failed %s: %w", srv.Addr, grace, err)
	}
	select {
	case err := <-serveErrCh:
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
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
