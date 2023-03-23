// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsutil

import (
	"context"
	"os"
	"strings"
	"sync"

	"cloudeng.io/aws/awsconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
)

var (
	accountIDOnce sync.Once
	accountID     string
	accountIDErr  error
)

// AccountID returns the account id from the aws.Config and caches it
// locally.
func AccountID(ctx context.Context, cfg aws.Config) (string, error) {
	accountIDOnce.Do(func() {
		accountID, accountIDErr = awsconfig.AccountID(ctx, cfg)
	})
	return accountID, accountIDErr
}

// IsArn returns true if the supplied string is an ARN.
func IsARN(name string) bool {
	return strings.HasPrefix(name, "arn:aws:")
}

// Region obtains the AWS region either from the supplied config or from
// the environment.
func Region(_ context.Context, cfg aws.Config) string {
	if len(cfg.Region) > 0 {
		return cfg.Region
	}
	if r := os.Getenv("AWS_REGION"); len(r) > 0 {
		return r
	}
	return os.Getenv("AWS_DEFAULT_REGION")
}
