// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
)

// FS implements a plugin-based file system for keychain
// that implements file.ReadFileFS and file.WriteFileFS.
type FS struct {
	pluginSpecific any
	binary         string
	args           []string
	logger         *slog.Logger
}

// NewFS creates a new FS instance with the specified plugin path, plugin-specific
// data, and plugin arguments. The plugin-specific data is passed to the plugin
// in the request.
func NewFS(pluginPath string, pluginSpecific any, args ...string) *FS {
	return &FS{
		pluginSpecific: pluginSpecific,
		binary:         pluginPath,
		args:           args,
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// WithLogger returns a new FS instance with the provided logger.
func (f *FS) WithLogger(logger *slog.Logger) *FS {
	f.logger = logger.With("plugin", f.binary)
	return f
}

func (f FS) ReadFile(name string) ([]byte, error) {
	return f.ReadFileCtx(context.Background(), name)
}

func (f FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	req, err := NewRequest(name, f.pluginSpecific)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := RunExtPlugin(ctx, f.binary, req, f.args...)
	f.logger.Info("plugin read file", "name", name, "stderr", resp.Stderr, "error", err)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		if errors.Is(resp.Error, ErrKeyNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("plugin error: %w", resp.Error)
	}
	return resp.Contents, nil
}

func (f FS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return f.WriteFileCtx(context.Background(), name, data, perm)
}

func (f FS) WriteFileCtx(ctx context.Context, name string, data []byte, _ fs.FileMode) error {
	req, err := NewWriteRequest(name, data, f.pluginSpecific)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := RunExtPlugin(ctx, f.binary, req, f.args...)
	f.logger.Info("plugin write file", "name", name, "stderr", resp.Stderr, "error", err)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("error reported by plugin: %w", resp.Error)
	}
	return nil
}
