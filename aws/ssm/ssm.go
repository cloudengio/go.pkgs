// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/aws/awsconfig"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
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
// goroutine so the caller is not blocked.
//
// If PortForwardingSessionWithContext returns an error during the startup
// window (before it has begun accepting connections), that error is returned
// immediately. Once the session is forwarding, NewPortForwardingSession
// returns a non-nil *Session; call Wait to block until the session ends and
// retrieve any error that occurred during forwarding.
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
	go func() {
		if err := ssmclient.PortForwardingSessionWithContext(ctx, *cfg, &pfi); err != nil {
			errCh <- err
		}
		close(errCh)
	}()
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
func (s *Session) Wait(duration time.Duration) error {
	select {
	case <-time.After(duration):
		return context.DeadlineExceeded
	case err, ok := <-s.errCh:
		if !ok {
			return nil
		}
		return err
	}
}
