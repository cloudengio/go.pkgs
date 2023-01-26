// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"fmt"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/file"
	"cloudeng.io/path/cloudpath"
)

// Factory implements file.FSFactory for AWS S3.
type Factory struct {
	Config awsconfig.AWSFlags
}

// New implements file.FSFactory.
func (d Factory) New(ctx context.Context, scheme string) (file.FS, error) {
	if !d.Config.AWS {
		return nil, fmt.Errorf("AWS authentication must be enabled to use S3")
	}
	awsConfig, err := awsconfig.LoadUsingFlags(ctx, d.Config)
	if err != nil {
		return nil, err
	}
	return New(awsConfig), nil
}

func (d Factory) NewFromMatch(ctx context.Context, match cloudpath.Match) (file.FS, error) {
	return d.New(ctx, match.Scheme)
}

var _ file.FSFactory = (*Factory)(nil)
