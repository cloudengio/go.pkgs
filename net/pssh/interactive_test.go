// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// pipes returns the ends of two pipes standing in for a terminal's input and
// output: the test writes to in and reads from out.
type pipes struct {
	inR, inW, outR, outW *os.File
}

func newPipes(t *testing.T) pipes {
	t.Helper()
	var p pipes
	var err error
	if p.inR, p.inW, err = os.Pipe(); err != nil {
		t.Fatal(err)
	}
	if p.outR, p.outW, err = os.Pipe(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, f := range []*os.File{p.inR, p.inW, p.outR, p.outW} {
			f.Close()
		}
	})
	return p
}

func (p pipes) options(extra ...RunOption) []RunOption {
	return append([]RunOption{WithStdin(p.inR), WithStdout(p.outW)}, extra...)
}

func readN(t *testing.T, r io.Reader, n int) string {
	t.Helper()
	buf := make([]byte, n)
	got, err := io.ReadFull(r, buf)
	if err != nil {
		t.Fatalf("read %d bytes, got %q: %v", n, buf[:got], err)
	}
	return string(buf)
}

func runAsync(ctx context.Context, c *Client, cmd string, opts ...RunOption) <-chan error {
	done := make(chan error, 1)
	go func() { done <- c.RunInteractive(ctx, cmd, opts...) }()
	return done
}

func waitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for RunInteractive to return")
		return nil
	}
}

func nextPTYRequest(t *testing.T, ch <-chan ptyRequest) ptyRequest {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for a pty request")
		return ptyRequest{}
	}
}

func TestRunInteractive(t *testing.T) {
	requests := observePTYRequests(t)
	c, _, _ := runFixture(t)
	p := newPipes(t)

	done := runAsync(timeoutCtx(t), c, "cat", p.options(WithTerminal("test-term", WindowSize{132, 43}))...)
	if _, err := p.inW.WriteString("hello\n"); err != nil {
		t.Fatal(err)
	}
	if got, want := readN(t, p.outR, 6), "hello\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// EOF on the input ends cat, and so the command.
	p.inW.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}

	got := nextPTYRequest(t, requests)
	if got.Type != "pty-req" || got.Term != "test-term" || got.Cols != 132 || got.Rows != 43 {
		t.Errorf("got %+v, want a pty-req for test-term of 132x43", got)
	}
}

func TestRunInteractiveDefaultTerm(t *testing.T) {
	requests := observePTYRequests(t)
	c, _, _ := runFixture(t)
	p := newPipes(t)
	if err := c.RunInteractive(timeoutCtx(t), "true", p.options()...); err != nil {
		t.Fatal(err)
	}
	got := nextPTYRequest(t, requests)
	if got.Term != defaultTerm || got.Cols != defaultWidth || got.Rows != defaultHeight {
		t.Errorf("got %+v, want %s at %dx%d", got, defaultTerm, defaultWidth, defaultHeight)
	}
}

func TestRunInteractiveExitStatus(t *testing.T) {
	c, _, _ := runFixture(t)
	p := newPipes(t)
	err := c.RunInteractive(timeoutCtx(t), "echo before; exit 5", p.options()...)
	var exitErr *ssh.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitStatus() != 5 {
		t.Fatalf("got %v, want an exit status of 5", err)
	}
	if got, want := readN(t, p.outR, 7), "before\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRunInteractiveLeavesInput checks that nothing reads the input once
// RunInteractive has returned, as it would be consumed on behalf of a command
// that is no longer there.
func TestRunInteractiveLeavesInput(t *testing.T) {
	c, _, _ := runFixture(t)
	p := newPipes(t)
	if err := c.RunInteractive(timeoutCtx(t), "true", p.options()...); err != nil {
		t.Fatal(err)
	}
	if _, err := p.inW.WriteString("later"); err != nil {
		t.Fatal(err)
	}
	// The reader is stopped synchronously so, were it still running, it
	// would take the input within its poll interval.
	time.Sleep(500 * time.Millisecond)
	if got, want := readN(t, p.inR, 5), "later"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRunInteractiveResize(t *testing.T) {
	requests := observePTYRequests(t)
	c, _, _ := runFixture(t)
	p := newPipes(t)
	changes := make(chan WindowSize)

	done := runAsync(timeoutCtx(t), c, "cat", p.options(WithWindowChanges(changes))...)
	if got := nextPTYRequest(t, requests); got.Type != "pty-req" {
		t.Fatalf("got %+v, want a pty-req", got)
	}
	for _, sz := range []WindowSize{{100, 40}, {132, 50}} {
		changes <- sz
		got := nextPTYRequest(t, requests)
		if got.Type != "window-change" || int(got.Cols) != sz.Width || int(got.Rows) != sz.Height {
			t.Errorf("got %+v, want a window-change to %v", got, sz)
		}
	}
	p.inW.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
}

func TestRunInteractiveReader(t *testing.T) {
	requests := observePTYRequests(t)
	c, _, _ := runFixture(t)
	var out bytes.Buffer
	err := c.RunInteractive(timeoutCtx(t), "cat",
		WithStdin(strings.NewReader("from a reader\n")), WithStdout(&out))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "from a reader\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := nextPTYRequest(t, requests); got.Type != "pty-req" {
		t.Errorf("got %+v, want a pty-req", got)
	}
}

func TestRunInteractiveCancel(t *testing.T) {
	c, _, _ := runFixture(t)
	p := newPipes(t)
	ctx, cancel := context.WithCancel(timeoutCtx(t))
	defer cancel()

	done := runAsync(ctx, c, "echo started; sleep 60", p.options()...)
	readN(t, p.outR, len("started\n"))
	cancel()
	if err := waitRun(t, done); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want %v", err, context.Canceled)
	}
}

func TestRunInteractiveClientClosed(t *testing.T) {
	c, _, _ := runFixture(t)
	p := newPipes(t)
	done := runAsync(timeoutCtx(t), c, "echo started; sleep 60", p.options()...)
	readN(t, p.outR, len("started\n"))
	c.Close()
	if err := waitRun(t, done); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("got %v, want %v", err, ErrClientClosed)
	}
}

func TestRunInteractiveClosedWindowChanges(t *testing.T) {
	requests := observePTYRequests(t)
	c, _, _ := runFixture(t)
	p := newPipes(t)
	changes := make(chan WindowSize, 1)
	done := runAsync(timeoutCtx(t), c, "cat", p.options(WithWindowChanges(changes))...)
	_ = nextPTYRequest(t, requests)

	changes <- WindowSize{Width: 100, Height: 40}
	if got := nextPTYRequest(t, requests); got.Type != "window-change" || got.Cols != 100 || got.Rows != 40 {
		t.Fatalf("got %+v, want window-change 100x40", got)
	}
	close(changes)
	time.Sleep(20 * time.Millisecond)

	p.inW.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-requests:
		t.Fatalf("unexpected request after closing window changes: %+v", r)
	default:
	}
}
