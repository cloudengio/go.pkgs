// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Option func(o *options)

type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type options struct {
	s3options s3.Options
	client    Client
}

func WithS3Options(opts ...func(*s3.Options)) Option {
	return func(o *options) {
		for _, fn := range opts {
			fn(&o.s3options)
		}
	}
}

func WithS3Client(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

type s3fs struct {
	client  Client
	options options
}

func New(cfg aws.Config, options ...Option) file.FS {
	fs := &s3fs{}
	for _, fn := range options {
		fn(&fs.options)
	}
	fs.client = fs.options.client
	if fs.client == nil {
		fs.client = s3.NewFromConfig(cfg)
	}
	return fs
}

func (fs *s3fs) Open(ctx context.Context, name string) (fs.File, error) {
	matcher := cloudpath.AWSS3Matcher(name)
	if matcher == nil {
		return nil, fmt.Errorf("invalid s3 path: %v", name)
	}
	fmt.Printf("... %#v\n", matcher)
	bucket := matcher.Volume
	path := matcher.Path
	get := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &path,
	}
	obj, err := fs.client.GetObject(ctx, &get)
	if err != nil {
		return nil, err
	}
	return &s3file{obj: obj, path: name}, nil
}

type s3file struct {
	obj  *s3.GetObjectOutput
	path string
}

func (f *s3file) Stat() (fs.FileInfo, error) {
	return &fileinfo{
		name: f.path,
		size: f.obj.ContentLength,
		mode: 0400,
		mod:  *f.obj.LastModified,
		dir:  false,
		sys:  f.obj,
	}, nil
}

func (f *s3file) Read(p []byte) (int, error) {
	return f.obj.Body.Read(p)
}

func (f *s3file) Close() error {
	return f.obj.Body.Close()
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
