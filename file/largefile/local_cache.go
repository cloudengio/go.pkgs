// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"sync"
)

// DownloadCache is an interface for caching byte ranges of large files
// to support resumable downloads.
type DownloadCache interface {
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)
	// Outstanding returns an iterator over the byte ranges that have not yet been cached.
	Outstanding() iter.Seq[ByteRange]
	// Cached returns an iterator over the byte ranges that have been cached.
	Cached() iter.Seq[ByteRange]
	// Complete returns true if all byte ranges have been cached.
	Complete() bool
	Put(r ByteRange, data []byte) error
	Get(r ByteRange, data []byte) error
}

// LocalDownloadCache is a concrete implementation of RangeCache that uses
// a local file to cache byte ranges of large files.
// It allows for concurrent access.
type LocalDownloadCache struct {
	indexName string     // Name of the index file for the cache.
	mu        sync.Mutex // Protects access to the cache file.
	cache     *os.File
	written   *ByteRanges // Ranges that have been written to the cache.

}

// NewFilesForCache creates a new cache file and an index file for caching
// byte ranges of large files. It reserves space for the cache file and
// initializes the index file with the specified content size and block size.
// It returns an error if the files cannot be created or if the space cannot
// be reserved.
// The index file is used to track which byte ranges have been written to the cache.
// The cache file is used to store the actual data.
// The contentSize is the total size of the file in bytes, blockSize is the preferred
// block size for downloading the file, and concurrency is the number of concurrent
// writes used to reserve space for the cache file on systems that require writing
// to the file to reserve space (e.g., non-Linux systems).
func NewFilesForCache(ctx context.Context, filename, indexFileName string, contentSize int64, blockSize, concurrency int) error {
	for _, fn := range []string{filename, indexFileName} {
		if len(fn) == 0 {
			return fmt.Errorf("filename cannot be empty")
		}
		if err := os.Remove(fn); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing file %s: %w", fn, err)
		}
	}

	// Reserve space for the cache file.
	if err := ReserveSpace(ctx, filename, contentSize, blockSize, concurrency); err != nil {
		return fmt.Errorf("failed to reserve space for cache file %s: %w", filename, err)
	}

	// Save the index file
	if err := saveRanges(indexFileName, NewByteRanges(contentSize, blockSize)); err != nil {
		return fmt.Errorf("failed to create index file %s: %w", indexFileName, err)
	}

	return nil
}

// NewLocalDownloadCache creates a new LocalDownloadCache instance.
// It opens the cache file and loads the index file containing the byte ranges
// that have been written to the cache. It returns an error if the files cannot
// be opened or if the index file cannot be loaded.
// The cache file is used to store the actual data, and the index file is used
// to track which byte ranges have been written to the cache.
// The cache and index files must already exist and are expected to be have
// been created using NewFilesForCache.
func NewLocalDownloadCache(filename, indexFileName string) (*LocalDownloadCache, error) {
	cache := &LocalDownloadCache{
		indexName: indexFileName,
	}
	var err error
	cache.cache, err = os.OpenFile(filename, os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file %s: %w", filename, err)
	}
	cache.written, err = loadRanges(indexFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to load ranges from index file %s: %w", indexFileName, err)
	}
	return cache, nil
}

func loadRanges(filename string) (*ByteRanges, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read index file %s: %w", filename, err)
	}
	var br ByteRanges
	if err := json.Unmarshal(data, &br); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index file %s: %w", filename, err)
	}
	return &br, nil
}

func saveRanges(filename string, dr *ByteRanges) error {
	data, err := json.Marshal(dr)
	if err != nil {
		return fmt.Errorf("failed to marshal ranges to JSON: %w", err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write index file %s: %w", filename, err)
	}
	return nil
}

func (c *LocalDownloadCache) Outstanding() iter.Seq[ByteRange] {
	return c.written.NextClear(0)
}

func (c *LocalDownloadCache) Cached() iter.Seq[ByteRange] {
	return c.written.NextSet(0)
}

func (c *LocalDownloadCache) Complete() bool {
	for range c.written.NextClear(0) {
		return false
	}
	return true
}

// Put implements DownloadCache.
func (c *LocalDownloadCache) Put(r ByteRange, data []byte) error {
	if r.From < 0 || r.To > c.written.ContentLength() || r.From >= r.To {
		return fmt.Errorf("invalid range: %s", r)
	}
	if int64(len(data)) != r.Size() {
		return fmt.Errorf("data length %d does not match range size %d", len(data), r.To-r.From)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.cache.WriteAt(data, r.From); err != nil {
		return fmt.Errorf("failed to write data to cache for range %s: %w", r, err)
	}
	if err := c.cache.Sync(); err != nil {
		return fmt.Errorf("failed to sync cache file after writing range %s: %w", r, err)
	}
	c.written.Set(r.From) // Mark the range as cached.
	return saveRanges(c.indexName, c.written)
}

// Get implements DownloadCache.
func (c *LocalDownloadCache) Get(r ByteRange, data []byte) error {
	if r.From < 0 || r.To > c.written.ContentLength() || r.From >= r.To {
		return fmt.Errorf("invalid range: %s", r)
	}
	if int64(len(data)) != r.Size() {
		return fmt.Errorf("data length %d does not match range size %d", len(data), r.To-r.From)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.written.IsClear(r.From) {
		return fmt.Errorf("range %s is not cached", r)
	}
	if _, err := c.cache.ReadAt(data, r.From); err != nil {
		return fmt.Errorf("failed to read data from cache for range %s: %w", r, err)
	}
	return nil
}

// ContentLengthAndBlockSize implements DownloadCache.
func (c *LocalDownloadCache) ContentLengthAndBlockSize() (int64, int) {
	return c.written.contentSize, c.written.blockSize
}

// Close implements DownloadCache.
func (c *LocalDownloadCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache != nil {
		if err := c.cache.Close(); err != nil {
			return fmt.Errorf("failed to close cache file: %w", err)
		}
		c.cache = nil
	}
	return nil
}
