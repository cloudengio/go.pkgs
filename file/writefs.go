// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"io"
	"io/fs"
)

// FS is like fs.FS but with a context parameter.
type FS interface {
	Open(ctx context.Context, name string) (fs.File, error)
}

// WriteFS extends FS to add a Create method.
type WriteFS interface {
	FS
	Create(ctx context.Context, name string, mode fs.FileMode) (io.WriteCloser, string, error)
}

// FSFromFS wraps an fs.FS to implement file.FS.
func FSFromFS(fs fs.FS) FS {
	return &fsFromFS{fs}
}

type fsFromFS struct {
	fs fs.FS
}

func (f *fsFromFS) Open(ctx context.Context, name string) (fs.File, error) {
	return f.fs.Open(name)
}
