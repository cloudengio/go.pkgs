// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"cloudeng.io/security/keys/keychain/plugins"
)

// FS is an in-memory filesystem that reads and writes keys directly
// from an in-memory Plugin without running external processes.
// It implements file.ReadFileFS and file.WriteFileFS.
type FS struct {
	plugin   *Plugin
	writable bool
}

// NewFS creates a new in-memory FS instance backed by the given Plugin.
func NewFS(plugin *Plugin, writable bool) *FS {
	return &FS{
		plugin:   plugin,
		writable: writable,
	}
}

// Plugin returns the underlying Plugin.
func (f *FS) Plugin() *Plugin {
	return f.plugin
}

// ReadFile reads a key from the in-memory store.
func (f *FS) ReadFile(name string) ([]byte, error) {
	return f.ReadFileCtx(context.Background(), name)
}

// ReadFileCtx reads a key from the in-memory store with context.
func (f *FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	req := plugins.Request{
		ID:      plugins.NextID(),
		Keyname: name,
		Write:   false,
	}
	resp := f.plugin.HandleRequest(ctx, req)
	if resp.Error != nil {
		if errors.Is(resp.Error, plugins.ErrKeyNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, resp.Error
	}
	return resp.Contents, nil
}

// WriteFile writes a key to the in-memory store.
func (f *FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return f.WriteFileCtx(context.Background(), name, data, perm)
}

// WriteFileCtx writes a key to the in-memory store with context.
func (f *FS) WriteFileCtx(ctx context.Context, name string, data []byte, _ fs.FileMode) error {
	if !f.writable {
		return plugins.ErrReadOnly
	}
	req := plugins.Request{
		ID:       plugins.NextID(),
		Keyname:  name,
		Write:    true,
		Contents: data,
	}
	resp := f.plugin.HandleRequest(ctx, req)
	if resp.Error != nil {
		return resp.Error
	}
	return nil
}
