// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"cloudeng.io/file"
	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Option represents an option to New.
type Option func(o *options)

// Client represents the set of AWS S3 client methods used by s3fs.
type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type options struct {
	s3options s3.Options
	client    Client
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

type s3fs struct {
	client  Client
	options options
}

// New creates a new instance of file.FS backed by S3.
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

// Scheme implements fs.FS.
func (fs *s3fs) Scheme() string {
	return "s3"
}

// Open implements fs.FS.
func (fs *s3fs) Open(name string) (fs.File, error) {
	return fs.OpenCtx(context.Background(), name)
}

// OpenCtx implements file.FS.
func (fs *s3fs) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	match := cloudpath.AWSS3Matcher(name)
	if len(match.Matched) == 0 {
		return nil, fmt.Errorf("invalid s3 path: %v", name)
	}
	bucket := match.Volume
	key := strings.TrimPrefix(match.Key, "/")
	get := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	obj, err := fs.client.GetObject(ctx, &get)
	if err != nil {
		return nil, err
	}
	return &s3file{obj: obj, path: name}, nil
}

func (fs *s3fs) Readlink(_ context.Context, path string) (string, error) {
	return "", fmt.Errorf("Readlink is not implemented for s3")
}

func (fs *s3fs) Stat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfoFromFileInfo(info), nil
}

func (fs *s3fs) Lstat(ctx context.Context, path string) (file.Info, error) {
	return fs.Stat(ctx, path)
}

func (fs *s3fs) Join(components ...string) string {
	return path.Join(components...)
}

func (fs *s3fs) Base(p string) string {
	return path.Base(p)
}

func (fs *s3fs) IsPermissionError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDenied"
	}
	return false
}

func (fs *s3fs) IsNotExist(err error) bool {
	var nsk *types.NoSuchKey
	var nsb *types.NoSuchBucket
	return errors.As(err, &nsk) || errors.As(err, &nsb)
}

type s3xattr struct {
	XAttr file.XAttr
	obj   *s3.GetObjectOutput
}

func (fs *s3fs) XAttr(_ context.Context, name string, info file.Info) (file.XAttr, error) {
	sys := info.Sys()
	switch v := sys.(type) {
	case *s3xattr:
		return v.XAttr, nil
	}
	return file.XAttr{}, nil
}

func (fs *s3fs) SysXAttr(existing any, merge file.XAttr) any {
	switch v := existing.(type) {
	case *s3.GetObjectOutput:
		return &s3xattr{XAttr: merge, obj: v}
	case *s3xattr:
		return &s3xattr{XAttr: merge, obj: v.obj}
	}
	return nil
}

type s3file struct {
	obj  *s3.GetObjectOutput
	path string
}

func (f *s3file) Stat() (fs.FileInfo, error) {
	return file.NewInfo(
		f.path,
		aws.ToInt64(f.obj.ContentLength),
		0400,
		aws.ToTime(f.obj.LastModified),
		f.obj,
	), nil
}

func (f *s3file) Read(p []byte) (int, error) {
	return f.obj.Body.Read(p)
}

func (f *s3file) Close() error {
	return f.obj.Body.Close()
}
