// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// FS implements a plugin-based file system for keychain
// that implements file.ReadFileFS and file.WriteFileFS.
type FS struct {
	sysSpecific any
	binary      string
	args        []string
}

// NewFS creates a new FS instance with the specified plugin path, system-specific data, and plugin arguments.
func NewFS(pluginPath string, sysSpecific any, args ...string) *FS {
	return &FS{
		sysSpecific: sysSpecific,
		binary:      pluginPath,
		args:        args,
	}
}

func (f FS) ReadFile(name string) ([]byte, error) {
	return f.ReadFileCtx(context.Background(), name)
}

func (f FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	req, err := NewRequest(name, f.sysSpecific)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := RunExtPlugin(ctx, f.binary, req, f.args...)
	if err != nil {
		return nil, fmt.Errorf("failed to run plugin: %w", err)
	}
	if resp.Error != nil {
		if errors.Is(resp.Error, ErrKeyNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("plugin error: %w", resp.Error)
	}
	return DecodeContents(resp.Contents)
}

func (f FS) WriteFile(name string, data []byte) error {
	return f.WriteFileCtx(context.Background(), name, data)
}

func (f FS) WriteFileCtx(ctx context.Context, name string, data []byte) error {
	req, err := NewWriteRequest(name, data, f.sysSpecific)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := RunExtPlugin(ctx, f.binary, req, f.args...)
	if err != nil {
		return fmt.Errorf("failed to run plugin: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("plugin error: %s", resp.Error)
	}
	return nil
}
