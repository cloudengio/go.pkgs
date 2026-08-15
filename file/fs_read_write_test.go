// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file_test

import (
	"context"
	"io/fs"
	"testing"

	"cloudeng.io/file"
)

type mockReadWriteFS struct {
	data map[string][]byte
}

func (m *mockReadWriteFS) ReadFile(name string) ([]byte, error) {
	if d, ok := m.data[name]; ok {
		return d, nil
	}
	return nil, fs.ErrNotExist
}

func (m *mockReadWriteFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	return m.ReadFile(name)
}

func (m *mockReadWriteFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.data[name] = append([]byte(nil), data...)
	return nil
}

func (m *mockReadWriteFS) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error {
	return m.WriteFile(name, data, perm)
}

func TestReadWriteFSContext(t *testing.T) {
	ctx := context.Background()

	// Initial check on empty context
	if _, ok := file.ReadWriteFSFromContext(ctx); ok {
		t.Errorf("expected no ReadWriteFS in empty context")
	}
	if _, ok := file.ReadWriteFileFSFromContext(ctx); ok {
		t.Errorf("expected no ReadWriteFileFS in empty context")
	}

	mockFS := &mockReadWriteFS{data: make(map[string][]byte)}

	// Store in context
	ctxWithFS := file.ContextWithReadWriteFS(ctx, mockFS)

	retrieved, ok := file.ReadWriteFSFromContext(ctxWithFS)
	if !ok || retrieved != mockFS {
		t.Errorf("ReadWriteFSFromContext = %v, %v, want %v, true", retrieved, ok, mockFS)
	}

	retrieved2, ok := file.ReadWriteFileFSFromContext(ctxWithFS)
	if !ok || retrieved2 != mockFS {
		t.Errorf("ReadWriteFileFSFromContext = %v, %v, want %v, true", retrieved2, ok, mockFS)
	}

	// Store via alias
	ctxWithFS2 := file.ContextWithReadWriteFileFS(ctx, mockFS)
	if r, ok := file.ReadWriteFSFromContext(ctxWithFS2); !ok || r != mockFS {
		t.Errorf("ContextWithReadWriteFileFS failed")
	}

	// Remove from context
	ctxWithoutFS := file.ContextWithoutReadWriteFS(ctxWithFS)
	if _, ok := file.ReadWriteFSFromContext(ctxWithoutFS); ok {
		t.Errorf("expected no ReadWriteFS after ContextWithoutReadWriteFS")
	}
}
