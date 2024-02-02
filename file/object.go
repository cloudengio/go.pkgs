// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"io/fs"
)

// ObjectFS represents a writeable object store.
type ObjectFS interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Put(ctx context.Context, path string, perm fs.FileMode, data []byte) error
	EnsurePrefix(ctx context.Context, path string, perm fs.FileMode) error
	Delete(ctx context.Context, path string) error
	// DeleteAll delets all objects with the specified prefix.
	DeleteAll(ctx context.Context, prefix string) error
}
