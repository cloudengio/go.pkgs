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

// Store represents a store for objects. It uses an instance of content.FS
// to store and retrieve objects. The objects are stored in a hierarchy
// at the specified root prefix/path and all operations are relative to
// that root. It is intended to be backed by either a local or cloud
// filesystem.
type Store[Value, Response any] struct {
	fs                              FS
	valueEncoding, responseEncoding ObjectEncoding
	root                            string
	written                         int64
	read                            int64
}

type FS interface {
	file.ObjectFS
	file.FS
}

// NewStore returns a new instance of Store backed by the supplied
// content.FS and storing the specified objects encoded using the
// specified encodings.
func NewStore[V, R any](fs FS, path string, valueEncoding, responseEncoding ObjectEncoding) *Store[V, R] {
	return &Store[V, R]{
		fs:               fs,
		root:             path,
		valueEncoding:    valueEncoding,
		responseEncoding: responseEncoding,
	}
}

// EraseExisting deletes all existing contents of the store,
// ie. all objects beneath the root prefix.
func (s *Store[V, R]) EraseExisting(ctx context.Context) error {
	if err := s.fs.DeleteAll(ctx, s.root); err != nil {
		return fmt.Errorf("failed to delete store contents at %v: %v", s.root, err)
	}
	return nil
}

func (s *Store[V, R]) Store(ctx context.Context, prefix, name string, obj Object[V, R]) error {
	buf, err := obj.Encode(s.valueEncoding, s.responseEncoding)
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

func (s *Store[V, R]) Progress() (written, read int64) {
	return atomic.LoadInt64(&s.read), atomic.LoadInt64(&s.written)
}

func (s *Store[V, R]) Load(ctx context.Context, prefix, name string) (Type, Object[V, R], error) {
	var obj Object[V, R]
	path := s.fs.Join(s.root, prefix, name)
	buf, err := s.fs.Get(ctx, path)
	if err != nil {
		return "", obj, err
	}
	rd := bytes.NewReader(buf)
	typ, err := readSlice(rd)
	if err != nil {
		return "", obj, err
	}

	if err := obj.Decode(buf); err != nil {
		return Type(typ), obj, err
	}
	atomic.AddInt64(&s.read, 1)
	return Type(typ), obj, nil
}
