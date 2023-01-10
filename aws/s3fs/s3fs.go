// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"io/fs"
	"time"

	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/services/s3"
)

type Option func(o *options)

type options struct {
	s3options s3.Options
}

func WithS3Options(opts ...s3.Options) Option {
	return func(o *options) {
		for _, fn := range opts {
			fn(&o.s3options)
		}
	}
}

type s3fs struct {
	client  *s3.Client
	options options
}

func NewS3FS(ctx context.Context, cfg aws.Config, options ...Option) fs.FS {
	fs := &s3fs{}
	for _, fn := range options {
		fn(&fs.options)
	}
	fs.client = s3.NewFromConfig(cfg)
	return fs
}

func (fs *s3fs) Open(name string) (fs.File, error) {
	matcher := cloudpath.AWSS3Matcher(name)
	bucket := matcher.Volume
	path := matcher.Path
	get := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &path,
	}
	obj, err := fs.client.GetObject(context.Background(), &get, fs.options.s3options...)
}

type s3file struct {
}

func (f *s3file) Stat() (fs.FileInfo, error) {

}

func (f *s3file) Read(p []byte) (int, error) {

}

func (f *s3file) Close() error {

}

func (bc *BufferCloser) Close() error {
	return nil
}

type fileinfo struct {
	name string
	size int64
	mode fs.FileMode
	mod  time.Time
	dir  bool
	sys  interface{}
}

func (fi *fileinfo) Name() string {
	return fi.name
}

func (fi *fileinfo) Size() int64 {
	return fi.size
}

func (fi *fileinfo) Mode() fs.FileMode {
	return fi.mode
}

func (fi *fileinfo) ModTime() time.Time {
	return fi.mod
}

func (fi *fileinfo) IsDir() bool {
	return fi.dir
}

func (fi *fileinfo) Sys() interface{} {
	return fi.sys
}
