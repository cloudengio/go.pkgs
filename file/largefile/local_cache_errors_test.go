// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.
package largefile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDownloadSmallerThanBlockSizeFile(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 32
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	n, err := cache.WriteAt(make([]byte, contentSize), 0)
	if err != nil {
		t.Fatalf("WriteAt failed: %v", err)
	}
	if got, want := int64(n), contentSize; got != want {
		t.Errorf("WriteAt() = %v, want %v", got, want)
	}
	// Verify that the file is cached
	if !cache.Complete() {
		t.Errorf("Expected cache to be complete, but it is not")
	}
	// Verify that the content length and block size are correct
	gotContentLength, gotBlockSize := cache.ContentLengthAndBlockSize()
	if gotContentLength != contentSize || gotBlockSize != blockSize {
		t.Errorf("ContentLengthAndBlockSize() = (%v, %v), want (%v, %v)",
			gotContentLength, gotBlockSize, contentSize, blockSize)
	}
	// Verify that the cached bytes and blocks are correct
	bytes, blocks := cache.CachedBytesAndBlocks()
	if bytes != contentSize || blocks != 1 {
		t.Errorf("CachedBytesAndBlocks() = (%v, %v), want (%v, %v)",
			bytes, blocks, contentSize, 1)
	}

	if _, err := cache.WriteAt(make([]byte, contentSize-1), 0); !errors.Is(err, ErrCacheInvalidBlockSize) {
		t.Errorf("WriteAt with invalid block size did not return expected error: %v", err)
	}

	if _, err := cache.WriteAt(make([]byte, contentSize), contentSize); !errors.Is(err, ErrCacheInvalidOffset) {
		t.Errorf("WriteAt with invalid offset did not return expected error: %v", err)
	}

}

func TestLocalDownloadExactBlockSizeFile(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 128
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}

	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	n, err := cache.WriteAt(make([]byte, blockSize), 0)
	if err != nil {
		t.Fatalf("WriteAt failed: %v", err)
	}
	if got, want := n, blockSize; got != want {
		t.Errorf("WriteAt() = %v, want %v", got, want)
	}

	n, err = cache.WriteAt(make([]byte, blockSize), int64(blockSize))
	if err != nil {
		t.Fatalf("WriteAt failed: %v", err)
	}
	if got, want := n, blockSize; got != want {
		t.Errorf("WriteAt() = %v, want %v", got, want)
	}
	// Verify that the file is cached
	if !cache.Complete() {
		t.Errorf("Expected cache to be complete, but it is not")
	}
	// Verify that the content length and block size are correct
	gotContentLength, gotBlockSize := cache.ContentLengthAndBlockSize()
	if gotContentLength != contentSize || gotBlockSize != blockSize {
		t.Errorf("ContentLengthAndBlockSize() = (%v, %v), want (%v, %v)",
			gotContentLength, gotBlockSize, contentSize, blockSize)
	}
	// Verify that the cached bytes and blocks are correct
	bytes, blocks := cache.CachedBytesAndBlocks()
	if bytes != contentSize || blocks != 2 {
		t.Errorf("CachedBytesAndBlocks() = (%v, %v), want (%v, %v)",
			bytes, blocks, contentSize, 2)
	}

	if _, err := cache.WriteAt(make([]byte, contentSize-1), 0); !errors.Is(err, ErrCacheInvalidBlockSize) {
		t.Errorf("WriteAt with invalid block size did not return expected error: %v", err)
	}

	if _, err := cache.WriteAt(make([]byte, contentSize), contentSize); !errors.Is(err, ErrCacheInvalidOffset) {
		t.Errorf("WriteAt with invalid offset did not return expected error: %v", err)
	}

}

// GenAI: claude 2.7 wrote these tests.

