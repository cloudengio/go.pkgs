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
	"strings"
	"time"

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
	delimiter string
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

// WithDelimiter sets the delimiter to use when listing objects,
// the default is /.
func WithDelimiter(d string) Option {
	return func(o *options) {
		o.delimiter = d
	}
}

type T struct {
	client  Client
	options options
}

// New creates a new instance of filewalk.FS backed by S3.
func New(cfg aws.Config, options ...Option) filewalk.FS {
	return NewS3FS(cfg, options...)
}

// NewS3FS creates a new instance of filewalk.FS and
// file.ObjectFS backed by S3.
func NewS3FS(cfg aws.Config, options ...Option) *T {
	s3fs := &T{}
	s3fs.options.delimiter = "/"
	s3fs.options.scanSize = 1000
	for _, fn := range options {
		fn(&s3fs.options)
	}
	s3fs.client = s3fs.options.client
	if s3fs.client == nil {
		s3fs.client = s3.NewFromConfig(cfg)
	}
	return s3fs
}

// Scheme implements fs.FS.
func (s3fs *T) Scheme() string {
	return "s3"
}

// Open implements fs.FS.
func (s3fs *T) Open(name string) (fs.File, error) {
	return s3fs.OpenCtx(context.Background(), name)
}

// OpenCtx implements file.FS.
func (s3fs *T) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	match, res, err := getObject(ctx, s3fs.client, name)
	if err != nil {
		return nil, err
	}
	key := match.Key
	return &s3Readble{
		obj:    res,
		match:  match,
		client: s3fs.client,
		isDir:  strings.HasSuffix(key, s3fs.options.delimiter),
	}, nil
}

func (s3fs *T) Readlink(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("Readlink is not implemented for s3")
}

func basename(path string, delim string) string {
	if len(path) == 0 {
		return ""
	}
	isdir := false
	for lp := len(path); lp > 0 && strings.HasSuffix(path, delim); lp = len(path) {
		path = path[0 : lp-1]
		isdir = true
	}
	if idx := strings.LastIndex(path, delim); idx >= 0 {
		if isdir {
			return path[idx+1:] + string(delim)
		}
		return path[idx+1:]
	}
	if isdir {
		path += string(delim)
	}
	return path
}

// Stat invokes a Head operation on objects only. If name ends in /
// (or the currently configured delimiter) it is considered to be a
// prefix and a file.Info is created that reflects that (ie IsDir()
// returns true).
func (s3fs *T) Stat(ctx context.Context, name string) (file.Info, error) {
	match := cloudpath.AWSS3Matcher(name)
	if len(match.Matched) == 0 {
		return file.Info{}, fmt.Errorf("invalid s3 path: %v", name)
	}
	if len(match.Key) > 0 && strings.HasSuffix(match.Key, s3fs.options.delimiter) {
		name = basename(match.Key, s3fs.options.delimiter)
		return file.NewInfo(name, 0, fs.ModeDir, time.Time{}, nil), nil
	}
	return objectStat(ctx, s3fs.client, match)
}

func (s3fs *T) Lstat(ctx context.Context, path string) (file.Info, error) {
	return s3fs.Stat(ctx, path)
}

// Join concatenates the supplied components ensuring to insert
// delimiters only when necessary, that is components ending
// or starting with / (or the currently configured delimiter)
// will not
func (s3fs *T) Join(components ...string) string {
	delim := s3fs.options.delimiter
	size := 0
	for _, c := range components {
		size += len(c)
	}
	if size == 0 {
		return ""
	}
	joined := make([]byte, 0, size+len(components)-1)
	for _, c := range components {
		if len(c) == 0 {
			continue
		}
		if lj := len(joined); lj > 0 && !strings.HasSuffix(string(joined), delim) && !strings.HasPrefix(c, delim) {
			joined = append(joined, delim...)
		}
		joined = append(joined, c...)
	}
	return string(joined)
}

func (s3fs *T) Base(p string) string {
	return path.Base(p)
}

func (s3fs *T) IsPermissionError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "AccessDenied"
	}
	return false
}

func (s3fs *T) IsNotExist(err error) bool {
	var nsk *types.NoSuchKey
	var nsb *types.NoSuchBucket
	return errors.As(err, &nsk) || errors.As(err, &nsb)
}

type s3xattr struct {
	owner string
	obj   any
}

func (s3fs *T) XAttr(_ context.Context, _ string, info file.Info) (file.XAttr, error) {
	sys := info.Sys()
	if v, ok := sys.(s3xattr); ok {
		return file.XAttr{User: v.owner}, nil
	}
	return file.XAttr{}, nil
}

func (s3fs *T) SysXAttr(existing any, merge file.XAttr) any {
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
