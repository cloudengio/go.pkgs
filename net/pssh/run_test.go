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
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// runFixture starts a client connected to an in-process server that runs
// commands.
func runFixture(t *testing.T) (*Client, *sshServer, <-chan error) {
	t.Helper()
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")
	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))
	done := connectInBackground(t.Context(), c, fastBackoff)
	t.Cleanup(c.Close)
	return c, srv, done
}

func timeoutCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestRun(t *testing.T) {
	c, _, _ := runFixture(t)
	ctx := timeoutCtx(t)

	var stdout, stderr bytes.Buffer
	err := c.Run(ctx, "echo out; echo err >&2", WithStdout(&stdout), WithStderr(&stderr))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "out\n"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
	if got, want := stderr.String(), "err\n"; got != want {
		t.Errorf("stderr: got %q, want %q", got, want)
	}

	// Output is discarded if not asked for.
	if err := c.Run(ctx, "echo ignored"); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	err = c.Run(ctx, "echo before; exit 3", WithStdout(&stdout))
	var exitErr *ssh.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitStatus() != 3 {
		t.Fatalf("got %v, want an exit status of 3", err)
	}
	if got, want := stdout.String(), "before\n"; got != want {
		t.Errorf("stdout: got %q, want %q", got, want)
	}
}

func TestRunStdin(t *testing.T) {
	c, _, _ := runFixture(t)
	var stdout bytes.Buffer
	err := c.Run(timeoutCtx(t), "cat", WithStdin(strings.NewReader("hello\n")), WithStdout(&stdout))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "hello\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunConcurrent(t *testing.T) {
	c, _, _ := runFixture(t)
	ctx := timeoutCtx(t)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := range 8 {
		wg.Go(func() {
			var out bytes.Buffer
			if err := c.Run(ctx, fmt.Sprintf("echo %d", i), WithStdout(&out)); err != nil {
				errs <- err
				return
			}
			if got, want := out.String(), fmt.Sprintf("%d\n", i); got != want {
				errs <- fmt.Errorf("got %q, want %q", got, want)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestRunWaitsForConnection verifies that Run called before the connection is
// established waits for it.
func TestRunWaitsForConnection(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")
	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))
	t.Cleanup(c.Close)

	result := make(chan error, 1)
	go func() { result <- c.Run(timeoutCtx(t), "true") }()
	select {
	case err := <-result:
		t.Fatalf("Run returned before there was a connection: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	connectInBackground(t.Context(), c, fastBackoff)
	if err := waitForResult(t, result); err != nil {
		t.Fatal(err)
	}
	if err := c.WaitConnected(timeoutCtx(t)); err != nil {
		t.Errorf("WaitConnected: %v", err)
	}
}

// TestRunNeverConnected verifies that Run, and WaitConnected, are bounded by
// the context, and end when the client is closed.
func TestRunNeverConnected(t *testing.T) {
	userPriv, _ := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	c := NewClient(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", freePort(t)),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()))
	t.Cleanup(c.Close)
	connectInBackground(t.Context(), c, slowBackoff)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := c.Run(ctx, "true"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run: got %v, want context.DeadlineExceeded", err)
	}
	if err := c.WaitConnected(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("WaitConnected: got %v, want context.DeadlineExceeded", err)
	}

	result := make(chan error, 1)
	go func() { result <- c.Run(timeoutCtx(t), "true") }()
	time.Sleep(20 * time.Millisecond)
	c.Close()
	if err := waitForResult(t, result); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want ErrClientClosed", err)
	}
	if err := c.Run(t.Context(), "true"); !errors.Is(err, ErrClientClosed) {
		t.Errorf("after Close: got %v, want ErrClientClosed", err)
	}
}

// startSleeping starts a command that does not complete and returns a channel
// for its result once it is known to be running.
func startSleeping(ctx context.Context, t *testing.T, c *Client) <-chan error {
	t.Helper()
	pr, pw := io.Pipe()
	result := make(chan error, 1)
	go func() {
		result <- c.Run(ctx, "echo started; sleep 60", WithStdout(pw))
		pw.Close()
	}()
	buf := make([]byte, 8)
	if _, err := io.ReadFull(pr, buf[:len("started\n")]); err != nil {
		t.Fatalf("the command did not start: %v", err)
	}
	go io.Copy(io.Discard, pr) //nolint:errcheck
	return result
}

func TestRunCancel(t *testing.T) {
	c, _, _ := runFixture(t)
	ctx, cancel := context.WithCancel(timeoutCtx(t))
	result := startSleeping(ctx, t, c)
	cancel()
	if err := waitForResult(t, result); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	// The connection is unaffected.
	if err := c.Run(timeoutCtx(t), "true"); err != nil {
		t.Errorf("Run after cancel: %v", err)
	}
}

// TestRunConnectionLost verifies that a command running when the connection
// is lost is not retried, and that the client reconnects for the next.
func TestRunConnectionLost(t *testing.T) {
	c, srv, _ := runFixture(t)
	ctx := timeoutCtx(t)
	result := startSleeping(ctx, t, c)
	srv.drop()
	err := waitForResult(t, result)
	var exitErr *ssh.ExitError
	if err == nil || errors.As(err, &exitErr) {
		t.Errorf("got %v, want an error that is not an exit status", err)
	}
	var out bytes.Buffer
	if err := c.Run(ctx, "echo again", WithStdout(&out)); err != nil {
		t.Fatalf("Run after reconnecting: %v", err)
	}
	if got, want := out.String(), "again\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunClientClosed(t *testing.T) {
	c, _, _ := runFixture(t)
	result := startSleeping(timeoutCtx(t), t, c)
	c.Close()
	if err := waitForResult(t, result); !errors.Is(err, ErrClientClosed) {
		t.Errorf("got %v, want ErrClientClosed", err)
	}
}

// TestConnectionAvailableInOnConnect verifies that the connection can be used
// from the OnConnect callback.
func TestRunInOnConnect(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	srv := newSSHServer(t, hostSigner, userSigner, "auser")
	result := make(chan error, 1)
	var once sync.Once
	c := NewClient(t.Context(), "tcp", srv.addr(),
		WithUser("auser"), WithHostKey(hostSigner.PublicKey()),
		WithOnConnect(func(c *Client) {
			once.Do(func() {
				// Run in the callback's goroutine would be deadlock-prone for a
				// blocking command; the connection is set so it must not wait.
				result <- c.Run(timeoutCtx(t), "true")
			})
		}))
	t.Cleanup(c.Close)
	connectInBackground(t.Context(), c, fastBackoff)
	if err := waitForResult(t, result); err != nil {
		t.Fatal(err)
	}
}
