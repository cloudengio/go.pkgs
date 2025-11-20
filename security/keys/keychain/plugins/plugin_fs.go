// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

type FS struct {
	sysSpecific any
	binary      string
	args        []string
}

func NewFS(pluginPath string, sysSpecific any, args ...string) (*FS, error) {
	return &FS{
		sysSpecific: sysSpecific,
		binary:      pluginPath,
		args:        args,
	}, nil
}

func (f *FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
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
		return nil, fmt.Errorf("plugin error: %s", resp.Error)
	}
	return base64.StdEncoding.DecodeString(resp.Contents)
}

func (f *FS) WriteFileCtx(ctx context.Context, name string, data []byte) error {
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
