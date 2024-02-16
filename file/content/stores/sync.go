// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"context"

	"cloudeng.io/file/content"
)

// Sync represents a synchronous store for objects, ie. it implements
// content.ObjectStore. It uses an instance of content.FS to store and
// retrieve objects.
type Sync struct {
	fs content.FS
}

// New returns a new instance of Sync backed by the supplied
// content.FS and storing the specified objects encoded using the
// specified encodings.
func New(fs content.FS) *Sync {
	return &Sync{
		fs: fs,
	}
}

// EraseExisting deletes all contents of the store beneath root.
func (s *Sync) EraseExisting(ctx context.Context, root string) error {
	return eraseExisting(ctx, s.fs, root)
}

func (s *Sync) FS() content.FS {
	return s.fs
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Sync) Read(ctx context.Context, prefix, name string) (content.Type, []byte, error) {
	return read(ctx, s.fs, s.fs.Join(prefix, name))
}

// Write stores the data at the specified prefix and name in the store.
func (s *Sync) Write(ctx context.Context, prefix, name string, data []byte) error {
	return write(ctx, s.fs, prefix, name, data)
}
