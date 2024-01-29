// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Option represents an option to New.
type Option func(o *options)

type options struct {
	s3options s3.Options
	client    Client
	scanSize  int
}

// WithS3Options wraps s3.Options for use when creating an s3.Client.
func WithS3Options(opts ...func(*s3.Options)) Option {
	return func(o *options) {
		for _, fn := range opts {
			fn(&o.s3options)
		}
	}
}

// WithS3Client specifies the s3.Client to use. If not specified, a new is created.
func WithS3Client(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// WithScanSize sets the number of items to fetch in a single remote api
// invocation for operations such as DeleteAll which may require
// iterating over a range of objects.
func WithScanSize(s int) Option {
	return func(o *options) {
		o.scanSize = s
	}
}

type s3fs struct {
	client  Client
	options options
}

// New creates a new instance of filewalk.FS backed by S3.
func New(cfg aws.Config, options ...Option) filewalk.FS {
	s3fs := &s3fs{}
	for _, fn := range options {
		fn(&s3fs.options)
	}
	s3fs.client = s3fs.options.client
	if s3fs.client == nil {
		s3fs.client = s3.NewFromConfig(cfg)
	}
	return s3fs
}

func NewObjectFS(cfg aws.Config, options ...Option) file.ObjectFS {
	return New(cfg, options...).(file.ObjectFS)
}

// Scheme implements fs.FS.
func (s3fs *s3fs) Scheme() string {
	return "s3"
}

// Open implements fs.FS.
func (s3fs *s3fs) Open(name string) (fs.File, error) {
	return s3fs.OpenCtx(context.Background(), name)
}

// OpenCtx implements file.FS.
func (s3fs *s3fs) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	match, res, err := getObject(ctx, s3fs.client, name)
	if err != nil {
		return nil, err
	}
	key := match.Key
	return &s3Readble{
		obj:    res,
		match:  match,
		client: s3fs.client,
		isDir:  key[len(key)-1] == '/',
	}, nil
}

func (s3fs *s3fs) Readlink(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("Readlink is not implemented for s3")
}

func (s3fs *s3fs) Stat(ctx context.Context, name string) (file.Info, error) {
	match := cloudpath.AWSS3Matcher(name)
	if len(match.Matched) == 0 {
		return file.Info{}, fmt.Errorf("invalid s3 path: %v", name)
	}
	return objectStat(ctx, s3fs.client, match)
}

func (s3fs *s3fs) Lstat(ctx context.Context, path string) (file.Info, error) {
	return s3fs.Stat(ctx, path)
}

func (s3fs *s3fs) Join(components ...string) string {
	return path.Join(components...)
}

func (s3fs *s3fs) Base(p string) string {
	return path.Base(p)
}

func (s3fs *s3fs) IsPermissionError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDenied"
	}
	return false
}

func (s3fs *s3fs) IsNotExist(err error) bool {
	var nsk *types.NoSuchKey
	var nsb *types.NoSuchBucket
	return errors.As(err, &nsk) || errors.As(err, &nsb)
}

type s3xattr struct {
	owner string
	obj   any
}

func (s3fs *s3fs) XAttr(_ context.Context, _ string, info file.Info) (file.XAttr, error) {
	sys := info.Sys()
	if v, ok := sys.(s3xattr); ok {
		return file.XAttr{User: v.owner}, nil
	}
	return file.XAttr{}, nil
}

func (s3fs *s3fs) SysXAttr(existing any, merge file.XAttr) any {
	switch v := existing.(type) {
	case s3xattr:
		return s3xattr{owner: merge.User, obj: v.obj}
	}
	return existing
}

type s3Readble struct {
	obj    *s3.GetObjectOutput
	isDir  bool
	client Client
	match  cloudpath.Match
}

func (f *s3Readble) Stat() (fs.FileInfo, error) {
	return objectStat(context.Background(), f.client, f.match)
}

func (f *s3Readble) Read(p []byte) (int, error) {
	return f.obj.Body.Read(p)
}

func (f *s3Readble) Close() error {
	return f.obj.Body.Close()
}
