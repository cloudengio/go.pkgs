// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package rds

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/aws/dbpool"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dsql/auth"
)

// WithDSQLTokenExpiration returns a function that can be passed to
// GenerateDSQLToken to set the expiration time of the generated
// token.
func WithDSQLTokenExpiration(expiration time.Duration) func(o *auth.TokenOptions) {
	return func(o *auth.TokenOptions) {
		o.ExpiresIn = expiration
	}
}

// GenerateDSQLToken creates a 15-minute SigV4 signed authentication token.
func GenerateDSQLToken(ctx context.Context, endpoint string, admin bool, cfg aws.Config, opts ...func(*auth.TokenOptions)) (string, error) {
	// Generate the token using the built-in DSQL auth package.
	// This performs a local cryptographic signing operation (no network call is made).
	var token string
	var err error
	if admin {
		token, err = auth.GenerateDBConnectAdminAuthToken(
			ctx,
			endpoint,
			cfg.Region,
			cfg.Credentials,
			opts...,
		)
	} else {
		token, err = auth.GenerateDbConnectAuthToken(
			ctx,
			endpoint,
			cfg.Region,
			cfg.Credentials,
			opts...,
		)
	}
	if err != nil {
		return "", fmt.Errorf("failed to generate DSQL auth token: %w", err)
	}
	return token, nil
}

// TokenGenerator returns a dbpool.TokenGenerator that generates DSQL authentication tokens.
func TokenGenerator(endpoint string, admin bool, opts ...func(*auth.TokenOptions)) dbpool.TokenGenerator {
	return func(ctx context.Context, cfg aws.Config) (string, error) {
		return GenerateDSQLToken(ctx, endpoint, admin, cfg, opts...)
	}
}
