// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm

import (
	"testing"

	"cloudeng.io/aws/awsconfig"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
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
		LocalPort: 8080,
	}

	// 1. Missing config in context
	s, err := NewPortForwardingSession(ctx, pfi)
	if s != nil {
		t.Errorf("expected nil session, got %v", s)
	}
	if err != awsconfig.ErrConfigNotFound {
		t.Errorf("expected %v, got %v", awsconfig.ErrConfigNotFound, err)
	}
}
