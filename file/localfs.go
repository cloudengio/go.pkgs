// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
)

// Local represents the local filesystem. It implements FS and ObjectFS.
type Local struct{}

// LocalFS returns an instance of file.FS that provides access to the
// local filesystem.
func LocalFS() *Local {
	return &Local{}
}

func (f *Local) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *Local) Scheme() string {
	return "file"
}

func (f *Local) OpenCtx(_ context.Context, name string) (fs.File, error) {
	return os.Open(name)
}

func (f *Local) Readlink(_ context.Context, path string) (string, error) {
	return os.Readlink(path)
}

func (f *Local) Stat(_ context.Context, path string) (Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Info{}, err
	}
	return NewInfoFromFileInfo(info), nil
}

func (f *Local) Lstat(_ context.Context, path string) (Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Info{}, err
	}
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		return symlinkInfo(path, info)
	}
	return NewInfoFromFileInfo(info), nil
}

func (f *Local) Join(components ...string) string {
	return filepath.Join(components...)
}

func (f *Local) Base(path string) string {
	return filepath.Base(path)
}

func (f *Local) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (f *Local) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (f *Local) XAttr(_ context.Context, name string, info Info) (XAttr, error) {
	return xAttr(name, info)
}

func (f *Local) SysXAttr(existing any, merge XAttr) any {
	return mergeXAttr(existing, merge)
}

func (f *Local) Put(ctx context.Context, path string, perm fs.FileMode, data []byte) error {
	return os.WriteFile(path, data, perm)
}

func (f *Local) Get(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *Local) Delete(_ context.Context, path string) error {
	return os.Remove(path)
}

func (f *Local) DeleteAll(_ context.Context, path string) error {
	return os.RemoveAll(path)
}

func (f *Local) EnsurePrefix(_ context.Context, path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}
