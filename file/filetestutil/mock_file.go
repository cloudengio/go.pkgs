// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"io"
	"io/fs"
	"time"
)

type BufferCloser struct {
	*bytes.Buffer
}

func (rc *BufferCloser) Close() error {
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

func NewInfo(name string, size int, mode fs.FileMode, mod time.Time, dir bool, sys interface{}) fs.FileInfo {
	return &fileinfo{
		name: name,
		size: int64(size),
		mode: mode,
		mod:  mod,
		dir:  dir,
		sys:  sys,
	}
}

type file struct {
	rd   io.ReadCloser
	info fs.FileInfo
}

func NewFile(rd io.ReadCloser, info fs.FileInfo) fs.File {
	return &file{
		rd:   rd,
		info: info,
	}
}

func (f *file) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *file) Read(buf []byte) (int, error) {
	return f.rd.Read(buf)
}

func (f *file) Close() error {
	return f.rd.Close()
}
