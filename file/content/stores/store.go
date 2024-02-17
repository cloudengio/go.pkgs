// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"

	"cloudeng.io/file/content"
)

// ReadFunc is called by ReadV for each object read from the store.
// If the read operation returned an error it is passed to ReadFunc and if
// then returned by ReadFunc it will cause the entire ReadV operation
// to terminate and return an error.
type ReadFunc func(ctx context.Context, prefix, name string, typ content.Type, data []byte, err error) error

// T represents a common interface for both synchronous and asynchronous
// stores.
type T interface {
	content.ObjectStore
	EraseExisting(ctx context.Context, root string) error
	FS() content.FS
	ReadV(ctx context.Context, prefix string, names []string, fn ReadFunc) error
	Finish(context.Context) error
}

// New returns a new instance of T with the specified concurrency.
func New(fs content.FS, concurrency int) T {
	if concurrency > 1 {
		return NewAsync(fs, concurrency)
	}
	return NewSync(fs)
}
