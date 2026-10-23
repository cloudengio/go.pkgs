// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package netutil_test

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/net/netutil"
)

// Dialer is usable wherever a net.Dialer's Dial or DialContext is.
var (
	_ func(string, string) (net.Conn, error)                  = (&netutil.Dialer{}).Dial
	_ func(context.Context, string, string) (net.Conn, error) = (&netutil.Dialer{}).DialContext
)

// echoListener returns a listener on loopback that closes everything it
// accepts, and its address.
func echoListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	return ln.Addr().String()
}

func TestDialer(t *testing.T) {
	addr := echoListener(t)

	// The zero value dials as a net.Dialer does, by both methods.
	var d netutil.Dialer
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	if got := conn.RemoteAddr().String(); got != addr {
		t.Errorf("got %v, want %v", got, addr)
	}
	_ = conn.Close()

	conn, err = d.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	_ = conn.Close()
}

func TestDialerRecoveryIsOptIn(t *testing.T) {
	addr := echoListener(t)

	// The zero value dials as a net.Dialer does, and says so: the field that
	// enables the recovery is off until a caller sets it.
	var d netutil.Dialer
	if d.RecoverNoRouteToHost {
		t.Error("the recovery should be off by default")
	}
	conn, err := d.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	_ = conn.Close()

	// Enabling it changes nothing for a connection that succeeds, which is
	// every connection but the one it is for.
	d.RecoverNoRouteToHost = true
	conn, err = d.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	_ = conn.Close()
}

func TestDialerEmbeddedFields(t *testing.T) {
	addr := echoListener(t)

	// The embedded net.Dialer is what dials, so its fields apply: a chosen
	// source address is used as given rather than recovered from.
	d := netutil.Dialer{Dialer: net.Dialer{LocalAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}}}
	conn, err := d.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("DialContext: %v", err)
	}
	defer conn.Close()
	if got := conn.LocalAddr().(*net.TCPAddr).IP; !got.IsLoopback() {
		t.Errorf("got %v, want the configured loopback source address", got)
	}
}

func TestDialerPreservesErrors(t *testing.T) {
	// A port that nothing is listening on: the error must be the one the
	// system gave, since recovering from it elsewhere would only obscure it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	var d netutil.Dialer
	conn, err := d.DialContext(t.Context(), "tcp", addr)
	if err == nil {
		conn.Close()
		t.Fatal("expected an error")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Errorf("got %v, want a refused connection", err)
	}
}

func TestDialerContextCancelled(t *testing.T) {
	addr := echoListener(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var d netutil.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		conn.Close()
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestDialerTimeout(t *testing.T) {
	// An address in a range reserved for documentation, which is either
	// unroutable or routed nowhere: whichever error it yields must arrive
	// within the timeout rather than the system's own.
	d := netutil.Dialer{Dialer: net.Dialer{Timeout: 250 * time.Millisecond}}
	start := time.Now()
	conn, err := d.DialContext(t.Context(), "tcp", "192.0.2.1:9")
	if err == nil {
		conn.Close()
		t.Skip("192.0.2.1 is reachable on this network")
	}
	// The recovery path may try several interfaces, each with the timeout,
	// so this bounds the total rather than a single attempt.
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("took %v, the configured timeout was not applied", elapsed)
	}
}
