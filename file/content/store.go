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

// Store represents a store for objects. It uses an instance of content.FS
// to store and retrieve objects. The objects are stored in a hierarchy
// at the specified root prefix/path and all operations are relative to
// that root. It is intended to be backed by either a local or cloud
// filesystem.
type Store struct {
	fs      FS
	root    string
	written int64
	read    int64
}

// NewStore returns a new instance of Store backed by the supplied
// content.FS and storing the specified objects encoded using the
// specified encodings.
func NewStore(fs FS, path string) *Store {
	return &Store{
		fs:   fs,
		root: path,
	}
}

// EraseExisting deletes all existing contents of the store,
// ie. all objects beneath the root prefix.
func (s *Store) EraseExisting(ctx context.Context) error {
	if err := s.fs.DeleteAll(ctx, s.root); err != nil {
		return fmt.Errorf("failed to delete store contents at %v: %v", s.root, err)
	}
	return nil
}

func (s *Store) FS() FS {
	return s.fs
}

func (s *Store) Root() string {
	return s.root
}

func (o *Object[V, R]) Store(ctx context.Context, s *Store, prefix, name string, valueEncoding, responseEncoding ObjectEncoding) error {
	buf, err := o.Encode(valueEncoding, responseEncoding)
	if err != nil {
		return err
	}
	prefix = s.fs.Join(s.root, prefix)
	path := s.fs.Join(prefix, name)
	if err := s.fs.Put(ctx, path, 0600, buf); err != nil {
		if !s.fs.IsNotExist(err) {
			return err
		}
		if err := s.fs.EnsurePrefix(ctx, prefix, 0700); err != nil {
			return err
		}
		if err := s.fs.Put(ctx, path, 0600, buf); err != nil {
			return err
		}
	}
	atomic.AddInt64(&s.written, 1)
	return nil
}

func (o *Object[V, R]) Load(ctx context.Context, s *Store, prefix, name string) (Type, error) {
	ctype, buf, err := s.Read(ctx, prefix, name)
	if err != nil {
		return "", err
	}
	return ctype, o.Decode(buf)
}

func (s *Store) Read(ctx context.Context, prefix, name string) (Type, []byte, error) {
	name = s.fs.Join(s.root, prefix, name)
	buf, err := s.fs.Get(ctx, name)
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

// Stats returns the number of objects read and written to the store
// since this instance was created.
func (s *Store) Stats() (read, written int64) {
	return atomic.LoadInt64(&s.read), atomic.LoadInt64(&s.written)
}
