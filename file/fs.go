// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"io/fs"
)

// FS extends fs.FS with OpenCtx.
type FS interface {
	fs.FS
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}

// WrapFS wraps an fs.FS to implement file.FS.
func WrapFS(fs fs.FS) FS {
	return &fsFromFS{fs}
}

type fsFromFS struct {
	fs.FS
}

func (f *fsFromFS) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	return f.Open(name)
}
