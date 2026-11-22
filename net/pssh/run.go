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

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func (c *Client) setConn(client *ssh.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn = client
	close(c.ready)
}

// clearConn records that client, if it is still the current connection, has
// been lost.
func (c *Client) clearConn(client *ssh.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != client {
		return
	}
	c.conn = nil
	c.ready = make(chan struct{})
}

// connection returns the current connection, waiting for one if there is none.
func (c *Client) connection(ctx context.Context) (*ssh.Client, error) {
	for {
		c.mu.Lock()
		conn, ready := c.conn, c.ready
		c.mu.Unlock()
		if conn != nil {
			return conn, nil
		}
		select {
		case <-ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.doneCh:
			return nil, ErrClientClosed
		}
	}
}

// WaitConnected waits until the client is connected, which is when
// ConnectAndWait has established a connection. It returns ctx.Err() if ctx
// is done first and ErrClientClosed if the client is closed.
func (c *Client) WaitConnected(ctx context.Context) error {
	_, err := c.connection(ctx)
	return err
}

type runOptions struct {
	stdin          io.Reader
	stdout, stderr io.Writer
	login          bool
	// The terminal that RunInteractive requests.
	termType      string
	size          WindowSize
	windowChanges <-chan WindowSize
}

// RunOption represents an option for Run.
type RunOption func(*runOptions)

// WithStdin specifies the standard input of the command. It is discarded if
// not specified. The reader must return, eg. when the context is cancelled,
// for Run to do so.
func WithStdin(r io.Reader) RunOption {
	return func(o *runOptions) { o.stdin = r }
}

// WithStdout specifies where the standard output of the command is written,
// it is discarded if not specified.
func WithStdout(w io.Writer) RunOption {
	return func(o *runOptions) { o.stdout = w }
}

// WithStderr specifies where the standard error of the command is written,
// it is discarded if not specified.
func WithStderr(w io.Writer) RunOption {
	return func(o *runOptions) { o.stderr = w }
}

// WithLoginShell runs the command in the user's login shell, so that it sees
// the environment, notably PATH, that the profile files set up. It is the
// equivalent of running 'exec "$SHELL" -l -c cmd', which for some shells, eg.
// zsh, reads the login but not the interactive startup files.
func WithLoginShell() RunOption {
	return func(o *runOptions) { o.login = true }
}

// Run runs cmd on the server, waiting first for the client to be connected.
// The server runs cmd using the user's shell, as per ssh(1), and so it is
// subject to that shell's quoting and expansion, and does not run in a login
// shell unless WithLoginShell is used. Use ShellQuote and ShellJoin to
// construct commands from arbitrary strings.
//
// Run returns nil if the command exits with status 0, and an *ssh.ExitError,
// which carries the status, if it does not; use errors.As to retrieve it. If
// the connection is lost whilst the command is running the command is not
// retried and the error is not an *ssh.ExitError, typically an
// *ssh.ExitMissingError, since the command may have had some effect.
// ErrClientClosed is returned if the client is closed.
//
// If ctx is done the command is sent SIGTERM, which not all servers honour,
// the session is closed and ctx.Err() is returned.
//
// Many commands may be run concurrently, subject to the limit on sessions per
// connection that the server imposes.
func (c *Client) Run(ctx context.Context, cmd string, opts ...RunOption) error {
	return c.run(ctx, cmd, newRunOptions(opts), nil)
}

func newRunOptions(opts []RunOption) runOptions {
	var ro runOptions
	for _, fn := range opts {
		fn(&ro)
	}
	return ro
}

// run runs cmd. If prepare is not nil it is called with the new session and is
// responsible for configuring its streams, instead of the options doing so; the
// function that it returns is called once the command is done.
func (c *Client) run(ctx context.Context, cmd string, ro runOptions, prepare func(*ssh.Session) (func(), error)) error {
	if ro.login {
		cmd = loginShellCommand(cmd)
	}
	session, err := c.newSession(ctx)
	if err != nil {
		return err
	}
	defer session.Close()
	if c.opts.forwardAgent {
		// As ssh does, carry on without forwarding if it is refused.
		if err := agent.RequestAgentForwarding(session); err != nil {
			c.opts.logger.Warn("ssh agent forwarding was refused", "err", err)
		}
	}
	if prepare == nil {
		session.Stdin, session.Stdout, session.Stderr = ro.stdin, ro.stdout, ro.stderr
	} else {
		cleanup, err := prepare(session)
		if err != nil {
			return err
		}
		defer cleanup()
	}

	done := make(chan error, 1)
	go func() { done <- session.Run(cmd) }()
	select {
	case err := <-done:
		return c.sessionError(err)
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		session.Close()
		<-done //nolint:errcheck
		return ctx.Err()
	}
}

// newSession opens a session on the current connection. A connection that has
// been lost but not yet replaced is waited on rather than reported, since the
// command has not been started and so can safely be run on its successor.
func (c *Client) newSession(ctx context.Context) (*ssh.Session, error) {
	for {
		conn, err := c.connection(ctx)
		if err != nil {
			return nil, err
		}
		session, err := conn.NewSession()
		if err == nil {
			return session, nil
		}
		if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			return nil, c.sessionError(fmt.Errorf("opening ssh session: %w", err))
		}
		c.clearConn(conn)
	}
}

// sessionError returns ErrClientClosed in place of the error caused by
// closing the client, and leaves the exit status of a command alone.
func (c *Client) sessionError(err error) error {
	var exitErr *ssh.ExitError
	if err == nil || errors.As(err, &exitErr) {
		return err
	}
	select {
	case <-c.doneCh:
		return ErrClientClosed
	default:
		return err
	}
}
