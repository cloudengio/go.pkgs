// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"io"
	"io/fs"
)

// BufferCloser adds an io.Closer to bytes.Buffer.
type BufferCloser struct {
	*bytes.Buffer
}

func (bc *BufferCloser) Close() error {
	return nil
}

type mockFile struct {
	rd   io.ReadCloser
	info fs.FileInfo
}

func NewFile(rd io.ReadCloser, info fs.FileInfo) fs.File {
	return &mockFile{
		rd:   rd,
		info: info,
	}
}

func (f *mockFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *mockFile) Read(buf []byte) (int, error) {
	return f.rd.Read(buf)
}

func (f *mockFile) Close() error {
	return f.rd.Close()
}
