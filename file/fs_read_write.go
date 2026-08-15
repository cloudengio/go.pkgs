// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
)

type readWriteFSKey int

var readWriteFSKeyVal readWriteFSKey

// ContextWithReadWriteFS returns a new context that contains the provided
// instance of ReadWriteFileFS.
func ContextWithReadWriteFS(ctx context.Context, fs ReadWriteFileFS) context.Context {
	return context.WithValue(ctx, readWriteFSKeyVal, fs)
}

// ReadWriteFSFromContext returns the ReadWriteFileFS instance, if any,
// stored within the context.
func ReadWriteFSFromContext(ctx context.Context) (ReadWriteFileFS, bool) {
	c, ok := ctx.Value(readWriteFSKeyVal).(ReadWriteFileFS)
	return c, ok
}

// ContextWithReadWriteFileFS is an alias for ContextWithReadWriteFS.
func ContextWithReadWriteFileFS(ctx context.Context, fs ReadWriteFileFS) context.Context {
	return ContextWithReadWriteFS(ctx, fs)
}

// ReadWriteFileFSFromContext is an alias for ReadWriteFSFromContext.
func ReadWriteFileFSFromContext(ctx context.Context) (ReadWriteFileFS, bool) {
	return ReadWriteFSFromContext(ctx)
}

// ContextWithoutReadWriteFS returns a new context without a ReadWriteFileFS.
func ContextWithoutReadWriteFS(ctx context.Context) context.Context {
	return context.WithValue(ctx, readWriteFSKeyVal, nil)
}
