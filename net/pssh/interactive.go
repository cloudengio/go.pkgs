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
)

// WindowSize is the size of a terminal in characters.
type WindowSize struct{ Width, Height int }

const (
	defaultTerm   = "xterm-256color"
	defaultWidth  = 80
	defaultHeight = 24
)

// WithTerminal specifies the terminal type, eg. the value of $TERM, and the
// initial size of the pseudo terminal that RunInteractive requests. The defaults
// are xterm-256color and 80x24. A zero Width or Height is replaced by its
// default.
func WithTerminal(termType string, size WindowSize) RunOption {
	return func(o *runOptions) { o.termType, o.size = termType, size }
}

// WithWindowChanges specifies a channel of new sizes for the pseudo terminal
// that RunInteractive requests, typically of the local terminal in response to
// SIGWINCH.
func WithWindowChanges(ch <-chan WindowSize) RunOption {
	return func(o *runOptions) { o.windowChanges = ch }
}

// RunInteractive is like Run but runs cmd on a pseudo terminal: standard input
// and output are those of the process unless overridden by WithStdin and
// WithStdout, and WithStderr is ignored since a terminal has a single output
// stream. It is intended for commands that need a terminal, such as
// 'tmux -CC new -A -s main'.
//
// RunInteractive does not interact with the local terminal, which is the
// caller's responsibility: it should be put into raw mode, eg. using
// golang.org/x/term, before and restored after the call, and its type, size
// and subsequent changes to its size supplied using WithTerminal and
// WithWindowChanges.
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
		return prepareInteractive(s, in, out, ro)
	})
}

// prepareInteractive requests a terminal for s, connects it to in and out and
// starts passing on changes to its size. The returned function undoes all of
// that.
func prepareInteractive(s *ssh.Session, in io.Reader, out io.Writer, ro runOptions) (func(), error) {
	size := ro.size
	if size.Width <= 0 {
		size.Width = defaultWidth
	}
	if size.Height <= 0 {
		size.Height = defaultHeight
	}
	termType := ro.termType
	if termType == "" {
		termType = defaultTerm
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := s.RequestPty(termType, size.Height, size.Width, modes); err != nil {
		return nil, fmt.Errorf("requesting a pseudo terminal: %w", err)
	}

	stop := make(chan struct{})
	cleanup := func() { close(stop) }
	s.Stdout = out
	if f, ok := in.(*os.File); ok {
		w, err := s.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("opening stdin: %w", err)
		}
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			pumpStdin(f, w, stop)
		}()
		cleanup = func() { close(stop); <-finished }
	} else {
		s.Stdin = in
	}
	if ro.windowChanges != nil {
		go forwardWindowChanges(s, ro.windowChanges, stop)
	}
	return cleanup, nil
}

func forwardWindowChanges(s *ssh.Session, changes <-chan WindowSize, stop <-chan struct{}) {
	for {
		select {
		case sz, ok := <-changes:
			if !ok {
				return
			}
			_ = s.WindowChange(sz.Height, sz.Width)
		case <-stop:
			return
		}
	}
}
