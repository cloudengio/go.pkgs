// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"cloudeng.io/file"
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

// WithScannerOpenWait configures the amount of time to wait for the scanner
// to open a file before timing out. If zero then no timeout is applied
// and open is called directly.
func WithScannerOpenWait(d time.Duration) Option {
	return func(o *options) {
		o.scannerOpenWait = d
	}
}

// New returns an instance of file.FS that provides access to the
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

// R represents a local filesystem tree rooted at a specified directory.
type R struct {
	root string
	*T
}

// NewTree returns an instance of file.FS that provides access to the
// local filesystem tree rooted at the specified directory.
func NewTree(root string, opts ...Option) *R {
	r := &R{root: root}
	r.T = New(opts...)
	return r
}

func (f *R) Open(name string) (fs.File, error) {
	return f.T.Open(filepath.Join(f.root, name))
}

func (f *R) Scheme() string {
	return "file"
}

func (f *R) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	return f.T.OpenCtx(ctx, filepath.Join(f.root, name))
}

func (f *R) Readlink(ctx context.Context, path string) (string, error) {
	return f.T.Readlink(ctx, filepath.Join(f.root, path))
}

func (f *R) ReadFile(name string) ([]byte, error) {
	return f.T.ReadFile(filepath.Join(f.root, name))
}

func (f *R) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	return f.T.ReadFileCtx(ctx, filepath.Join(f.root, name))
}

func (f *R) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return f.T.WriteFile(filepath.Join(f.root, name), data, perm)
}

func (f *R) WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error {
	return f.T.WriteFileCtx(ctx, filepath.Join(f.root, name), data, perm)
}

func (f *R) Stat(ctx context.Context, path string) (file.Info, error) {
	return f.T.Stat(ctx, filepath.Join(f.root, path))
}

func (f *R) Lstat(ctx context.Context, path string) (file.Info, error) {
	return f.T.Lstat(ctx, filepath.Join(f.root, path))
}

func (f *R) Join(components ...string) string {
	return filepath.Join(components...)
}

func (f *R) Base(path string) string {
	return filepath.Base(path)
}

func (f *R) IsPermissionError(err error) bool {
	return f.T.IsPermissionError(err)
}

func (f *R) IsNotExist(err error) bool {
	return f.T.IsNotExist(err)
}

func (f *R) XAttr(ctx context.Context, name string, info file.Info) (file.XAttr, error) {
	return f.T.XAttr(ctx, filepath.Join(f.root, name), info)
}

func (f *R) SysXAttr(existing any, merge file.XAttr) any {
	return f.T.SysXAttr(existing, merge)
}

func (f *R) Put(ctx context.Context, path string, perm fs.FileMode, data []byte) error {
	return f.T.Put(ctx, filepath.Join(f.root, path), perm, data)
}

func (f *R) Get(ctx context.Context, path string) ([]byte, error) {
	return f.T.Get(ctx, filepath.Join(f.root, path))
}

func (f *R) Delete(ctx context.Context, path string) error {
	return f.T.Delete(ctx, filepath.Join(f.root, path))
}

func (f *R) DeleteAll(ctx context.Context, path string) error {
	return f.T.DeleteAll(ctx, filepath.Join(f.root, path))
}

func (f *R) EnsurePrefix(ctx context.Context, path string, perm fs.FileMode) error {
	return f.T.EnsurePrefix(ctx, filepath.Join(f.root, path), perm)
}
