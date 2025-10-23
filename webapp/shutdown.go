// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/sync/errgroup"
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

// NewHTTPServerOnly returns a new *http.Server whose address defaults
// to ":http" and with it's BaseContext set to the supplied context.
// ErrorLog is set to log errors via the ctxlog package.
func NewHTTPServerOnly(ctx context.Context, addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: time.Minute,
		ErrorLog:          ctxlog.NewLogLogger(ctx, slog.LevelError),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
}

// NewTLSServerOnly returns a new *http.Server whose address defaults
// to ":https" and with it's BaseContext set to the supplied context and
// TLSConfig set to the supplied config.
// ErrorLog is set to log errors via the ctxlog package.
func NewTLSServerOnly(ctx context.Context, addr string, handler http.Handler, cfg *tls.Config) *http.Server {
	hs := NewHTTPServerOnly(ctx, addr, handler)
	hs.TLSConfig = cfg
	return hs
}

// NewHTTPServer returns a new *http.Server using ParseAddrPortDefaults(addr, "http")
// to obtain the address to listen on and NewHTTPServerOnly to create the server.
func NewHTTPServer(ctx context.Context, addr string, handler http.Handler) (net.Listener, *http.Server, error) {
	return newServer(ctx, ParseAddrPortDefaults(addr, "http"), handler, nil)
}

// NewTLSServer returns a new *http.Server using ParseAddrPortDefaults(addr, "https")
// to obtain the address to listen on and NewTLSServerOnly to create the server.
func NewTLSServer(ctx context.Context, addr string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error) {
	return newServer(ctx, ParseAddrPortDefaults(addr, "https"), handler, cfg)
}

func newServer(ctx context.Context, addr string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	if cfg == nil {
		srv := NewHTTPServerOnly(ctx, addr, handler)
		return ln, srv, nil
	}
	srv := NewTLSServerOnly(ctx, addr, handler, cfg)
	return ln, srv, nil
}

// ParseAddrPortDefaults parses addr and returns an address:port string.
// If addr is empty then ":<port>" is returned.
// If addr contains a colon then it is assumed to be a valid
// address:port and returned as is.
// If addr contains no colon then it is assumed to be a port if
// it can be parsed as an integer, in which case ":" is prepended.
// Otherwise it is assumed to be a host, in which case port :<port>
// is appended. Thus "localhost" becomes "localhost:<port>".
func ParseAddrPortDefaults(addr, port string) string {
	port = strings.TrimPrefix(port, ":")
	if len(addr) == 0 {
		return ":" + port
	}
	if idx := strings.Index(addr, ":"); idx >= 0 {
		return addr
	}
	// could be a host, or a port.
	if _, err := strconv.Atoi(addr); err == nil {
		// port
		return ":" + addr
	}
	// host
	return net.JoinHostPort(addr, port)
}

// WaitForServers waits for all supplied addresses to be available
// by attempting to open a TCP connection to each address at the
// specified interval.
func WaitForServers(ctx context.Context, interval time.Duration, addrs ...string) error {
	if len(addrs) == 1 {
		return ping(ctx, interval, addrs[0])
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, addr := range addrs {
		g.Go(func() error {
			return ping(ctx, interval, addr)
		})
	}
	return g.Wait()
}

func ping(ctx context.Context, interval time.Duration, addr string) error {
	for {
		ctxlog.Logger(ctx).Info("ping: server", "addr", addr)
		_, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			ctxlog.Info(ctx, "ping: server timeout", "addr", addr, "duration", interval.String())

		}
	}
}
