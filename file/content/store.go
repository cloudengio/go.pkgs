// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"context"

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
func (o *Object[V, R]) Load(ctx context.Context, s ObjectStore, prefix, name string) (Type, error) {
	ctype, buf, err := s.Read(ctx, prefix, name)
	if err != nil {
		return "", err
	}
	return ctype, o.Decode(buf)
}
