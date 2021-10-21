// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package awsconfig provides support for obtaining configuration and
// associated credentials information for use with AWS.
package awsconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ConfigOption represents an option to Load.
type ConfigOption func(o *options)

type options struct {
	passthrough []func(*config.LoadOptions) error
}

// WithConfigOptions will pass the supplied options from the aws config
// package.
func WithConfigOptions(fn ...func(*config.LoadOptions) error) ConfigOption {
	return func(o *options) {
		o.passthrough = append(o.passthrough, fn...)
	}
}

// Load attempts to load configuration information from multiple sources,
// including the current process' environment, shared configuration files
// (by default $HOME/.aws) and also from ec2 instance metadata (currently
// for the AWS region).
func Load(ctx context.Context, opts ...ConfigOption) (aws.Config, error) {
	o := &options{}
	for _, fn := range opts {
		fn(o)
	}
	return config.LoadDefaultConfig(ctx, o.passthrough...)
}

// AccountID uses the sts service to obtain the calling processes
// Amazon Account ID (number).
func AccountID(ctx context.Context, cfg aws.Config) (string, error) {
	svc := sts.New(sts.Options{
		Region:      cfg.Region,
		Credentials: cfg.Credentials,
	})
	output, err := svc.GetCallerIdentity(ctx,
		&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *output.Account, nil
}

// DebugPrintConfig dumps the aws.Config to help with debugging configuration
// issues. It displays the types of the fields that can't be directly printed.
func DebugPrintConfig(ctx context.Context, out io.Writer, cfg aws.Config) error {
	fmt.Fprintf(out, "region: %v\n", cfg.Region)
	fmt.Fprintf(out, "credentials provider: %T\n", cfg.Credentials)
	if cp := cfg.Credentials; cp != nil {
		creds, err := cp.Retrieve(ctx)
		if err == nil {
			creds.SecretAccessKey = ""
			creds.SessionToken = ""
			buf := &strings.Builder{}
			enc := json.NewEncoder(buf)
			enc.SetIndent("       ", "  ")
			if err := enc.Encode(creds); err != nil {
				return err
			}
			fmt.Fprintf(out, "credentials: %v\n", buf.String())
		} else {
			fmt.Printf("credentials: %v\n", err)
		}
	}
	for _, cs := range cfg.ConfigSources {
		fmt.Fprintf(out, "config source: %T\n", cs)
	}
	fmt.Fprintf(out, "http client: %T\n", cfg.HTTPClient)
	fmt.Fprintf(out, "endpoint resolver: %T\n", cfg.EndpointResolver)
	fmt.Fprintf(out, "retryer: %T\n", cfg.Retryer)
	fmt.Fprintf(out, "logger: %T\n", cfg.Logger)
	fmt.Fprintf(out, "client log mode: %v\n", cfg.ClientLogMode)
	return nil
}
