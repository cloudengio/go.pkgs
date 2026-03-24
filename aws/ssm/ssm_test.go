// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ssm_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awstestutil"
	cloudengssm "cloudeng.io/aws/ssm"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mmmorris1975/ssm-session-client/ssmclient"
)

// TestNewPortForwardingSession_NoConfig verifies that NewPortForwardingSession
// returns ErrConfigNotFound when no AWS config is stored in the context.
func TestNewPortForwardingSession_NoConfig(t *testing.T) {
	ctx := context.Background()
	pfi := ssmclient.PortForwardingInput{
		Target:     "i-12345678901234567",
		RemotePort: 5432,
	}
	session, err := cloudengssm.NewPortForwardingSession(ctx, pfi)
	if session != nil {
		t.Error("expected nil session when config is missing")
	}
	if !errors.Is(err, awsconfig.ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

// TestNewPortForwardingSession_SessionFails verifies that errors from the
// underlying SSM session are wrapped and returned as
// "failed to start SSM port forwarding session: ...".
//
// The AWS config's BaseEndpoint is set to an unreachable local address so
// the StartSession API call fails immediately with a connection error — no
// real AWS credentials or network access are required.
func TestNewPortForwardingSession_SessionFails(t *testing.T) {
	cfg := awstestutil.DefaultAWSConfig()
	cfg.BaseEndpoint = aws.String("http://localhost:9999")

	ctx := awsconfig.ContextWith(context.Background(), &cfg)
	pfi := ssmclient.PortForwardingInput{
		Target:     "i-12345678901234567",
		RemotePort: 5432,
	}

	session, err := cloudengssm.NewPortForwardingSession(ctx, pfi)
	if session != nil {
		t.Error("expected nil session when SSM session fails")
	}
	if err == nil {
		t.Fatal("expected error when SSM session cannot be established, got nil")
	}
	if !strings.Contains(err.Error(), "failed to start SSM port forwarding session") {
		t.Errorf("expected 'failed to start SSM port forwarding session' in error, got: %v", err)
	}
}
