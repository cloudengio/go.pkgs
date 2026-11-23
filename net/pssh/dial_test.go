// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestDialContext verifies that a connection to a service reachable from the
// server is carried by the connection to it, which is what a proxy host is.
func TestDialContext(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")
	echo := echoServer(t)

	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"),
		WithHostKey(hostSigner.PublicKey()))
	done := connectInBackground(t.Context(), c, fastBackoff)
	if err := c.WaitConnected(t.Context()); err != nil {
		t.Fatal(err)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", echo)
	conn, err := c.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	const msg = "over the connection"
	if _, err := io.WriteString(conn, msg); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if got, want := string(reply), msg; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	conn.Close()

	c.Close()
	if err := waitForResult(t, done); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
	// A closed client has no connection to carry one, and will not acquire one.
	if _, err := c.DialContext(t.Context(), "tcp", addr); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
}

// TestDialContextWaitsForConnection verifies that a client which is not
// connected is waited for rather than reported, as it is by Run, and that the
// wait is bounded by the context.
func TestDialContextWaitsForConnection(t *testing.T) {
	_, hostSigner := newKey(t)
	// No ConnectAndWait, so this client never connects.
	c := NewClient(t.Context(), "tcp", testAddr, WithHostKey(hostSigner.PublicKey()))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.DialContext(ctx, "tcp", "127.0.0.1:1"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want %v", err, context.Canceled)
	}
}

// TestWithDialer verifies that the supplied dialer is what the handshake runs
// over, and that it is asked for the client's own network and address: the
// server is reached by whatever means the dialer uses, but it is still the
// server that was named.
func TestWithDialer(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")

	var mu sync.Mutex
	var dialed []string
	// testAddr is not a host this test can reach, so a connection to it is
	// proof that the dialer below was used to make it.
	c := NewClient(t.Context(), "tcp", testAddr,
		WithUser("auser"),
		WithHostKey(hostSigner.PublicKey()),
		WithDialer(func(ctx context.Context, network, addr string) (net.Conn, error) {
			mu.Lock()
			dialed = append(dialed, network+" "+addr)
			mu.Unlock()
			return new(net.Dialer).DialContext(ctx, "tcp", srv.addr())
		}))
	done := connectInBackground(t.Context(), c, fastBackoff)
	if err := c.WaitConnected(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv.waitForConnection(t)

	mu.Lock()
	got := dialed
	mu.Unlock()
	if want := []string{"tcp " + testAddr}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %v, want %v", got, want)
	}

	c.Close()
	if err := waitForResult(t, done); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
}

// TestWithDialerTimeout verifies that the dial timeout bounds a supplied
// dialer, which has no timeout of its own in the way that net.Dialer does,
// and that exceeding it is a failure that is retried.
func TestWithDialerTimeout(t *testing.T) {
	userPriv, _ := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)

	type attempt struct {
		err       error
		retryable bool
	}
	attempts := make(chan attempt, 1)
	c := NewClient(t.Context(), "tcp", testAddr,
		WithUser("auser"),
		WithHostKey(hostSigner.PublicKey()),
		WithDialTimeout(50*time.Millisecond),
		// A dialer that never connects of its own accord, as one waiting on a
		// proxy host that is not connected does.
		WithDialer(func(ctx context.Context, _, _ string) (net.Conn, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
		WithOnAttemptFailed(func(err error, retryable bool) {
			select {
			case attempts <- attempt{err: err, retryable: retryable}:
			default: // Only the first attempt is of interest.
			}
		}))
	done := connectInBackground(t.Context(), c, fastBackoff)

	select {
	case a := <-attempts:
		if !errors.Is(a.err, context.DeadlineExceeded) {
			t.Errorf("got %v, want %v", a.err, context.DeadlineExceeded)
		}
		if !a.retryable {
			t.Error("a dial that timed out should be retryable")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the dial was not bounded by the dial timeout")
	}

	c.Close()
	if err := waitForResult(t, done); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
}

// TestProxyHost verifies that a server is reached over the connection that the
// proxy host maintains, and that the two recover together: the connection to
// the server is carried by the proxy host's, so losing that loses it too, and
// it is established again once the proxy host has reconnected.
func TestProxyHost(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, proxyKey := newKey(t)
	_, targetKey := newKey(t)
	serveAgent(t, userPriv)
	proxySrv := newSSHServer(t, proxyKey, userSigner, "auser")
	targetSrv := newSSHServer(t, targetKey, userSigner, "auser")

	proxy := NewClient(t.Context(), "tcp", proxySrv.addr(),
		WithUser("auser"),
		WithHostKey(proxyKey.PublicKey()))
	// The target is verified by its own host key, since only the transport
	// to it changes.
	target := NewClient(t.Context(), "tcp", targetSrv.addr(),
		WithUser("auser"),
		WithHostKey(targetKey.PublicKey()),
		WithProxyHost(proxy))

	proxyDone := connectInBackground(t.Context(), proxy, fastBackoff)
	targetDone := connectInBackground(t.Context(), target, fastBackoff)

	if err := target.WaitConnected(t.Context()); err != nil {
		t.Fatal(err)
	}
	proxySrv.waitForConnection(t)
	targetSrv.waitForConnection(t)

	proxySrv.drop()
	// The proxy host reconnects, and the connection it carries is made again
	// over its new connection.
	proxySrv.waitForConnection(t)
	targetSrv.waitForConnection(t)
	if err := target.WaitConnected(t.Context()); err != nil {
		t.Fatal(err)
	}

	target.Close()
	proxy.Close()
	if err := waitForResult(t, targetDone); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
	if err := waitForResult(t, proxyDone); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
}

// TestProxyHostClosed verifies that a proxy host which has been closed, as it is
// when its own ConnectAndWait has returned, ends the client that connects
// through it rather than being retried indefinitely. The backoff would make a
// retry take longer than this test waits, so returning at all is proof that
// there was none.
func TestProxyHostClosed(t *testing.T) {
	userPriv, _ := newKey(t)
	_, proxyKey := newKey(t)
	_, targetKey := newKey(t)
	serveAgent(t, userPriv)

	proxy := NewClient(t.Context(), "tcp", "proxy.example.com:22",
		WithUser("auser"),
		WithHostKey(proxyKey.PublicKey()))
	proxy.Close()

	target := NewClient(t.Context(), "tcp", testAddr,
		WithUser("auser"),
		WithHostKey(targetKey.PublicKey()),
		WithProxyHost(proxy))

	err := waitForResult(t, connectInBackground(t.Context(), target, slowBackoff))
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want %v", err, ErrClientClosed)
	}
	// The failure is the proxy host's, which the error has to say.
	if got, want := fmt.Sprint(err), "via proxy host proxy.example.com:22"; !strings.Contains(got, want) {
		t.Errorf("error %q does not mention %q", got, want)
	}
}
