// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm

import (
	"context"
	"fmt"

	"cloudeng.io/aws/awsconfig"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// Session represents an active SSM port forwarding session.
// It provides access to the local port that is being forwarded to the
// remote host.
type Session struct {
	pfi ssmclient.PortForwardingInput
}

// NewPortForwardingSession starts a new SSM port forwarding session based
// on the provided input parameters.
// It returns a Session object that can be used to retrieve the local port
// for the forwarded connection.
// The session can be closed by canceling the supplied context.
func NewPortForwardingSession(ctx context.Context, pfi ssmclient.PortForwardingInput) (*Session, error) {
	cfg, ok := awsconfig.FromContext(ctx)
	if !ok || cfg == nil {
		return nil, awsconfig.ErrConfigNotFound
	}
	err := ssmclient.PortForwardingSessionWithContext(ctx, *cfg, &pfi)
	if err != nil {
		fmt.Printf("failed to start SSM port forwarding session: %v\n", err)
		return nil, fmt.Errorf("failed to start SSM port forwarding session: %w", err)
	}
	return &Session{pfi: pfi}, nil
}

// LocalPort returns the local port that is being forwarded to the remote
// host. Clients can connect to this port to access the remote service
// through the SSM tunnel.
func (s *Session) LocalPort() int {
	return s.pfi.LocalPort
}
