// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/aws/awsconfig"
	"github.com/alexbacchin/ssm-session-client/ssmclient"
)

// Session represents an active SSM port forwarding session.
// It provides access to the local port that is being forwarded to the
// remote host.
type Session struct {
	pfi   ssmclient.PortForwardingInput
	errCh <-chan error
}

// NewPortForwardingSession starts a new SSM port forwarding session based on
// the provided input parameters. The underlying forwarding call runs in a
// goroutine so the caller is not blocked. Note that the tunnel will be
// ready to accept connections when NewPortForwardingSession returns.
//
// The session can be closed by canceling the supplied context.
func NewPortForwardingSession(ctx context.Context, pfi ssmclient.PortForwardingInput) (*Session, error) {
	if pfi.LocalPort == 0 || pfi.RemotePort == 0 || pfi.Target == "" {
		return nil, fmt.Errorf("invalid PortForwardingInput: LocalPort, RemotePort, and Target are required")
	}
	cfg, ok := awsconfig.FromContext(ctx)
	if !ok || cfg == nil {
		return nil, awsconfig.ErrConfigNotFound
	}

	errCh := make(chan error, 1)
	pfi.ReadyCh = make(chan struct{})
	go func() {
		if err := ssmclient.PortForwardingSessionWithContext(ctx, *cfg, &pfi); err != nil {
			errCh <- err
		}
		close(errCh)
	}()
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	select {
	case <-pfi.ReadyCh:
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for SSM session to be ready")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &Session{pfi: pfi, errCh: errCh}, nil
}

// LocalPort returns the local port that is being forwarded to the remote
// host. Clients can connect to this port to access the remote service
// through the SSM tunnel.
func (s *Session) LocalPort() int {
	return s.pfi.LocalPort
}

// Wait blocks until the session ends and returns any error that occurred
// during forwarding. A nil error means the session was closed cleanly
// (typically because the context was cancelled).
func (s *Session) Wait(ctx context.Context, duration time.Duration) error {
	t := time.NewTimer(duration)
	defer t.Stop()
	select {
	case <-t.C:
		return context.DeadlineExceeded
	case <-ctx.Done():
		return ctx.Err()
	case err, ok := <-s.errCh:
		if !ok {
			return nil
		}
		return err
	}
}
