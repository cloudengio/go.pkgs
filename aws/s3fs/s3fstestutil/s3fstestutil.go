// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fstestutil

import (
	"context"
	"fmt"

	"cloudeng.io/aws/s3fs"
	"cloudeng.io/file"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Option func(o *options)

type options struct {
	bucket            string
	stripLeadingSlash bool
}

func WithBucket(b string) Option {
	return func(o *options) {
		o.bucket = b
	}
}

func WithLeadingSlashStripped() Option {
	return func(o *options) {
		o.stripLeadingSlash = true
	}
}

func NewMockFS(fs file.FS, opts ...Option) s3fs.Client {
	m := &mfs{fs: fs}
	for _, fn := range opts {
		fn(&m.options)
	}
	return m
}

type mfs struct {
	options options
	fs      file.FS
}

func (m *mfs) GetObject(ctx context.Context, params *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if b := m.options.bucket; len(b) > 0 && *params.Bucket != b {
		return nil, fmt.Errorf("unknown bucket %q", *params.Bucket)
	}
	key := *params.Key
	if m.options.stripLeadingSlash && key[0] == '/' {
		key = key[1:]
	}
	file, err := m.fs.OpenCtx(ctx, key)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &s3.GetObjectOutput{
		ContentLength: fi.Size(),
		Body:          file,
	}, nil
}
