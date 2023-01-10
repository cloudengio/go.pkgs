// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package s3fs

import (
	"io/fs"
	"time"
)

type s3fs struct {
}

func NewS3FS() fs.FS {

}

func (fs *s3fs) Open(name string) (fs.File, error) {

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
