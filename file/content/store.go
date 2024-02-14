// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"cloudeng.io/file"
)

// FS represents the interface to a filesystem/object store that is used to
// back Store.
type FS interface {
	file.FS
	file.ObjectFS
}

// ObjectStore represents the interface used by Objects to store and retrieve
// their data.
type ObjectStore interface {
	Read(ctx context.Context, prefix, name string) (Type, []byte, error)
	Write(ctx context.Context, prefix, name string, data []byte) error
}

// Store represents an for objects. It uses an instance of content.FS
// to store and retrieve objects. The objects are stored in a hierarchy
// at the specified root prefix/path and all operations are relative to
// that root. It is intended to be backed by either a local or cloud
// filesystem.
type Store struct {
	fs      FS
	written int64
	read    int64
}

// NewStore returns a new instance of Store backed by the supplied
// content.FS and storing the specified objects encoded using the
// specified encodings.
func NewStore(fs FS) *Store {
	return &Store{
		fs: fs,
	}
}

// EraseExisting deletes all contents of the store beneath root.
func (s *Store) EraseExisting(ctx context.Context, root string) error {
	if err := s.fs.DeleteAll(ctx, root); err != nil {
		return fmt.Errorf("failed to delete store contents at %v: %v", root, err)
	}
	return nil
}

func (s *Store) FS() FS {
	return s.fs
}

// Read retrieves the object type and serialized data at the specified prefix and name
// from the store. The caller is responsible for using the returned type to
// decode the data into an appropriate object.
func (s *Store) Read(ctx context.Context, prefix, name string) (Type, []byte, error) {
	path := s.fs.Join(prefix, name)
	buf, err := s.fs.Get(ctx, path)
	if err != nil {
		return "", nil, err
	}
	rd := bytes.NewReader(buf)
	typ, err := readSlice(rd)
	if err != nil {
		return "", nil, err
	}
	atomic.AddInt64(&s.read, 1)
	return Type(typ), buf, nil
}

// Write stores the data at the specified prefix and name in the store.
func (s *Store) Write(ctx context.Context, prefix, name string, data []byte) error {
	path := s.fs.Join(prefix, name)
	if err := s.fs.Put(ctx, path, 0600, data); err != nil {
		if !s.fs.IsNotExist(err) {
			return err
		}
		if err := s.fs.EnsurePrefix(ctx, prefix, 0700); err != nil {
			return err
		}
		if err := s.fs.Put(ctx, path, 0600, data); err != nil {
			return err
		}
	}
	atomic.AddInt64(&s.written, 1)
	return nil
}

// Stats returns the number of objects read and written to the store
// since this instance was created.
func (s *Store) Stats() (read, written int64) {
	return atomic.LoadInt64(&s.read), atomic.LoadInt64(&s.written)
}

// Store serializes the object and writes the resulting data the supplied store
// at the specified prefix and name.
func (o *Object[V, R]) Store(ctx context.Context, s ObjectStore, prefix, name string, valueEncoding, responseEncoding ObjectEncoding) error {
	buf, err := o.Encode(valueEncoding, responseEncoding)
	if err != nil {
		return err
	}
	return s.Write(ctx, prefix, name, buf)
}

// Load reads the serialized data from the supplied store at the specified
// prefix and name and deserializes it into the object. The caller is responsible
// for ensuring that the type of the stored object and the type of the object
// are identical.
func (o *Object[V, R]) Load(ctx context.Context, s Store, prefix, name string) (Type, error) {
	ctype, buf, err := s.Read(ctx, prefix, name)
	if err != nil {
		return "", err
	}
	return ctype, o.Decode(buf)
}
