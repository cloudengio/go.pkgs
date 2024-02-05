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
)

// Factory implements file.FSFactory for AWS S3.
type Factory struct {
	Config  awsconfig.AWSFlags
	Options []Option
}

func (f Factory) newFS(ctx context.Context) (*T, error) {
	if !f.Config.AWS {
		return nil, fmt.Errorf("AWS authentication must be enabled to use S3")
	}
	awsConfig, err := awsconfig.LoadUsingFlags(ctx, f.Config)
	if err != nil {
		return nil, err
	}
	return NewS3FS(awsConfig, f.Options...), nil
}

// New implements file.FSFactory.
func (f Factory) NewFS(ctx context.Context) (file.FS, error) {
	return f.newFS(ctx)
}

func (f Factory) NewObjectFS(ctx context.Context) (file.ObjectFS, error) {
	return f.newFS(ctx)
}

var _ file.FSFactory = (*Factory)(nil)
var _ file.ObjectFSFactory = (*Factory)(nil)
