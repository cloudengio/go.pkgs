// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"io/fs"
	"os"
)

type fsKey int

var fsKeyVal fsKey

// ContextWithFS returns a new context that contains the provided instance
// of fs.ReadFileFS stored with as a valye within it.
func ContextWithFS(ctx context.Context, container fs.ReadFileFS) context.Context {
	return context.WithValue(ctx, fsKeyVal, container)
}

// FSFromContext returns the fs.ReadFileFS instance, if any,
// stored within the context.
func FSFromContext(ctx context.Context) (fs.ReadFileFS, bool) {
	c, ok := ctx.Value(fsKeyVal).(fs.ReadFileFS)
	return c, ok
}

// FSOpen will open name using the context's fs.ReadFileFS instance if
// one is present, otherwise it will use os.Open.
func FSOpen(ctx context.Context, name string) (fs.File, error) {
	if fs, ok := FSFromContext(ctx); ok {
		return fs.Open(name)
	}
	return os.Open(name)

}

// FSreadAll will read name using the context's fs.ReadFileFS instance if
// one is present, otherwise it will use os.ReadFile.
func FSReadFile(ctx context.Context, name string) ([]byte, error) {
	if fs, ok := FSFromContext(ctx); ok {
		return fs.ReadFile(name)
	}
	return os.ReadFile(name)
}
