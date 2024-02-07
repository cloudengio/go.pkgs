// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"cloudeng.io/file"
)

// T represents the local filesystem. It implements FS, ObjectFS
// and filewalk.FS
type T struct{}

// NewLocalFS returns an instance of file.FS that provides access to the
// local filesystem.
func New() *T {
	return &T{}
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
	return os.IsNotExist(err)
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
