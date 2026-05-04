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
	mu    sync.Mutex
	data  map[string][]byte
	hits  map[string]int
	delay time.Duration
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

func (m *mockFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	m.mu.Lock()
	m.hits[name]++
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.delay):
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
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

	// Poll until the cached entry expires and the backing FS is hit again.
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := c.ReadFileCtx(ctx, "ttlfile"); err != nil {
			t.Fatalf("ReadFileCtx failed: %v", err)
		}
		if got := fs.getHits("ttlfile"); got == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for TTL expiry; got %d hits, want 2", fs.getHits("ttlfile"))
		}
		time.Sleep(5 * time.Millisecond)
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

func TestCacheConcurrency(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["concurrency_file"] = []byte("concurrent_data")
	// Add an artificial delay to ensure goroutines overlap and trigger singleflight
	fs.delay = 50 * time.Millisecond

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0), WithSingleFlight(true))
	t.Cleanup(func() { _ = c.Close() })

	var wg sync.WaitGroup
	numRoutines := 50
	for range numRoutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := c.ReadFileCtx(ctx, "concurrency_file")
			if err != nil {
				t.Errorf("ReadFileCtx failed: %v", err)
			}
			if string(data) != "concurrent_data" {
				t.Errorf("got %q, want %q", data, "concurrent_data")
			}
		}()
	}
	wg.Wait()

	// Due to singleflight, the underlying FS should only be hit exactly once
	if got, want := fs.getHits("concurrency_file"), 1; got != want {
		t.Errorf("expected %d hits due to singleflight, got %d", want, got)
	}
}

func TestCacheCloseIdempotency(t *testing.T) {
	fs := newMockFS()
	c := NewCachingReadFileFS(fs, WithCleanupInterval(10*time.Millisecond))

	// Sequential closes after concurrent ones
	for i := range 3 {
		if err := c.Close(); err != nil {
			t.Errorf("Sequential Close %d failed: %v", i, err)
		}
	}
}

func TestCacheSingleflightRetry_Timeout(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["file_retry"] = []byte("retry_data")

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0), WithSingleFlight(true))
	t.Cleanup(func() { _ = c.Close() })

	fs.delay = 100 * time.Millisecond

	ctx1, cancel1 := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel1()

	ctx2, cancel2 := context.WithTimeout(ctx, 1*time.Second)
	defer cancel2()

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error
	var data2 []byte

	go func() {
		defer wg.Done()
		_, err1 = c.ReadFileCtx(ctx1, "file_retry")
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		data2, err2 = c.ReadFileCtx(ctx2, "file_retry")
	}()

	wg.Wait()

	if err1 == nil {
		t.Error("expected caller 1 to fail with timeout")
	}

	if err2 != nil {
		t.Errorf("expected caller 2 to succeed, got: %v", err2)
	}

	if string(data2) != "retry_data" {
		t.Errorf("got %q, want %q", data2, "retry_data")
	}

	if got, want := fs.getHits("file_retry"), 2; got != want {
		t.Errorf("expected %d hits (1 failed, 1 retry), got %d", want, got)
	}
}

func TestCacheSingleflightRetry_Canceled(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["file_retry_cancel"] = []byte("retry_data")

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0), WithSingleFlight(true))
	t.Cleanup(func() { _ = c.Close() })

	fs.delay = 100 * time.Millisecond

	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithTimeout(ctx, 1*time.Second)
	defer cancel2()

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error
	var data2 []byte

	go func() {
		defer wg.Done()
		_, err1 = c.ReadFileCtx(ctx1, "file_retry_cancel")
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		data2, err2 = c.ReadFileCtx(ctx2, "file_retry_cancel")
	}()

	time.Sleep(10 * time.Millisecond)
	cancel1()

	wg.Wait()

	if err1 == nil {
		t.Error("expected caller 1 to fail with canceled")
	}

	if err2 != nil {
		t.Errorf("expected caller 2 to succeed, got: %v", err2)
	}

	if string(data2) != "retry_data" {
		t.Errorf("got %q, want %q", data2, "retry_data")
	}

	if got, want := fs.getHits("file_retry_cancel"), 2; got != want {
		t.Errorf("expected %d hits (1 failed, 1 retry), got %d", want, got)
	}
}

func TestCacheSingleflightRetry_BothCanceled(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["file_both_cancel"] = []byte("retry_data")

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0), WithSingleFlight(true))
	t.Cleanup(func() { _ = c.Close() })

	fs.delay = 100 * time.Millisecond

	ctx1, cancel1 := context.WithCancel(ctx)
	ctx2, cancel2 := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error

	go func() {
		defer wg.Done()
		_, err1 = c.ReadFileCtx(ctx1, "file_both_cancel")
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		_, err2 = c.ReadFileCtx(ctx2, "file_both_cancel")
	}()

	time.Sleep(10 * time.Millisecond)
	cancel1()
	cancel2()

	wg.Wait()

	if err1 == nil {
		t.Error("expected caller 1 to fail with canceled")
	}

	if err2 == nil {
		t.Error("expected caller 2 to fail with canceled")
	}
}

func TestCacheSingleflight_OtherError(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()

	c := NewCachingReadFileFS(fs, WithCleanupInterval(0), WithSingleFlight(true))
	t.Cleanup(func() { _ = c.Close() })

	fs.delay = 100 * time.Millisecond

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error

	go func() {
		defer wg.Done()
		_, err1 = c.ReadFileCtx(ctx, "file_not_found")
	}()

	time.Sleep(10 * time.Millisecond)

	go func() {
		defer wg.Done()
		_, err2 = c.ReadFileCtx(ctx, "file_not_found")
	}()

	wg.Wait()

	if err1 == nil || err2 == nil {
		t.Error("expected both callers to fail")
	}
}

func TestSingleFlightReadFileFS(t *testing.T) {
	ctx := context.Background()
	fs := newMockFS()
	fs.data["concurrency_file"] = []byte("concurrent_data")
	// Add an artificial delay to ensure goroutines overlap and trigger singleflight
	fs.delay = 50 * time.Millisecond

	sfFS := NewSingleFlightReadFileFS(fs)

	var wg sync.WaitGroup
	numRoutines := 50
	for range numRoutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := sfFS.ReadFileCtx(ctx, "concurrency_file")
			if err != nil {
				t.Errorf("ReadFileCtx failed: %v", err)
			}
			if string(data) != "concurrent_data" {
				t.Errorf("got %q, want %q", data, "concurrent_data")
			}
		}()
	}
	wg.Wait()

	// Due to singleflight, the underlying FS should only be hit exactly once
	if got, want := fs.getHits("concurrency_file"), 1; got != want {
		t.Errorf("expected %d hits due to singleflight, got %d", want, got)
	}
}
