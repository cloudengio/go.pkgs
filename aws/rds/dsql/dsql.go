// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package dsql provides utilities for working with AWS DSQL, including
// generating authentication tokens and managing DSQL-related VPC endpoints.
package dsql

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/dbpool"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dsql/auth"
	"github.com/aws/aws-sdk-go-v2/service/dsql"
)

var clusterIDRE = regexp.MustCompile(`^[a-z0-9]{26}$`)

// clusterIDFromIdentifier returns the bare 26-character cluster ID from either
// a raw ID or a full endpoint hostname (e.g. "<id>.dsql.us-east-1.on.aws").
func clusterIDFromIdentifier(id string) (string, error) {
	candidate := id
	if strings.ContainsRune(id, '.') {
		candidate = id[:strings.IndexByte(id, '.')]
	}
	if !clusterIDRE.MatchString(candidate) {
		return "", fmt.Errorf("cannot extract a valid cluster ID from %q: expected 26 lowercase alphanumeric characters", id)
	}
	return candidate, nil
}

// WithTokenExpiration returns a function that can be passed to GenerateToken
// to set the expiration time of the generated token.
func WithTokenExpiration(expiration time.Duration) func(o *auth.TokenOptions) {
	return func(o *auth.TokenOptions) {
		o.ExpiresIn = expiration
	}
}

// GenerateToken creates a 15-minute SigV4 signed authentication token.
func GenerateToken(ctx context.Context, endpoint string, admin bool, cfg aws.Config, opts ...func(*auth.TokenOptions)) (string, error) {
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
		return GenerateToken(ctx, endpoint, admin, cfg, opts...)
	}
}

// Client is a minimal interface for interacting with DSQL operations needed to manage VPC endpoints.
type Client interface {
	GetVpcEndpointServiceName(ctx context.Context, params *dsql.GetVpcEndpointServiceNameInput, optFns ...func(*dsql.Options)) (*dsql.GetVpcEndpointServiceNameOutput, error)
}

// Cluster represents a DSQL cluster and provides methods to retrieve information about it.
type Cluster struct {
	opts options
	id   string
}

type Option func(*options)

type options struct {
	client Client
}

// WithDSQLClient returns an Option that allows specifying a custom DSQL client
// implementation, which can be useful for testing or if you want to use a
// pre-configured client.
func WithDSQLClient(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// NewCluster creates a new Cluster instance for the given cluster ID.
func NewCluster(cfg aws.Config, id string, opts ...Option) (*Cluster, error) {
	var options options
	for _, opt := range opts {
		opt(&options)
	}
	if options.client == nil {
		options.client = dsql.NewFromConfig(cfg)
	}

	return &Cluster{
		opts: options,
		id:   id,
	}, nil
}

// GetPrivateLinkServiceName retrieves the VPC endpoint service name for the cluster.
func (c *Cluster) GetPrivateLinkServiceName(ctx context.Context) (string, error) {
	id, err := clusterIDFromIdentifier(c.id)
	if err != nil {
		return "", fmt.Errorf("failed to get DSQL service name: %w", err)
	}
	output, err := c.opts.client.GetVpcEndpointServiceName(ctx, &dsql.GetVpcEndpointServiceNameInput{
		Identifier: aws.String(id),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get DSQL service name: %w", err)
	}
	return aws.ToString(output.ClusterVpcEndpoint), nil
}

// PrivateLinkServiceName is a helper function that retrieves the VPC public
// endpoint and endpoint service name for a DSQL cluster given its cluster ID
// or endpoint hostname. service name for a given cluster ID using the aws.Config
// from the context to create a DSQL client. This is a convenient wrapper around
// Cluster.GetPrivateLinkServiceName. clusterID may be either a bare 26-character
// cluster ID or a full endpoint hostname (e.g. "<id>.dsql.us-east-1.on.aws").
func PrivateLinkServiceName(ctx context.Context, clusterID string) (publicEndpoint, endpointServiceName string, err error) {
	id, err := clusterIDFromIdentifier(clusterID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get DSQL service name: %w", err)
	}
	cfg, ok := awsconfig.FromContext(ctx)
	if !ok {
		return "", "", fmt.Errorf("aws config not found in context")
	}
	client := dsql.NewFromConfig(cfg)
	output, err := client.GetVpcEndpointServiceName(ctx, &dsql.GetVpcEndpointServiceNameInput{
		Identifier: aws.String(id),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get DSQL service name: %w", err)
	}
	publicEndpoint = aws.ToString(output.ClusterVpcEndpoint)
	endpointServiceName = aws.ToString(output.ServiceName)
	return publicEndpoint, endpointServiceName, nil
}
