// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"os"
)

type fsKey int

var fsKeyVal fsKey

// ContextWithFS returns a new context that contains the provided instances
// of ReadFileFS stored with as a value within it.
func ContextWithFS(ctx context.Context, container ...ReadFileFS) context.Context {
	return context.WithValue(ctx, fsKeyVal, container)
}

// FSFromContext returns the list of ReadFileFS instances, if any,
// stored within the context.
func FSFromContext(ctx context.Context) ([]ReadFileFS, bool) {
	c, ok := ctx.Value(fsKeyVal).([]ReadFileFS)
	return c, ok
}

// FSreadFile is like FSOpen but calls ReadFile instead of Open.
func FSReadFile(ctx context.Context, name string) ([]byte, error) {
	if fss, ok := FSFromContext(ctx); ok {
		for _, fs := range fss {
			if data, err := fs.ReadFileCtx(ctx, name); err == nil {
				return data, nil
			}
		}
	}
	return os.ReadFile(name)
}
