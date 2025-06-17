// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalDownloadCachePutErrors(t *testing.T) {
	ctx := context.Background() // Use a background context for testing
	const contentSize int64 = 128
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := NewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}

	cache, err := NewLocalDownloadCache(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	// Do not defer close yet, we might close it manually for some tests

	validData := make([]byte, blockSize)

	tests := []struct {
		name       string
		r          ByteRange
		data       []byte
		wantErrMsg string
		preTestOp  func(c *LocalDownloadCache) // Operations before Put
		postTestOp func(c *LocalDownloadCache) // Cleanup/restore after Put
	}{
		{"invalid range From < 0", ByteRange{-10, 54}, validData, "invalid range", nil, nil},
		{"invalid range To > contentSize", ByteRange{0, contentSize + 1}, validData, "invalid range", nil, nil},
		{"invalid range From >= To", ByteRange{64, 64}, validData, "invalid range", nil, nil},
		{"data length mismatch", ByteRange{0, 64}, make([]byte, 32), "data length 32 does not match range size 64", nil, nil},
		{
			name:       "seek fails (closed file)",
			r:          ByteRange{0, 64},
			data:       validData,
			wantErrMsg: "failed to seek",
			preTestOp: func(c *LocalDownloadCache) {
				c.cache.Close() // Close the underlying file
			},
			postTestOp: func(*LocalDownloadCache) { // Re-open for subsequent tests if necessary (or test in isolation)
				// This test effectively makes the cache unusable afterwards.
				// For a real scenario, one might need to re-initialize.
			},
		},
		// Write fails test is similar to seek fails if file is closed.
		// If seek succeeds but write fails (e.g. disk full), that's harder to unit test without OS-level mocks.
		{
			name:       "saveRanges fails (index unwritable)",
			r:          ByteRange{0, 64},
			data:       validData,
			wantErrMsg: "failed to write index file",
			preTestOp: func(c *LocalDownloadCache) {
				// Re-open cache file if closed by previous test (if tests are not isolated)
				// For this test, ensure cache file is open, but index is bad.
				var openErr error
				c.cache, openErr = os.OpenFile(cacheFilePath, os.O_RDWR, 0644)
				if openErr != nil {
					t.Fatalf("Failed to re-open cache file for test: %v", openErr)
				}
				os.Remove(c.indexName)                              // Remove if it's a file
				if err := os.Mkdir(c.indexName, 0755); err != nil { // Make it a directory
					t.Fatalf("Failed to make indexName a directory: %v", err)
				}
			},
			postTestOp: func(c *LocalDownloadCache) {
				os.RemoveAll(c.indexName) // Clean up the directory
				c.cache.Close()           // Close the cache file opened in preTestOp
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-initialize or manage cache state carefully if tests are not fully isolated
			// For simplicity, some tests might leave the cache in a specific state.
			// The `postTestOp` for "seek fails" doesn't restore the cache.
			// If a test needs a fresh cache, it should re-initialize.
			// For "saveRanges fails", we re-open/close cache file.

			if tt.preTestOp != nil {
				tt.preTestOp(cache)
			}

			err := cache.Put(tt.r, tt.data)

			if tt.wantErrMsg == "" && err != nil {
				t.Errorf("Put() error = %v, wantErr nil", err)
			}
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("Put() error = nil, wantErrMsg %q", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Put() error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
			}
			if tt.postTestOp != nil {
				tt.postTestOp(cache)
			}
		})
	}
	cache.cache.Close() // Close at the end of the parent test function
}

func TestLocalDownloadCacheGetErrors(t *testing.T) {
	ctx := context.Background() // Use a background context for testing
	const contentSize int64 = 128
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := NewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	// Create a cache and put some data for valid read attempts
	cache, err := NewLocalDownloadCache(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	// Put data in the first block
	err = cache.Put(ByteRange{0, 64}, make([]byte, 64))
	if err != nil {
		t.Fatalf("Setup Put failed: %v", err)
	}
	// Do not defer close, will manage manually for error tests

	validBuffer := make([]byte, blockSize)

	tests := []struct {
		name       string
		r          ByteRange
		buffer     []byte
		wantErrMsg string
		preTestOp  func(c *LocalDownloadCache)
		postTestOp func(c *LocalDownloadCache)
	}{
		{"invalid range From < 0", ByteRange{-10, 54}, validBuffer, "invalid range", nil, nil},
		{"invalid range To > contentSize", ByteRange{0, contentSize + 1}, validBuffer, "invalid range", nil, nil},
		{"invalid range From >= To", ByteRange{64, 64}, validBuffer, "invalid range", nil, nil},
		{"buffer length mismatch", ByteRange{0, 64}, make([]byte, 32), "data length 32 does not match range size 64", nil, nil},
		{
			name:       "seek fails (closed file)",
			r:          ByteRange{0, 64},
			buffer:     validBuffer,
			wantErrMsg: "failed to seek",
			preTestOp: func(c *LocalDownloadCache) {
				c.cache.Close()
			},
			postTestOp: func(*LocalDownloadCache) {
				// Cache is unusable after this
			},
		},
		// Read fails is similar if file is closed after seek.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preTestOp != nil {
				tt.preTestOp(cache)
			}

			err := cache.Get(tt.r, tt.buffer)

			if tt.wantErrMsg == "" && err != nil {
				t.Errorf("Get() error = %v, wantErr nil", err)
			}
			if tt.wantErrMsg != "" {
				if err == nil {
					t.Errorf("Get() error = nil, wantErrMsg %q", tt.wantErrMsg)
				} else if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Get() error = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
			}
			if tt.postTestOp != nil {
				tt.postTestOp(cache)
			}
		})
	}
	cache.cache.Close()
}
