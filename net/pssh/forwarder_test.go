// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"
)

// echoServer runs a server that echoes back whatever is sent to it, standing
// in for a service reachable from the far side of the ssh connection. It
// returns the port it listens on.
func echoServer(t *testing.T) int {
	t.Helper()
	return echoServerOn(t, 0)
}

// echoServerOn is echoServer bound to a specific port, so that a service can
// be started on a port that a forward is already pointing at.
func echoServerOn(t *testing.T, port int) int {
	t.Helper()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn) //nolint:errcheck
			}()
		}
	}()
	return l.Addr().(*net.TCPAddr).Port
}

// freePort returns a port that nothing is listening on.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// echoThrough sends a message to the local end of a forward and returns what
// comes back, which has travelled to the echo server over the ssh connection
// and back.
func echoThrough(t *testing.T, port int, msg string) string {
	t.Helper()
	conn, reply := openEcho(t, port, msg)
	conn.Close()
	return reply
}

// openEcho dials the local end of a forward, sends msg and waits for it to
// come back, returning the reply and the connection still open. A reply
// proves the whole path is established, which dialling alone does not.
func openEcho(t *testing.T, port int, msg string) (net.Conn, string) {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, msg); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	return conn, string(reply)
}

func newForwarder(t *testing.T, forwards ...forward) *forwarder {
	t.Helper()
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithHostKey(hostSigner.PublicKey()))
	return &forwarder{
		client:   dialSSH(t, c, hostSigner, userSigner),
		forwards: forwards,
		logger:   slog.New(slog.DiscardHandler),
	}
}

// TestForwarderMultiple verifies that several forwards run at once. Each
// accept loop runs in its own goroutine, so forwardAll must return with all of
// them established rather than blocking in the first.
func TestForwarderMultiple(t *testing.T) {
	firstEcho, secondEcho := echoServer(t), echoServer(t)
	first, second := freePort(t), freePort(t)

	f := newForwarder(t,
		forward{localPort: first, remotePort: firstEcho},
		forward{localPort: second, remotePort: secondEcho})
	if err := f.forwardAll(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Both must carry traffic, and to the right place: each forward names a
	// different echo server.
	if got, want := echoThrough(t, first, "first"), "first"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := echoThrough(t, second, "second"), "second"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	f.stopAll(t.Context())
}

// TestForwarderStopReleasesListeners verifies that stopping the forwards frees
// their local ports. A connection that is replaced re-binds the same ports, so
// a listener left open would make every reconnection fail.
func TestForwarderStopReleasesListeners(t *testing.T) {
	ports := []int{freePort(t), freePort(t)}
	f := newForwarder(t,
		forward{localPort: ports[0], remotePort: echoServer(t)},
		forward{localPort: ports[1], remotePort: echoServer(t)})
	if err := f.forwardAll(t.Context()); err != nil {
		t.Fatal(err)
	}
	// Hold a forwarded connection open, so that stopAll has both a listener
	// and a connection to close. It has to be carrying traffic before
	// stopAll runs: a connection that has merely been dialled may not have
	// been accepted yet, in which case closing the listener alone would
	// account for it and the check below would prove nothing.
	held, reply := openEcho(t, ports[0], "held")
	defer held.Close()
	if reply != "held" {
		t.Fatalf("the held connection is not carrying traffic: got %q", reply)
	}

	f.stopAll(t.Context())

	for _, port := range ports {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Errorf("port %v was not released: %v", port, err)
			continue
		}
		l.Close()
	}

	// The connection that was open must have been closed too, otherwise the
	// wait in stopAll would not have returned.
	if err := held.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := held.Read(make([]byte, 1)); err == nil {
		t.Error("the forwarded connection was left open")
	}
}

// TestForwarderBindFailure verifies that a forward whose port cannot be bound
// is reported, and that stopAll then releases the forwards that were
// established before it.
func TestForwarderBindFailure(t *testing.T) {
	good := freePort(t)
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()

	f := newForwarder(t,
		forward{localPort: good, remotePort: echoServer(t)},
		forward{localPort: taken.Addr().(*net.TCPAddr).Port, remotePort: echoServer(t)})

	err = f.forwardAll(t.Context())
	if err == nil {
		t.Fatal("expected binding an occupied port to fail")
	}
	// The classification matters: retrying will not free the port.
	if Retryable(err) {
		t.Errorf("%v: reported as retryable, want permanent", err)
	}

	f.stopAll(t.Context())
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", good))
	if err != nil {
		t.Errorf("the forward established before the failure was not released: %v", err)
	} else {
		l.Close()
	}
}

// TestForwarderUnreachableRemote verifies that a connection which cannot be
// forwarded does not end the forward: once the remote port is listening,
// connections are carried again.
func TestForwarderUnreachableRemote(t *testing.T) {
	local, remote := freePort(t), freePort(t)
	f := newForwarder(t, forward{localPort: local, remotePort: remote})
	if err := f.forwardAll(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer f.stopAll(t.Context())

	// Nothing is listening on the remote port, so this connection is accepted
	// locally and then abandoned.
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", local))
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("expected the abandoned connection to be closed")
	}
	conn.Close()

	// With the remote port now listening, the forward must carry traffic
	// again. Dialling it is not enough to show that: the local port stays
	// bound whether or not the accept loop is still running, so only a reply
	// distinguishes a forward that survived from one that did not.
	echoServerOn(t, remote)
	if got, want := echoThrough(t, local, "after"), "after"; got != want {
		t.Errorf("got %v, want %v: the forward did not survive a failed connection", got, want)
	}
}

// TestForwarderConnectionsPruned verifies that closed forwarded connections
// are removed from f.conns rather than retained indefinitely.
func TestForwarderConnectionsPruned(t *testing.T) {
	echo := echoServer(t)
	port := freePort(t)
	f := newForwarder(t, forward{localPort: port, remotePort: echo})
	if err := f.forwardAll(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer f.stopAll(t.Context())

	// Open and close several connections sequentially.
	for i := 0; i < 5; i++ {
		conn, reply := openEcho(t, port, fmt.Sprintf("ping-%d", i))
		if reply != fmt.Sprintf("ping-%d", i) {
			t.Fatalf("unexpected reply: %v", reply)
		}
		conn.Close()
	}

	// Wait for goroutines to finish closing and pruning from conns.
	deadline := time.Now().Add(5 * time.Second)
	for {
		f.mu.Lock()
		count := len(f.conns)
		f.mu.Unlock()
		if count == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected f.conns to be 0 after all connections closed, got %d", count)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
