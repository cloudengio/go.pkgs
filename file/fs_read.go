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

// ContextWithFS returns a new context that contains the provided instances
// of fs.ReadFileFS stored with as a value within it.
func ContextWithFS(ctx context.Context, container ...fs.ReadFileFS) context.Context {
	return context.WithValue(ctx, fsKeyVal, container)
}

// FSFromContext returns the list of fs.ReadFileFS instancees, if any,
// stored within the context.
func FSFromContext(ctx context.Context) ([]fs.ReadFileFS, bool) {
	c, ok := ctx.Value(fsKeyVal).([]fs.ReadFileFS)
	return c, ok
}

// FSOpen will attempt to open filename using the context's set of
// fs.ReadFileFS instances (if any), in the order in which they were
// provided to ContextWithFS, returning the first successful result.
// If no fs.ReadFileFS instances are present in the context or
// none successfully open the file, then os.Open is used.
func FSOpen(ctx context.Context, filename string) (fs.File, error) {
	if fss, ok := FSFromContext(ctx); ok {
		for _, fs := range fss {
			if f, err := fs.Open(filename); err == nil {
				return f, nil
			}
		}
	}
	return os.Open(filename)
}

// FSreadFile is like FSOpen but calls ReadFile instead of Open.
func FSReadFile(ctx context.Context, name string) ([]byte, error) {
	if fss, ok := FSFromContext(ctx); ok {
		for _, fs := range fss {
			if data, err := fs.ReadFile(name); err == nil {
				return data, nil
			}
		}
	}
	return os.ReadFile(name)
}
