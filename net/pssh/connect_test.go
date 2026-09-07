// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
	"golang.org/x/crypto/ssh"
)

// sshServer accepts ssh connections on a real address, so that a client's
// reconnection loop can be exercised against a server whose connections can be
// dropped underneath it.
type sshServer struct {
	listener               net.Listener
	hostSigner, userSigner ssh.Signer
	user                   string
	accepted               chan struct{}

	mu    sync.Mutex
	conns []net.Conn
}

func newSSHServer(t *testing.T, hostSigner, userSigner ssh.Signer, user string) *sshServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &sshServer{
		listener:   l,
		hostSigner: hostSigner,
		userSigner: userSigner,
		user:       user,
		// Buffered so that a connection the test is not waiting for does not
		// block the accept loop.
		accepted: make(chan struct{}, 16),
	}
	t.Cleanup(s.close)
	go s.serve()
	return s
}

func (s *sshServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()
		serveSSH(conn, s.hostSigner, s.userSigner, s.user)
		select {
		case s.accepted <- struct{}{}:
		default:
		}
	}
}

func (s *sshServer) addr() string { return s.listener.Addr().String() }

// waitForConnection waits for the server to accept a connection. Note that it
// returns when the connection is accepted, which is before its handshake and
// any forwards it carries are complete.
func (s *sshServer) waitForConnection(t *testing.T) {
	t.Helper()
	select {
	case <-s.accepted:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the client to connect")
	}
}

// drop closes every connection accepted so far, as a server that restarts
// does, without closing the listener.
func (s *sshServer) drop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, conn := range s.conns {
		conn.Close()
	}
	s.conns = nil
}

func (s *sshServer) close() {
	s.listener.Close()
	s.drop()
}

func fastBackoff() ratecontrol.Backoff {
	return ratecontrol.NewExponentialBackoff(time.Millisecond, 20)
}

// slowBackoff makes any retry take longer than a test will wait, so that a
// test which returns promptly has demonstrably not retried.
func slowBackoff() ratecontrol.Backoff {
	return ratecontrol.NewExponentialBackoff(time.Hour, 10)
}

// echoEventually waits for a forward to carry traffic, which happens some
// short time after the server accepts the connection carrying it.
func echoEventually(t *testing.T, port int, msg string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if echoOnce(port, msg) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the forward on port %v never carried traffic", port)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func echoOnce(port int, msg string) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return false
	}
	if _, err := io.WriteString(conn, msg); err != nil {
		return false
	}
	reply := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false
	}
	return string(reply) == msg
}

// connectInBackground runs ConnectAndWait and returns a channel carrying its
// result, since it does not return whilst the connection is being maintained.
func connectInBackground(ctx context.Context, c *Client, backoff func() ratecontrol.Backoff) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := c.ConnectAndWait(ctx, backoff)
		done <- err
	}()
	return done
}

func waitForResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("ConnectAndWait did not return")
		return nil
	}
}

// TestConnectAndWaitReconnects verifies that a connection lost to a server
// that drops it is established again, along with the forwards it carries, and
// that closing the client is what ends it.
func TestConnectAndWaitReconnects(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")

	local := freePort(t)
	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"),
		WithHostKey(hostSigner.PublicKey()),
		WithLocalPortForward(local, echoServer(t)))

	// The backoff factory is called once per attempt sequence, so counting
	// the calls shows whether the delays are reset after a connection
	// succeeds rather than carried into the next outage.
	var mu sync.Mutex
	created := 0
	backoff := func() ratecontrol.Backoff {
		mu.Lock()
		defer mu.Unlock()
		created++
		return fastBackoff()
	}

	done := connectInBackground(t.Context(), c, backoff)
	srv.waitForConnection(t)
	echoEventually(t, local, "first")

	// Dropping the connection must be recovered from, and the forward
	// re-established on the same local port. That only works if the previous
	// listener was released, so this is the reconnection case that the
	// forwarder's teardown exists for.
	srv.drop()
	srv.waitForConnection(t)
	echoEventually(t, local, "second")

	c.Close()
	if err := waitForResult(t, done); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want ErrClientClosed", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if created < 2 {
		t.Errorf("the backoff was created %v times, want at least 2: it is not reset after a successful connection", created)
	}
}

