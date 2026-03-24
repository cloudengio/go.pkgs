// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package aurora

import (
	"context"
	"fmt"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/dbpool"
	"github.com/aws/aws-sdk-go-v2/feature/dsql/auth"
)

// GenerateDSQLToken creates a 15-minute SigV4 signed authentication token.
func GenerateDSQLToken(ctx context.Context, endpoint string) (string, error) {
	cfg, ok := awsconfig.FromContext(ctx)
	if !ok {
		return "", awsconfig.ErrConfigNotFound
	}
	// Generate the token using the built-in DSQL auth package.
	// This performs a local cryptographic signing operation (no network call is made).
	token, err := auth.GenerateDBConnectAdminAuthToken(
		ctx,
		endpoint,
		cfg.Region,
		cfg.Credentials,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate DSQL auth token: %w", err)
	}

	return token, nil
}

// TokenGenerator returns a dbpool.TokenGenerator that generates DSQL authentication tokens.
func TokenGenerator(ctx context.Context, endpoint string) dbpool.TokenGenerator {
	return func(ctx context.Context) (string, error) {
		return GenerateDSQLToken(ctx, endpoint)
	}
}
