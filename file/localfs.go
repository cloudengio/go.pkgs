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

type localfs struct{}

// LocalFS returns an instance of file.FS that provides access to the
// local filesystem.
func LocalFS() FS {
	return &localfs{}
}

func (f *localfs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (f *localfs) Scheme() string {
	return "file"
}

func (f *localfs) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	return os.Open(name)
}

func (f *localfs) Readlink(_ context.Context, path string) (string, error) {
	return os.Readlink(path)
}

func (f *localfs) Stat(_ context.Context, path string) (Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Info{}, err
	}
	return NewInfoFromFileInfo(info), nil
}

func (f *localfs) Lstat(_ context.Context, path string) (Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Info{}, err
	}
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		return symlinkInfo(path, info)
	}
	return NewInfoFromFileInfo(info), nil
}

func (f *localfs) Join(components ...string) string {
	return filepath.Join(components...)
}

func (f *localfs) Base(path string) string {
	return filepath.Base(path)
}

func (f *localfs) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (f *localfs) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (f *localfs) XAttr(_ context.Context, name string, info Info) (XAttr, error) {
	return xAttr(name, info)
}

func (f *localfs) SysXAttr(existing any, merge XAttr) any {
	return mergeXAttr(existing, merge)
}
