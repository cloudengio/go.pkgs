// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// windowSize is the size of a terminal in characters.
type windowSize struct{ width, height int }

const (
	defaultTerm   = "xterm-256color"
	defaultWidth  = 80
	defaultHeight = 24
)

func withResizes(ch <-chan windowSize) RunOption {
	return func(o *runOptions) { o.resizes = ch }
}

// RunInteractive is like Run but runs cmd on a pseudo terminal that is
// connected to the local terminal: standard input and output are those of the
// process unless overridden by WithStdin and WithStdout, and WithStderr is
// ignored since a terminal has a single output stream. It is intended for
// commands that need a terminal, such as 'tmux -CC new -A -s main'.
//
// If standard input is a terminal it is put into raw mode for the duration and
// restored afterwards, and changes to the size of the terminal are passed on to
// the remote one. The pseudo terminal is requested with the value of $TERM, or
// xterm-256color if that is unset.
//
// Unlike Run, a standard input that is an *os.File is never read after
// RunInteractive returns, so that no input is lost to a command that has
// finished; any other reader behaves as described for WithStdin.
//
// When ctx is done, or the connection is lost, the pseudo terminal is closed
// and so the command is sent SIGHUP rather than SIGTERM, which is the
// appropriate signal for one that is detached from its terminal.
func (c *Client) RunInteractive(ctx context.Context, cmd string, opts ...RunOption) error {
	ro := newRunOptions(opts)
	var in io.Reader = os.Stdin
	if ro.stdin != nil {
		in = ro.stdin
	}
	var out io.Writer = os.Stdout
	if ro.stdout != nil {
		out = ro.stdout
	}
	return c.run(ctx, cmd, ro, func(s *ssh.Session) (func(), error) {
		return prepareInteractive(s, in, out, ro.resizes)
	})
}

// prepareInteractive requests a terminal for s, connects it to in and out and
// starts passing on changes to the local terminal's size. The returned
// function undoes all of that.
func prepareInteractive(s *ssh.Session, in io.Reader, out io.Writer, resizes <-chan windowSize) (func(), error) {
	var undo []func()
	cleanup := func() {
		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}
	}
	local := localTerminal(in, out)
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec // fds fit in an int.
		state, err := term.MakeRaw(int(f.Fd())) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("setting the terminal to raw mode: %w", err)
		}
		undo = append(undo, func() { _ = term.Restore(int(f.Fd()), state) }) //nolint:gosec
	}

	size := windowSize{defaultWidth, defaultHeight}
	if local != nil {
		if w, h, err := term.GetSize(int(local.Fd())); err == nil { //nolint:gosec
			size = windowSize{w, h}
		}
	}
	termName := os.Getenv("TERM")
	if termName == "" {
		termName = defaultTerm
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := s.RequestPty(termName, size.height, size.width, modes); err != nil {
		cleanup()
		return nil, fmt.Errorf("requesting a pseudo terminal: %w", err)
	}

	stop := make(chan struct{})
	s.Stdout = out
	if f, ok := in.(*os.File); ok {
		w, err := s.StdinPipe()
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("opening stdin: %w", err)
		}
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			pumpStdin(f, w, stop)
		}()
		undo = append(undo, func() { <-finished })
	} else {
		s.Stdin = in
	}
	if resizes == nil {
		resizes = watchResize(local, stop)
	}
	go forwardResizes(s, resizes, stop)
	undo = append(undo, func() { close(stop) })
	return cleanup, nil
}

// localTerminal returns the first of in and out that is a terminal, or nil.
func localTerminal(in io.Reader, out io.Writer) *os.File {
	for _, v := range []any{out, in} {
		if f, ok := v.(*os.File); ok && term.IsTerminal(int(f.Fd())) { //nolint:gosec
			return f
		}
	}
	return nil
}

func forwardResizes(s *ssh.Session, resizes <-chan windowSize, stop <-chan struct{}) {
	for {
		select {
		case sz := <-resizes:
			_ = s.WindowChange(sz.height, sz.width)
		case <-stop:
			return
		}
	}
}
