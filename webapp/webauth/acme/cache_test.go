// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package acme_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"cloudeng.io/os/lockedfile"
	"cloudeng.io/webapp/webauth/acme"
	"github.com/stretchr/testify/require"
)

type mockCacheFS struct {
	mu    sync.Mutex
	store map[string][]byte
	err   error // to inject errors
}

func newMockCacheFS() *mockCacheFS {
	return &mockCacheFS{
		store: make(map[string][]byte),
	}
}

func (m *mockCacheFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	data, ok := m.store[name]
	if !ok {
		return nil, acme.ErrCacheMiss
	}
	return data, nil
}

func (m *mockCacheFS) WriteFileCtx(_ context.Context, name string, data []byte, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.store[name] = data
	return nil
}

func (m *mockCacheFS) Delete(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	delete(m.store, name)
	return nil
}

func TestIsLocalName(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		local bool
	}{
		{"example.com", false},
		{"foo.bar.org", false},
		{"example.com+token", true},
		{"example.com+rsa", true},
		{"acme_account+key", true},
		{"acme_account.key", true},
		{"something/http-01/foo", true},
	}
	for _, tc := range testCases {
		if got, want := acme.IsLocalName(tc.name), tc.local; got != want {
			t.Errorf("IsLocalName(%q): got %v, want %v", tc.name, got, want)
		}
	}
}

func setupCache(t *testing.T, readonly bool) (*acme.CachingStore, *mockCacheFS, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	mockFS := newMockCacheFS()
	cache := acme.NewCachingStore(tmpDir, mockFS, readonly)
	return cache, mockFS, func() {}
}

func TestCacheReadonly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache, _, cleanup := setupCache(t, true)
	defer cleanup()

	// Put should fail.
	err := cache.Put(ctx, "remote.com", []byte("cert"))
	if !errors.Is(err, acme.ErrReadonlyCache) {
		t.Errorf("got %v, want %v", err, acme.ErrReadonlyCache)
	}
	err = cache.Put(ctx, "local.key", []byte("key"))
	if !errors.Is(err, acme.ErrReadonlyCache) {
		t.Errorf("got %v, want %v", err, acme.ErrReadonlyCache)
	}

	// Delete should fail.
	err = cache.Delete(ctx, "remote.com")
	if !errors.Is(err, acme.ErrReadonlyCache) {
		t.Errorf("got %v, want %v", err, acme.ErrReadonlyCache)
	}
	err = cache.Delete(ctx, "local.key")
	if !errors.Is(err, acme.ErrReadonlyCache) {
		t.Errorf("got %v, want %v", err, acme.ErrReadonlyCache)
	}

	// Get should return miss.
	_, err = cache.Get(ctx, "remote.com")
	if !errors.Is(err, acme.ErrCacheMiss) {
		t.Errorf("got %v, want %v", err, acme.ErrCacheMiss)
	}
	_, err = cache.Get(ctx, "local.key")
	if !errors.Is(err, acme.ErrCacheMiss) {
		t.Errorf("got %v, want %v", err, acme.ErrCacheMiss)
	}
}

func TestCacheReadWrite(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache, mockFS, cleanup := setupCache(t, false)
	defer cleanup()

	remoteName, remoteData := "remote.com", []byte("cert-data")
	localName, localData := "acme_account.key", []byte("key-data")

	// Test Put
	if err := cache.Put(ctx, remoteName, remoteData); err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(ctx, localName, localData); err != nil {
		t.Fatal(err)
	}

	// Test Get
	// Remote
	gotData, err := cache.Get(ctx, remoteName)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotData, remoteData) {
		t.Errorf("got %q, want %q", gotData, remoteData)
	}
	if got, want := mockFS.store[remoteName], remoteData; !reflect.DeepEqual(got, want) {
		t.Errorf("backing store: got %q, want %q", got, want)
	}

	// Local
	gotData, err = cache.Get(ctx, localName)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotData, localData) {
		t.Errorf("got %q, want %q", gotData, localData)
	}

	// Test Delete
	// Remote
	if err := cache.Delete(ctx, remoteName); err != nil {
		t.Fatal(err)
	}
	_, err = cache.Get(ctx, remoteName)
	if !errors.Is(err, acme.ErrCacheMiss) {
		t.Errorf("got %v, want %v", err, acme.ErrCacheMiss)
	}
	if _, ok := mockFS.store[remoteName]; ok {
		t.Errorf("remote entry not deleted from backing store")
	}

	// Local
	if err := cache.Delete(ctx, localName); err != nil {
		t.Fatal(err)
	}
	_, err = cache.Get(ctx, localName)
	if !errors.Is(err, acme.ErrCacheMiss) {
		t.Errorf("got %v, want %v", err, acme.ErrCacheMiss)
	}
}

func TestCacheLocking(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tmpDir := t.TempDir()
	mockFS := newMockCacheFS()
	cache := acme.NewCachingStore(tmpDir, mockFS, false)

	localName, localData := "local.key+token", []byte("key")
	if err := cache.Put(ctx, localName, localData); err != nil {
		t.Fatal(err)
	}

	// Manually lock the file to simulate contention.
	m := lockedfile.MutexAt(filepath.Join(tmpDir, "dir.lock"))
	unlock, err := m.Lock()
	if err != nil {
		t.Fatal(err)
	}

	timeCh := make(chan time.Time, 3)
	errCh := make(chan error, 3)

	start := time.Now()

	go func() {
		err := cache.Put(ctx, localName, []byte("new-data"))
		errCh <- err
		timeCh <- time.Now()
		_, err = cache.Get(ctx, localName)
		errCh <- err
		timeCh <- time.Now()
		err = cache.Delete(ctx, localName)
		errCh <- err
		timeCh <- time.Now()
	}()

	time.Sleep(time.Second)
	unlock()
	stopped := time.Now().Add(time.Millisecond * 100)
	for range 3 {
		err := <-errCh
		if err != nil {
			t.Errorf("got %v, want %v", err, context.DeadlineExceeded)
		}
		done := <-timeCh
		if done.Before(start) || done.After(stopped) {
			t.Errorf("operation did not wait for lock release")
		}
	}

	if err := cache.Put(ctx, localName, localData); err != nil {
		t.Fatal(err)
	}

	// Remove the lock file to simulate lock acquisition failure.
	err = os.Remove(filepath.Join(tmpDir, "dir.lock"))
	require.NoError(t, err)
	err = os.Chmod(tmpDir, 0000)
	require.NoError(t, err)

	err = cache.Put(ctx, localName, []byte("new-data"))
	require.ErrorIs(t, err, acme.ErrLockFailed)

	_, err = cache.Get(ctx, localName)
	require.ErrorIs(t, err, acme.ErrLockFailed)

	err = cache.Delete(ctx, localName)
	require.ErrorIs(t, err, acme.ErrLockFailed)

	err = os.Chmod(tmpDir, 0700)
	require.NoError(t, err)
}

func TestNullCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cache := acme.NewNullCache()

	if err := cache.Put(ctx, "any", []byte("any")); err != nil {
		t.Errorf("Put failed: %v", err)
	}

	if _, err := cache.Get(ctx, "any"); !errors.Is(err, acme.ErrCacheMiss) {
		t.Errorf("Get: got %v, want %v", err, acme.ErrCacheMiss)
	}

	if err := cache.Delete(ctx, "any"); err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}
