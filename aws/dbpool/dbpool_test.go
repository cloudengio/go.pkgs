// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dbpool_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloudeng.io/aws/dbpool"
)

// TestInvalidConnectionString verifies that a malformed connection string
// causes NewConnectionPool to fail immediately.
func TestInvalidConnectionString(t *testing.T) {
	ctx := t.Context()
	pool, err := dbpool.NewConnectionPool(ctx, "not::a::valid::connection::string")
	if err == nil {
		pool.Close()
		t.Fatal("expected error for invalid connection string, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse connection string") {
		t.Errorf("expected 'failed to parse connection string' in error, got: %v", err)
	}
}

// TestPoolCreation verifies that a pool can be created without immediately
// connecting. pgxpool is lazy: connections are made on demand, not at pool
// creation time, so an unreachable host is not an error here.
func TestPoolCreation(t *testing.T) {
	ctx := t.Context()
	pool, err := dbpool.NewConnectionPool(ctx, "postgres://localhost:9999/test")
	if err != nil {
		t.Fatalf("expected no error for lazy pool creation: %v", err)
	}
	defer pool.Close()
}

// TestWithServerName verifies that pool creation with WithServerName succeeds
// and does not eagerly connect.
func TestWithServerName(t *testing.T) {
	ctx := t.Context()
	pool, err := dbpool.NewConnectionPool(ctx, "postgres://localhost:9999/test",
		dbpool.WithServerName("my-cluster.dsql.us-east-1.on.aws"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()
}

// TestWithTokenGenerator_NotCalledAtCreation verifies that the token generator
// is not invoked at pool creation time. Connections are lazy, so the
// BeforeConnect hook should not fire until a connection is actually acquired.
func TestWithTokenGenerator_NotCalledAtCreation(t *testing.T) {
	ctx := t.Context()
	called := false
	pool, err := dbpool.NewConnectionPool(ctx, "postgres://localhost:9999/test",
		dbpool.WithTokenGenerator(func(_ context.Context) (string, error) {
			called = true
			return "test-token", nil
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()
	if called {
		t.Error("token generator must not be called during pool creation (connections are lazy)")
	}
}

// TestWithTokenGenerator_ErrorPropagated verifies that a token generator error
// surfaces as the connection error when WithAcquireConnection is true.
//
// pgxpool calls BeforeConnect before the TCP dial, so this test does not need a
// reachable database server: the token generator fails first and the error is
// wrapped and returned by NewConnectionPool.
func TestWithTokenGenerator_ErrorPropagated(t *testing.T) {
	ctx := t.Context()
	tokenErr := errors.New("token generation failed")
	_, err := dbpool.NewConnectionPool(ctx, "postgres://localhost:9999/test",
		dbpool.WithTokenGenerator(func(_ context.Context) (string, error) {
			return "", tokenErr
		}),
		dbpool.WithAcquireConnection(true),
	)
	if err == nil {
		t.Fatal("expected error when token generator fails, got nil")
	}
	if !errors.Is(err, tokenErr) {
		t.Errorf("expected error chain to contain tokenErr, got: %v", err)
	}
	if !strings.Contains(err.Error(), "failed to acquire initial connection") {
		t.Errorf("expected 'failed to acquire initial connection' in error, got: %v", err)
	}
}

// TestWithAcquireConnection_Failure verifies that WithAcquireConnection(true)
// returns an error when the database server is unreachable.
func TestWithAcquireConnection_Failure(t *testing.T) {
	ctx := t.Context()
	_, err := dbpool.NewConnectionPool(ctx, "postgres://localhost:9999/test",
		dbpool.WithAcquireConnection(true),
	)
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
	if !strings.Contains(err.Error(), "failed to acquire initial connection") {
		t.Errorf("expected 'failed to acquire initial connection' in error, got: %v", err)
	}
}
