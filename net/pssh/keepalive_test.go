// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"testing"
	"time"
)

func keepAliveFixture(t *testing.T, interval time.Duration) (*Client, *sshServer) {
	t.Helper()
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")
	c := NewClient(t.Context(), "tcp", srv.addr(), WithUser("auser"),
		WithHostKey(hostSigner.PublicKey()), WithKeepAlive(interval))
	connectInBackground(t.Context(), c, fastBackoff)
	t.Cleanup(c.Close)
	return c, srv
}

func TestKeepAliveHealthyConnection(t *testing.T) {
	c, srv := keepAliveFixture(t, 20*time.Millisecond)
	srv.waitForConnection(t)
	ctx := timeoutCtx(t)
	if err := c.WaitConnected(ctx); err != nil {
		t.Fatal(err)
	}
	// Many intervals pass, during which the answered requests must not
	// cause the connection to be replaced.
	time.Sleep(500 * time.Millisecond)
	select {
	case <-srv.accepted:
		t.Fatal("a healthy connection was replaced")
	default:
	}
	if err := c.Run(ctx, "true"); err != nil {
		t.Fatal(err)
	}
}

func TestKeepAliveSilentConnection(t *testing.T) {
	silentServer.Store(true)
	t.Cleanup(func() { silentServer.Store(false) })
	_, srv := keepAliveFixture(t, 50*time.Millisecond)
	// The server never answers, so each connection is closed by the client
	// and replaced.
	srv.waitForConnection(t)
	srv.waitForConnection(t)
}

func TestNoKeepAliveByDefault(t *testing.T) {
	silentServer.Store(true)
	t.Cleanup(func() { silentServer.Store(false) })
	_, _, _ = runFixture(t)
	time.Sleep(300 * time.Millisecond)
}
