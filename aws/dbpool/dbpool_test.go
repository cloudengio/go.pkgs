// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dbpool_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloudeng.io/aws/dbpool"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDSN = "postgres://localhost:9999/test"

func mustParseConfig(t *testing.T, dsn string) *pgxpool.Config {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	return cfg
}

// TestConfigWithOverrides verifies that ConfigWithOverrides applies field
// overrides correctly and rejects malformed connection strings.
func TestConfigWithOverrides(t *testing.T) {
	t.Run("InvalidConnectionString", func(t *testing.T) {
		_, err := dbpool.ConfigWithOverrides("not::a::valid::connection::string", "", "", "", 0)
		if err == nil {
			t.Fatal("expected error for invalid connection string, got nil")
		}
		if !strings.Contains(err.Error(), "failed to parse connection string") {
			t.Errorf("expected 'failed to parse connection string' in error, got: %v", err)
		}
	})

	t.Run("NoOverrides", func(t *testing.T) {
		cfg, err := dbpool.ConfigWithOverrides(testDSN, "", "", "", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ConnConfig.Host != "localhost" {
			t.Errorf("host: got %q, want %q", cfg.ConnConfig.Host, "localhost")
		}
		if cfg.ConnConfig.Port != 9999 {
			t.Errorf("port: got %d, want %d", cfg.ConnConfig.Port, 9999)
		}
		if cfg.ConnConfig.Database != "test" {
			t.Errorf("database: got %q, want %q", cfg.ConnConfig.Database, "test")
		}
	})

	t.Run("WithOverrides", func(t *testing.T) {
		cfg, err := dbpool.ConfigWithOverrides(testDSN, "mydb", "myuser", "myhost", 5433)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ConnConfig.Database != "mydb" {
			t.Errorf("database: got %q, want %q", cfg.ConnConfig.Database, "mydb")
		}
		if cfg.ConnConfig.User != "myuser" {
			t.Errorf("user: got %q, want %q", cfg.ConnConfig.User, "myuser")
		}
		if cfg.ConnConfig.Host != "myhost" {
			t.Errorf("host: got %q, want %q", cfg.ConnConfig.Host, "myhost")
		}
		if cfg.ConnConfig.Port != 5433 {
			t.Errorf("port: got %d, want %d", cfg.ConnConfig.Port, 5433)
		}
	})
}

// TestPoolCreation verifies that a pool can be created without immediately
// connecting. pgxpool is lazy: connections are made on demand, not at pool
// creation time, so an unreachable host is not an error here.
func TestPoolCreation(t *testing.T) {
	ctx := t.Context()
	pool, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN))
	if err != nil {
		t.Fatalf("expected no error for lazy pool creation: %v", err)
	}
	defer pool.Close()
}

// TestWithServerName verifies that pool creation with WithServerName succeeds
// and does not eagerly connect.
func TestWithServerName(t *testing.T) {
	ctx := t.Context()
	pool, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
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
	pool, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
		dbpool.WithTokenGenerator(func(_ context.Context, _ aws.Config) (string, error) {
			called = true
			return "test-token", nil
		}, 15*time.Minute),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()
	if called {
		t.Error("token generator must not be called during pool creation (connections are lazy)")
	}
}

// TestWithTokenGenerator_ExpirationTooShort verifies that a token expiration
// of 10 seconds or less is rejected at pool creation time.
func TestWithTokenGenerator_ExpirationTooShort(t *testing.T) {
	ctx := t.Context()
	for _, d := range []time.Duration{0, time.Second, 10 * time.Second} {
		_, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
			dbpool.WithTokenGenerator(func(_ context.Context, _ aws.Config) (string, error) {
				return "tok", nil
			}, d),
		)
		if err == nil {
			t.Errorf("expected error for token expiration %v, got nil", d)
		}
	}
}

// TestWithTokenGenerator_SetsMaxConnLifetime verifies that NewConnectionPool
// sets MaxConnLifetime to tokenExpiration minus 10 seconds.
func TestWithTokenGenerator_SetsMaxConnLifetime(t *testing.T) {
	ctx := t.Context()
	expiration := 15 * time.Minute
	pool, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
		dbpool.WithTokenGenerator(func(_ context.Context, _ aws.Config) (string, error) {
			return "tok", nil
		}, expiration),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pool.Close()
	if got, want := pool.Config().MaxConnLifetime, expiration-10*time.Second; got != want {
		t.Errorf("MaxConnLifetime: got %v, want %v", got, want)
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
	_, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
		dbpool.WithTokenGenerator(func(_ context.Context, _ aws.Config) (string, error) {
			return "", tokenErr
		}, 15*time.Minute),
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
	_, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
		dbpool.WithAcquireConnection(true),
	)
	if err == nil {
		t.Fatal("expected error for unreachable host, got nil")
	}
	if !strings.Contains(err.Error(), "failed to acquire initial connection") {
		t.Errorf("expected 'failed to acquire initial connection' in error, got: %v", err)
	}
}

// TestWithAWSConfig verifies that an explicit AWS config is used by the token
// generator in preference to whatever might be in the context.
func TestWithAWSConfig(t *testing.T) {
	ctx := t.Context()
	explicitCfg := aws.Config{Region: "us-west-2"}
	var capturedCfg aws.Config
	tokenErr := errors.New("sentinel")
	_, err := dbpool.NewConnectionPool(ctx, mustParseConfig(t, testDSN),
		dbpool.WithAWSConfig(explicitCfg),
		dbpool.WithTokenGenerator(func(_ context.Context, cfg aws.Config) (string, error) {
			capturedCfg = cfg
			return "", tokenErr
		}, 15*time.Minute),
		dbpool.WithAcquireConnection(true),
	)
	if !errors.Is(err, tokenErr) {
		t.Fatalf("expected tokenErr in error chain, got: %v", err)
	}
	if capturedCfg.Region != explicitCfg.Region {
		t.Errorf("token generator received region %q, want %q", capturedCfg.Region, explicitCfg.Region)
	}
}
