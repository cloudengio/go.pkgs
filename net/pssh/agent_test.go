// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"bytes"
	"testing"
	"time"
)

// wantListing checks that the server was able to list the user's key, which
// is held by the client's agent, through a forwarded channel.
func wantListing(t *testing.T, af *agentForwarding, srv *sshServer) {
	t.Helper()
	select {
	case l := <-af.listed:
		if l.err != nil {
			t.Fatalf("listing the forwarded agent: %v", l.err)
		}
		if len(l.keys) != 1 || !bytes.Equal(l.keys[0].Marshal(), srv.userSigner.PublicKey().Marshal()) {
			t.Fatalf("got %v, want the user's key", l.keys)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the forwarded agent to be used")
	}
}

func TestAgentForwardingRun(t *testing.T) {
	af := observeAgentForwarding(t)
	c, srv, _ := runFixture(t, WithAgentForwarding(true))
	if err := c.Run(timeoutCtx(t), "true"); err != nil {
		t.Fatal(err)
	}
	wantListing(t, af, srv)
}

func TestAgentForwardingRunInteractive(t *testing.T) {
	af := observeAgentForwarding(t)
	c, srv, _ := runFixture(t, WithAgentForwarding(true))
	p := newPipes(t)
	if err := c.RunInteractive(timeoutCtx(t), "true", p.options()...); err != nil {
		t.Fatal(err)
	}
	wantListing(t, af, srv)
}

func TestAgentForwardingNotRequested(t *testing.T) {
	af := observeAgentForwarding(t)
	c, _, _ := runFixture(t)
	if err := c.Run(timeoutCtx(t), "true"); err != nil {
		t.Fatal(err)
	}
	if got := af.requests.Load(); got != 0 {
		t.Errorf("got %d requests for agent forwarding, want none", got)
	}
}

func TestAgentForwardingRefused(t *testing.T) {
	af := observeAgentForwarding(t)
	af.deny.Store(true)
	c, _, _ := runFixture(t, WithAgentForwarding(true))
	// The command runs regardless, as it does for ssh -A.
	if err := c.Run(timeoutCtx(t), "true"); err != nil {
		t.Fatal(err)
	}
	if got := af.requests.Load(); got != 1 {
		t.Errorf("got %d requests for agent forwarding, want 1", got)
	}
}

func TestAgentForwardingAfterReconnect(t *testing.T) {
	af := observeAgentForwarding(t)
	connected := make(chan struct{}, 8)
	c, srv, _ := runFixture(t, WithAgentForwarding(true), WithOnConnect(func(*Client) {
		connected <- struct{}{}
	}))
	ctx := timeoutCtx(t)
	if err := c.Run(ctx, "true"); err != nil {
		t.Fatal(err)
	}
	wantListing(t, af, srv)
	select {
	case <-connected:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	srv.drop()
	select {
	case <-connected:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := c.Run(ctx, "true"); err != nil {
		t.Fatal(err)
	}
	wantListing(t, af, srv)
}
