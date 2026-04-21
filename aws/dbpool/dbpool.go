// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dbpool

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"cloudeng.io/aws/awsconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option is a functional option for configuring the connection pool.
type Option func(o *options)

type options struct {
	serverName        string
	tokenGenerator    TokenGenerator
	tokenExpiration   time.Duration
	acquireConnection bool
	cfg               aws.Config
}

// WithServerName sets the TLS ServerName for connections in the pool.
// This is required for services like DSQL that use the ServerName for
// routing and authentication.
func WithServerName(serverName string) Option {
	return func(o *options) {
		o.serverName = serverName
	}
}

// WithTokenGenerator sets a custom TokenGenerator that will be called
// to generate a fresh authentication token for every new connection.
// This is essential for services like DSQL that require short-lived tokens.
func WithTokenGenerator(tokenGenerator TokenGenerator, tokenExpiration time.Duration) Option {
	return func(o *options) {
		o.tokenGenerator = tokenGenerator
		o.tokenExpiration = tokenExpiration
	}
}

// WithAcquireConnection forces the pool to acquire a connection during
// initialization.
// This can be used to validate the connection parameters and fail fast
// if there are issues.
func WithAcquireConnection(acquire bool) Option {
	return func(o *options) {
		o.acquireConnection = acquire
	}
}

// WithAWSConfig sets the AWS configuration to be used by the TokenGenerator.
// The default is to look for the config in the context, but this option allows
// it to be explicitly provided.
func WithAWSConfig(cfg aws.Config) Option {
	return func(o *options) {
		o.cfg = cfg
	}
}

// Pool is a thin wrapper around pgxpool.Pool that simplifies
// creating connection pools.
type Pool struct {
	*pgxpool.Pool
}

// TokenGenerator is a function type that generates an authentication token.
type TokenGenerator func(ctx context.Context, cfg aws.Config) (string, error)

// NewConnectionPool creates a new connection pool with the
// given configuration and options. If the WithServerName name
// option is used, the ServerName will be set in the TLS config for
// all connections. If a TokenGenerator is provided, it will be called
// to generate a fresh authentication token for every new connection
// and the pool's max connection lifetime will be set to the token
// expiration specified in WithTokenGenerator (minus 10 seconds)
// to ensure that connections are recycled before tokens expire.
func NewConnectionPool(ctx context.Context, poolConfig *pgxpool.Config, opts ...Option) (*Pool, error) {
	var options options
	cfg, ok := awsconfig.FromContext(ctx)
	if ok {
		options.cfg = cfg.Copy()
	}
	for _, fn := range opts {
		fn(&options)
	}

	// Use a custom TLS config to set the ServerName for services that
	// require it (eg. dsql)
	if options.serverName != "" {
		if poolConfig.ConnConfig.TLSConfig == nil {
			poolConfig.ConnConfig.TLSConfig = &tls.Config{}
		}
		poolConfig.ConnConfig.TLSConfig.ServerName = options.serverName
	}

	if options.tokenGenerator != nil {
		if options.tokenExpiration <= time.Second*10 {
			return nil, fmt.Errorf("token expiration must be greater than 10 seconds")
		}
		poolConfig.MaxConnLifetime = options.tokenExpiration - (time.Second * 10)
		poolConfig.BeforeConnect = func(ctx context.Context, cc *pgx.ConnConfig) error {
			token, err := options.tokenGenerator(ctx, options.cfg)
			if err != nil {
				return fmt.Errorf("failed to generate token: %w", err)
			}
			cc.Password = token
			return nil
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if options.acquireConnection {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to acquire initial connection: %w", err)
		}
		conn.Release()
	}

	return &Pool{Pool: pool}, nil
}

// ConfigWithOverrides parses the connection string into a pgxpool.
// Config and applies any overrides for the database, user, host, or
// port if they are non-empty or non-zero.
func ConfigWithOverrides(connection string, database, user, host string, port uint16) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}
	if database != "" {
		poolConfig.ConnConfig.Database = database
	}
	if user != "" {
		poolConfig.ConnConfig.User = user
	}
	if host != "" {
		poolConfig.ConnConfig.Host = host
	}
	if port != 0 {
		poolConfig.ConnConfig.Port = port
	}
	return poolConfig, nil
}