// TestConnectAndWaitPermanentError verifies that an error which retrying
// cannot resolve ends the loop rather than being retried. The backoff used
// would take an hour to allow a second attempt, so returning at all shows that
// no retry was made.
func TestConnectAndWaitPermanentError(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	_, otherSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")

	// The server presents a host key other than the one pinned here.
	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"), WithHostKey(otherSigner.PublicKey()))

	err := waitForResult(t, connectInBackground(t.Context(), c, slowBackoff))
	if err == nil {
		t.Fatal("expected an error")
	}
	if Retryable(err) {
		t.Errorf("%v: reported as retryable", err)
	}
	if got, want := err.Error(), "host key mismatch"; !strings.Contains(got, want) {
		t.Errorf("error %q does not contain %q", got, want)
	}
}

// TestConnectAndWaitBackoffExhausted verifies that a retryable error is
// retried, and that the loop ends when the backoff will allow no more.
func TestConnectAndWaitBackoffExhausted(t *testing.T) {
	userPriv, _ := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)

	// Nothing is listening, so every attempt is refused, which is retryable.
	c := NewClient(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", freePort(t)),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))

	steps := 3
	backoff := func() ratecontrol.Backoff {
		return ratecontrol.NewExponentialBackoff(time.Millisecond, steps)
	}
	err := waitForResult(t, connectInBackground(t.Context(), c, backoff))
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "backoff exhausted"; !strings.Contains(got, want) {
		t.Errorf("error %q does not contain %q", got, want)
	}
	// The error that caused the attempts is kept, since it is what says why
	// the connection could not be made.
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Errorf("error %v does not wrap the connection failure", err)
	}
}

// TestConnectAndWaitCancelled verifies that cancelling the context ends the
// loop whilst it is waiting to retry.
func TestConnectAndWaitCancelled(t *testing.T) {
	userPriv, _ := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)

	ctx, cancel := context.WithCancel(t.Context())
	c := NewClient(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", freePort(t)),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))

	done := connectInBackground(ctx, c, slowBackoff)
	cancel()
	if err := waitForResult(t, done); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

// TestConnectAndWaitCloseWhilstRetrying verifies that closing the client ends
// the loop whilst it is waiting to retry, as distinct from whilst it is
// connected.
func TestConnectAndWaitCloseWhilstRetrying(t *testing.T) {
	userPriv, _ := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)

	c := NewClient(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", freePort(t)),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))

	done := connectInBackground(t.Context(), c, slowBackoff)
	c.Close()
	if err := waitForResult(t, done); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want ErrClientClosed", err)
	}
	// Close is idempotent: a second call must not panic on a closed channel.
	c.Close()
}

func TestNewClientDefaults(t *testing.T) {
	t.Setenv("USER", "from-environment")
	c := NewClient(t.Context(), "tcp", testAddr)
	if got, want := c.opts.user, "from-environment"; got != want {
		t.Errorf("user: got %v, want %v", got, want)
	}
	if got, want := c.opts.dialTimeout, DefaultDialTimeout; got != want {
		t.Errorf("dialTimeout: got %v, want %v", got, want)
	}
	if c.opts.logger == nil {
		t.Error("logger: got nil, want a logger")
	}

	c = NewClient(t.Context(), "tcp", testAddr,
		WithUser("explicit"),
		WithDialTimeout(time.Minute),
		WithLocalPortForward(1234, 5678))
	if got, want := c.opts.user, "explicit"; got != want {
		t.Errorf("user: got %v, want %v", got, want)
	}
	if got, want := c.opts.dialTimeout, time.Minute; got != want {
		t.Errorf("dialTimeout: got %v, want %v", got, want)
	}
	if got, want := c.opts.localForwards, []forward{{localPort: 1234, remotePort: 5678}}; !slices.Equal(got, want) {
		t.Errorf("forwards: got %v, want %v", got, want)
	}
}

// TestWithLogger verifies that the supplied logger is the one the client
// uses, and that it carries the address the client was created for, so that
// the output of several clients can be told apart.
func TestWithLogger(t *testing.T) {
	var out bytes.Buffer
	c := NewClient(t.Context(), "tcp", testAddr,
		WithLogger(slog.New(slog.NewTextHandler(&out, nil))))
	c.opts.logger.Info("a message")

	logged := out.String()
	for _, want := range []string{"a message", "ssh.network=tcp", "ssh.addr=" + testAddr} {
		if !strings.Contains(logged, want) {
			t.Errorf("logged %q does not contain %q", logged, want)
		}
	}
}

func TestDefaultKnownHostsFiles(t *testing.T) {
	files := defaultKnownHostsFiles()
	if got, want := files[len(files)-1], "/etc/ssh/ssh_known_hosts"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory in this environment")
	}
	if got, want := files[0], filepath.Join(home, ".ssh", "known_hosts"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