func TestLocalDownloadCacheErrors(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	const contentSize int64 = 130
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}

	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("OpenCacheFiles failed: %v", err)
	}

	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	t.Run("WriteAt error cases", func(t *testing.T) {
		tests := []struct {
			name    string
			data    []byte
			offset  int64
			wantErr error
			setup   func(*testing.T, *LocalDownloadCache)
			cleanup func(*testing.T, *LocalDownloadCache)
		}{
			{
				name:    "negative offset",
				data:    make([]byte, blockSize),
				offset:  -1,
				wantErr: ErrCacheInvalidOffset,
			},
			{
				name:    "offset beyond content length",
				data:    make([]byte, blockSize),
				offset:  contentSize,
				wantErr: ErrCacheInvalidOffset,
			},
			{
				name:    "unaligned offset",
				data:    make([]byte, blockSize),
				offset:  1,
				wantErr: ErrCacheInvalidOffset,
			},
			{
				name:    "wrong block size for middle block",
				data:    make([]byte, blockSize-1),
				offset:  0,
				wantErr: ErrCacheInvalidBlockSize,
			},
			{
				name:    "wrong block size for last block",
				data:    make([]byte, blockSize),
				offset:  128,
				wantErr: ErrCacheInvalidBlockSize,
			},
			{
				name:    "correct last block size",
				data:    make([]byte, contentSize-int64(blockSize)),
				offset:  128,
				wantErr: nil,
			},
			{
				name:    "file closed",
				data:    make([]byte, blockSize),
				offset:  0,
				wantErr: os.ErrClosed,
				setup: func(t *testing.T, c *LocalDownloadCache) {
					if err := c.data.Close(); err != nil {
						t.Fatalf("Failed to close cache file: %v", err)
					}
				},
				cleanup: func(t *testing.T, c *LocalDownloadCache) {
					var err error
					c.data, err = os.OpenFile(cacheFilePath, os.O_RDWR, 0600)
					if err != nil {
						t.Fatalf("Failed to reopen cache file: %v", err)
					}
				},
			},
			{
				name:    "index file not writable",
				data:    make([]byte, blockSize),
				offset:  0,
				wantErr: nil, // This will cause an error but not necessarily return one of our sentinel errors
				setup: func(t *testing.T, c *LocalDownloadCache) {
					if err := os.Chmod(c.indexStore.wr.Name(), 0400); err != nil {
						t.Fatalf("Failed to make index read-only: %v", err)
					}
				},
				cleanup: func(t *testing.T, c *LocalDownloadCache) {
					if err := os.Chmod(c.indexStore.wr.Name(), 0600); err != nil {
						t.Fatalf("Failed to make index writable: %v", err)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.setup != nil {
					tt.setup(t, cache)
				}

				_, err := cache.WriteAt(tt.data, tt.offset)
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("WriteAt() error = %v, wantErr %v", err, tt.wantErr)
				}

				if tt.cleanup != nil {
					tt.cleanup(t, cache)
				}
			})
		}
	})

	t.Run("ReadAt error cases", func(t *testing.T) {
		// First write some valid data for testing reads
		validData := make([]byte, blockSize)
		for i := range validData {
			validData[i] = byte(i)
		}
		if _, err := cache.WriteAt(validData, 0); err != nil {
			t.Fatalf("Failed to write valid data: %v", err)
		}

		tests := []struct {
			name    string
			buffer  []byte
			offset  int64
			wantErr error
			setup   func(*testing.T, *LocalDownloadCache)
			cleanup func(*testing.T, *LocalDownloadCache)
		}{
			{
				name:    "negative offset",
				buffer:  make([]byte, blockSize),
				offset:  -1,
				wantErr: ErrCacheInvalidOffset,
			},
			{
				name:    "offset beyond content length",
				buffer:  make([]byte, blockSize),
				offset:  contentSize,
				wantErr: ErrCacheInvalidOffset,
			},
			{
				name:    "wrong buffer size for last block",
				buffer:  make([]byte, blockSize),
				offset:  128,
				wantErr: ErrCacheInvalidBlockSize,
			},
			{
				name:    "correct last block size",
				buffer:  make([]byte, contentSize-int64(blockSize)),
				offset:  128,
				wantErr: nil,
			},
			{
				name:    "reading uncached data",
				buffer:  make([]byte, contentSize-int64(blockSize)),
				offset:  64,
				wantErr: ErrCacheUncachedRange, // Expected error message about uncached data
			},
			{
				name:    "file closed",
				buffer:  make([]byte, blockSize),
				offset:  0,
				wantErr: os.ErrClosed,
				setup: func(t *testing.T, c *LocalDownloadCache) {
					if err := c.data.Close(); err != nil {
						t.Fatalf("Failed to close cache file: %v", err)
					}
				},
				cleanup: func(t *testing.T, c *LocalDownloadCache) {
					var err error
					c.data, err = os.OpenFile(cacheFilePath, os.O_RDWR, 0600)
					if err != nil {
						t.Fatalf("Failed to reopen cache file: %v", err)
					}
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.setup != nil {
					tt.setup(t, cache)
				}

				_, err := cache.ReadAt(tt.buffer, tt.offset)
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("ReadAt() error = %v, wantErr %v", err, tt.wantErr)
				}

				if tt.cleanup != nil {
					tt.cleanup(t, cache)
				}
			})
		}
	})

	t.Run("NewFilesForCache errors", func(t *testing.T) {
		tests := []struct {
			name        string
			filename    string
			indexFile   string
			contentSize int64
			blockSize   int
			wantErr     bool
		}{
			{
				name:        "empty filename",
				filename:    "",
				indexFile:   "index.idx",
				contentSize: 100,
				blockSize:   64,
				wantErr:     true,
			},
			{
				name:        "empty index filename",
				filename:    "cache.dat",
				indexFile:   "",
				contentSize: 100,
				blockSize:   64,
				wantErr:     true,
			},
			{
				name:        "directory for cache file",
				filename:    tmpDir,
				indexFile:   indexFilePath,
				contentSize: 100,
				blockSize:   64,
				wantErr:     true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := CreateNewFilesForCache(ctx, tt.filename, tt.indexFile, tt.contentSize, tt.blockSize, 1, nil)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewFilesForCache() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("OpenCacheFiles errors", func(t *testing.T) {
		tests := []struct {
			name      string
			filename  string
			indexFile string
			setup     func(*testing.T)
			wantErr   bool
		}{
			{
				name:      "non-existent cache file",
				filename:  filepath.Join(tmpDir, "nonexistent.dat"),
				indexFile: indexFilePath,
				wantErr:   true,
			},
			{
				name:      "non-existent index file",
				filename:  cacheFilePath,
				indexFile: filepath.Join(tmpDir, "nonexistent.idx"),
				wantErr:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.setup != nil {
					tt.setup(t)
				}
				_, _, err := OpenCacheFiles(tt.filename, tt.indexFile)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewLocalDownloadCache() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("NewLocalDownloadCache errors", func(t *testing.T) {
		tests := []struct {
			name      string
			filename  string
			indexFile string
			setup     func(*testing.T)
			wantErr   bool
		}{
			{
				name:      "corrupted index file",
				filename:  cacheFilePath,
				indexFile: filepath.Join(tmpDir, "corrupted.idx"),
				setup: func(t *testing.T) {
					if err := os.WriteFile(filepath.Join(tmpDir, "corrupted.idx"), []byte("not json"), 0600); err != nil {
						t.Fatalf("Failed to write corrupted index file: %v", err)
					}
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.setup != nil {
					tt.setup(t)
				}
				cacheFile, indexFile, err := OpenCacheFiles(tt.filename, tt.indexFile)
				if err != nil {
					t.Fatalf("Failed to open cache files: %v", err)
				}
				_, err = NewLocalDownloadCache(cacheFile, indexFile)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewLocalDownloadCache() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("ByteRanges functionality", func(t *testing.T) {
		err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
		if err != nil {
			t.Fatalf("NewFilesForCache failed: %v", err)
		}
		cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
		if err != nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}
		// Test that NextOutstanding and NextCached work correctly
		// First, clear all cached data
		freshCache, err := NewLocalDownloadCache(cacheFile, indexFile)
		if err != nil {
			t.Fatalf("Failed to create fresh cache: %v", err)
		}
		defer freshCache.Close()

		// Nothing should be cached initially
		var br ByteRange
		if idx := freshCache.NextCached(0, &br); idx != -1 {
			t.Errorf("Expected no cached blocks, got index %d with range %+v", idx, br)
		}

		// All blocks should be outstanding
		if idx := freshCache.NextOutstanding(0, &br); idx == -1 {
			t.Errorf("Expected outstanding blocks, got none")
		}

		// Write first block
		data := make([]byte, blockSize)
		if _, err := freshCache.WriteAt(data, 0); err != nil {
			t.Fatalf("Failed to write first block: %v", err)
		}

		// Now first block should be cached
		idx := freshCache.NextCached(0, &br)
		if idx == -1 || br.From != 0 || br.Size() != int64(blockSize) {
			t.Errorf("Expected first block cached, got index %d with range %+v", idx, br)
		}

		// Second block should be outstanding
		idx = freshCache.NextOutstanding(0, &br)
		if idx == -1 || br.From != int64(blockSize) {
			t.Errorf("Expected second block outstanding, got index %d with range %+v", idx, br)
		}

		// Complete() should return false
		if freshCache.Complete() {
			t.Errorf("Expected cache to be incomplete")
		}

		// Write second block
		data = make([]byte, blockSize)
		if _, err := freshCache.WriteAt(data, int64(blockSize)); err != nil {
			t.Fatalf("Failed to write second block: %v", err)
		}

		// Write third (last) block
		lastBlockSize := contentSize - int64(blockSize)*2
		data = make([]byte, lastBlockSize)
		if _, err := freshCache.WriteAt(data, int64(blockSize)*2); err != nil {
			t.Fatalf("Failed to write last block: %v", err)
		}

		// Now cache should be complete
		if !freshCache.Complete() {
			t.Errorf("Expected cache to be complete")
		}

		// NextOutstanding should return -1
		if idx := freshCache.NextOutstanding(0, &br); idx != -1 {
			t.Errorf("Expected no outstanding blocks, got index %d with range %+v", idx, br)
		}

		// Check cached bytes and blocks
		bytes, blocks := freshCache.CachedBytesAndBlocks()
		if bytes != contentSize || blocks != 3 {
			t.Errorf("Expected %d bytes in %d blocks, got %d bytes in %d blocks", contentSize, 3, bytes, blocks)
		}
	})
}

func TestContentLengthAndBlockSize(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 130
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}

	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}

	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	gotContentLength, gotBlockSize := cache.ContentLengthAndBlockSize()
	if gotContentLength != contentSize || gotBlockSize != blockSize {
		t.Errorf("ContentLengthAndBlockSize() = (%v, %v), want (%v, %v)",
			gotContentLength, gotBlockSize, contentSize, blockSize)
	}
}

func TestLocalDownloadCacheClose(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, 100, 64, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}

	// Test closing
	if err := cache.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Test that operations after close fail
	_, err = cache.WriteAt(make([]byte, 64), 0)
	if err == nil {
		t.Errorf("WriteAt after Close() succeeded, want error")
	}

}
