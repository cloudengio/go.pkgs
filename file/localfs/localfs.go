// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file"
	"cloudeng.io/file/largefile"
)

// T represents the local filesystem. It implements FS, ObjectFS
// and filewalk.FS
type T struct {
	opts options
}

type Option func(o *options)

type options struct {
	scannerOpenWait time.Duration
}

func WithScannerOpenWait(d time.Duration) Option {
	return func(o *options) {
		o.scannerOpenWait = d
	}
}

// NewLocalFS returns an instance of file.FS that provides access to the
// local filesystem.
func New(opts ...Option) *T {
	t := &T{}
	for _, fn := range opts {
		fn(&t.opts)
	}
	return t
}

func (f *T) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *T) Scheme() string {
	return "file"
}

func (f *T) OpenCtx(_ context.Context, name string) (fs.File, error) {
	return os.Open(name)
}

func (f *T) Readlink(_ context.Context, path string) (string, error) {
	return os.Readlink(path)
}

func (f *T) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (f *T) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	return f.ReadFile(name)
}

func (f *T) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (f *T) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (f *T) Stat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfoFromFileInfo(info), nil
}

func (f *T) Lstat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return file.Info{}, err
	}
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		return symlinkInfo(path, info)
	}
	return file.NewInfoFromFileInfo(info), nil
}

func (f *T) Join(components ...string) string {
	return filepath.Join(components...)
}

func (f *T) Base(path string) string {
	return filepath.Base(path)
}

func (f *T) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (f *T) IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func (f *T) XAttr(_ context.Context, name string, info file.Info) (file.XAttr, error) {
	return xAttr(name, info)
}

func (f *T) SysXAttr(existing any, merge file.XAttr) any {
	return mergeXAttr(existing, merge)
}

func (f *T) Put(_ context.Context, path string, perm fs.FileMode, data []byte) error {
	return os.WriteFile(path, data, perm)
}

func (f *T) Get(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *T) Delete(_ context.Context, path string) error {
	return os.Remove(path)
}

func (f *T) DeleteAll(_ context.Context, path string) error {
	return os.RemoveAll(path)
}

func (f *T) EnsurePrefix(_ context.Context, path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

type LargeFile struct {
	f         *os.File
	blockSize int
	size      int64
	digest    digests.Hash
}

const DefaultLargeFileBlockSize = 1024 * 1024 * 16 // Default block size is 16 MiB.

func NewLargeFile(filename string, blockSize int, digest digests.Hash) (*LargeFile, error) {
	if blockSize <= 0 {
		blockSize = DefaultLargeFileBlockSize
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, fs.ErrInvalid
	}
	size := info.Size()
	return &LargeFile{
		f:         f,
		blockSize: blockSize,
		size:      size,
		digest:    digest,
	}, nil
}

type noRetry struct{}

func (noRetry) IsRetryable() bool {
	return false
}

func (noRetry) BackoffDuration() (bool, time.Duration) {
	return false, 0
}

func (lf *LargeFile) Name() string {
	return lf.f.Name()
}

func (lf *LargeFile) ContentLengthAndBlockSize() (int64, int) {
	return lf.size, lf.blockSize
}

func (lf *LargeFile) Digest() digests.Hash {
	return lf.digest
}

func (lf *LargeFile) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	return reader{f: lf.f, at: from}, noRetry{}, nil
}

type reader struct {
	f  *os.File
	at int64
}

func (r reader) Read(p []byte) (int, error) {
	return r.f.ReadAt(p, r.at)
}

func (r reader) Close() error {
	return nil
}
