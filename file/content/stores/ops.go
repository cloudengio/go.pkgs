// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores

import (
	"bytes"
	"context"
	"fmt"

	"cloudeng.io/file/content"
	"cloudeng.io/file/content/internal"
)

func eraseExisting(ctx context.Context, fs content.FS, root string) error {
	if err := fs.DeleteAll(ctx, root); err != nil {
		return fmt.Errorf("failed to delete store contents at %v: %v", root, err)
	}
	return nil
}

func read(ctx context.Context, fs content.FS, path string) (content.Type, []byte, error) {
	buf, err := fs.Get(ctx, path)
	if err != nil {
		return "", nil, err
	}
	rd := bytes.NewReader(buf)
	typ, err := internal.ReadSlice(rd)
	if err != nil {
		return "", nil, err
	}
	return content.Type(typ), buf, nil
}

func write(ctx context.Context, fs content.FS, prefix, name string, data []byte) error {
	path := fs.Join(prefix, name)
	if err := fs.Put(ctx, path, 0600, data); err != nil {
		if !fs.IsNotExist(err) {
			return err
		}
		if err := fs.EnsurePrefix(ctx, prefix, 0700); err != nil {
			return err
		}
		if err := fs.Put(ctx, path, 0600, data); err != nil {
			return err
		}
	}
	return nil
}
