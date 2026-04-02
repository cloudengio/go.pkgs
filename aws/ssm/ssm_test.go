// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloudeng.io/aws/awsconfig"
	"github.com/alexbacchin/ssm-session-client/ssmclient"
)

func TestSessionLocalPort(t *testing.T) {
	s := &Session{
		pfi: ssmclient.PortForwardingInput{
			LocalPort: 8080,
		},
	}
	if got, want := s.LocalPort(), 8080; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestNewPortForwardingSession(t *testing.T) {
	ctx := t.Context()
	pfi := ssmclient.PortForwardingInput{
		LocalPort:  8080,
		RemotePort: 80,
		Target:     "example.com",
	}

	// Missing config in context.
	s, err := NewPortForwardingSession(ctx, pfi)
	if s != nil {
		t.Errorf("expected nil session, got %v", s)
	}
	if err != awsconfig.ErrConfigNotFound {
		t.Errorf("expected %v, got %v", awsconfig.ErrConfigNotFound, err)
	}
}

// TestSessionWaitClean verifies that Wait returns nil when the session ends
// cleanly before the timeout.
func TestSessionWaitClean(t *testing.T) {
	errCh := make(chan error, 1)
	close(errCh)

	s := &Session{
		pfi:   ssmclient.PortForwardingInput{LocalPort: 5432},
		errCh: errCh,
	}
	if err := s.Wait(t.Context(), time.Second); err != nil {
		t.Errorf("expected nil from Wait on clean shutdown, got %v", err)
	}
}

// TestSessionWaitError verifies that Wait surfaces errors from the forwarding loop.
func TestSessionWaitError(t *testing.T) {
	sentinel := errors.New("forwarding error")
	errCh := make(chan error, 1)
	errCh <- sentinel
	close(errCh)

	s := &Session{
		pfi:   ssmclient.PortForwardingInput{LocalPort: 5432},
		errCh: errCh,
	}
	if got := s.Wait(t.Context(), time.Second); !errors.Is(got, sentinel) {
		t.Errorf("expected sentinel error, got %v", got)
	}
}

// TestSessionWaitTimeout verifies that Wait returns DeadlineExceeded when the
// session does not end within the given duration.
func TestSessionWaitTimeout(t *testing.T) {
	errCh := make(chan error) // never closed or written to

	s := &Session{
		pfi:   ssmclient.PortForwardingInput{LocalPort: 5432},
		errCh: errCh,
	}
	if got := s.Wait(t.Context(), 50*time.Millisecond); !errors.Is(got, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", got)
	}
}
