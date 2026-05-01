// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cachefs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockFS struct {
	mu   sync.Mutex
	data map[string][]byte
	hits map[string]int
}

func newMockFS() *mockFS {
	return &mockFS{
		data: make(map[string][]byte),
		hits: make(map[string]int),
	}
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	return m.ReadFileCtx(context.Background(), name)
}

func (m *mockFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits[name]++
	if d, ok := m.data[name]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found: %s", name)
}

func (m *mockFS) getHits(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hits[name]
}

func TestCacheBasic(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["file1"] = []byte("content1")
	fs.data["file2"] = []byte("content2")

	// Disable cleanup loop for this test
	c := NewCachingReadFileFS(fs, WithCleanupInterval(0))
	t.Cleanup(func() { _ = c.Close() })

	// Read file1 for the first time
	data, err := c.ReadFileCtx(ctx, "file1")
	if err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := string(data), "content1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := fs.getHits("file1"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}

	// Read file1 again, should hit cache
	data, err = c.ReadFile("file1")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if got, want := string(data), "content1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := fs.getHits("file1"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}

	// Read file2
	data, err = c.ReadFileCtx(ctx, "file2")
	if err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := string(data), "content2"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := fs.getHits("file2"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}
}

func TestCacheTTL(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["ttlfile"] = []byte("data")

	c := NewCachingReadFileFS(fs, WithTTL(20*time.Millisecond), WithCleanupInterval(0))
	t.Cleanup(func() { _ = c.Close() })

	// First read
	if _, err := c.ReadFileCtx(ctx, "ttlfile"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := fs.getHits("ttlfile"), 1; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}

	// Wait for TTL to expire
	time.Sleep(40 * time.Millisecond)

	// Second read, should fetch from FS again
	if _, err := c.ReadFileCtx(ctx, "ttlfile"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}
	if got, want := fs.getHits("ttlfile"), 2; got != want {
		t.Errorf("got %d hits, want %d", got, want)
	}
}

func TestCacheCleanup(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["cleanfile"] = []byte("data")

	c := NewCachingReadFileFS(fs, WithTTL(20*time.Millisecond), WithCleanupInterval(30*time.Millisecond))
	t.Cleanup(func() { _ = c.Close() })

	if _, err := c.ReadFileCtx(ctx, "cleanfile"); err != nil {
		t.Fatalf("ReadFileCtx failed: %v", err)
	}

	// Verify it's in the cache map
	c.mu.RLock()
	_, ok := c.cache["cleanfile"]
	c.mu.RUnlock()
	if !ok {
		t.Fatal("expected file to be in cache")
	}

	// Wait for TTL to expire AND cleanup loop to run
	time.Sleep(100 * time.Millisecond)

	// Verify it's removed from the cache map
	c.mu.RLock()
	_, ok = c.cache["cleanfile"]
	c.mu.RUnlock()
	if ok {
		t.Error("expected file to be cleaned up from cache")
	}
}

func TestCacheErrors(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0))
	t.Cleanup(func() { _ = c.Close() })

	if _, err := c.ReadFileCtx(ctx, "nonexistent"); err == nil {
		t.Fatal("expected error, got nil")
	}

	// Ensure the error wasn't cached
	c.mu.RLock()
	_, ok := c.cache["nonexistent"]
	c.mu.RUnlock()
	if ok {
		t.Error("expected failed read to not be cached")
	}
}
