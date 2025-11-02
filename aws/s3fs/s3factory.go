// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"fmt"

	"cloudeng.io/aws/awsconfig"
)

// Factory wraps creating an S3FS with the configuration required to
// correctly initialize it.
type Factory struct {
	Config  awsconfig.AWSConfig
	Options []Option
}

func (f Factory) New(ctx context.Context) (*T, error) {
	if !f.Config.AWS {
		return nil, fmt.Errorf("AWS authentication must be enabled to use S3")
	}
	awsConfig, err := f.Config.Load(ctx)
	if err != nil {
		return nil, err
	}
	return NewS3FS(awsConfig, f.Options...), nil
}
