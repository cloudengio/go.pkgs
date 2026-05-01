// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cachefs

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type mockInvalidateFS struct {
	mu   sync.Mutex
	data map[string][]byte
	hits map[string]int
}

func newMockInvalidateFS() *mockInvalidateFS {
	return &mockInvalidateFS{
		data: make(map[string][]byte),
		hits: make(map[string]int),
	}
}

func (m *mockInvalidateFS) ReadFile(name string) ([]byte, error) {
	return m.ReadFileCtx(context.Background(), name)
}

func (m *mockInvalidateFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits[name]++
	if d, ok := m.data[name]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found: %s", name)
}

func (m *mockInvalidateFS) getHits(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits[name]
}

func TestCacheInvalidate(t *testing.T) {
	ctx := context.Background()
	fs := newMockInvalidateFS()
	fs.data["file1"] = []byte("content1")

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0))
	t.Cleanup(func() { _ = c.Close() })

	// First read, should fetch from FS
	if _, err := c.ReadFileCtx(ctx, "file1"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := fs.getHits("file1"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}

	// Second read, should hit cache
	if _, err := c.ReadFileCtx(ctx, "file1"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := fs.getHits("file1"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}

	// Invalidate the cache entry
	c.Invalidate("file1")

	// Third read, should fetch from FS again
	if _, err := c.ReadFileCtx(ctx, "file1"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := fs.getHits("file1"), 2; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}
}
