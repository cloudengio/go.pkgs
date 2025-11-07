// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file/diskusage"
)

// DownloadCache is an interface for caching byte ranges of large files
// to support resumable downloads.
type DownloadCache interface {
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)

	// CachedBytesAndBlocks returns the total number of bytes and blocks already stored in
	// the cache.
	CachedBytesAndBlocks() (bytes, blocks int64)

	// NextOutstanding finds the next byte range that has not been cached
	// starting from the specified 'start' index. Its return value is either
	// -1 if there are no more outstanding ranges, or the value of the next
	// starting index to continue searching at.
	// To iterate over all outstanding ranges, call this method repeatedly
	// until it returns -1 as follows:
	//    for start := NextOutstanding(0, &br); start != -1; start = NextOutstanding(start, &br) {
	//        // Do something with the byte range br.
	//    }
	NextOutstanding(start int, br *ByteRange) int

	// NextCached finds the next byte range that has been cached in the same manner
	// as NextOutstanding.
	NextCached(start int, br *ByteRange) int

	// Tail returns the contiguous range of bytes that have been cached so far.
	// If this has not grown since the last call to Tail, Tail will block until
	// the tail is extended. If the context is done before the tail is extended,
	// it returns a ByteRange with From and To set to -1.
	Tail(context.Context) ByteRange

	// Complete returns true if all byte ranges have been cached.
	Complete() bool

	// WriteAt writes at most blocksize bytes starting at the specified offset.
	// Offset must be aligned with the block boundaries. It returns an error if
	// data is not exactly blocksize bytes long, unless the offset is at the
	// end of the file, in which case it must be (content length % blocksize)
	// bytes long.
	WriteAt(data []byte, off int64) (int, error)

	// ReadAt reads at most len(data) bytes starting at the specified offset.
	// It returns an error if any of the data to be read is not already cached.
	// The offset need not be aligned with the block boundaries.
	ReadAt(data []byte, off int64) (int, error)
}

// LocalDownloadCache is a concrete implementation of RangeCache that uses
// a local file to cache byte ranges of large files.
// It allows for concurrent access.
type LocalDownloadCache struct {
	mu                sync.RWMutex // Protects access to the cache file.
	data              CacheFileReadWriter
	indexStore        *indexStore
	lastTailByteRange int64 // The last byte range returned by Tail.
	lastBlockSize     int64 // size of last block
	lastBlockOffset   int64 // offset of last block
}

// CreateNewFilesForCache creates a new cache file and an index file for caching
// byte ranges of large files. It will remove any existing files with the same names
// before creating new ones. It reserves space for the cache file and
// initializes the index file with the specified content size and block size.
// It returns an error if the files cannot be created or if the space cannot
// be reserved.
// The index file is used to track which byte ranges have been written to the cache.
// The cache file is used to store the actual data.
// The contentSize is the total size of the file in bytes, blockSize is the preferred
// block size for downloading the file, and concurrency is the number of concurrent
// writes used to reserve space for the cache file on systems that require writing
// to the file to reserve space (e.g., non-Linux systems).
func CreateNewFilesForCache(ctx context.Context, filename, indexFileName string, contentSize int64, blockSize, concurrency int, progressCh chan<- int64) error {
	for _, fn := range []string{filename, indexFileName} {
		if len(fn) == 0 {
			return fmt.Errorf("filename cannot be empty")
		}
		if err := os.Remove(fn); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing file %s: %w", fn, err)
		}
	}

	// Reserve space for the cache file, use 64MiB block size for reservation.
	if err := ReserveSpace(ctx, filename, contentSize, int(64*diskusage.MiB), concurrency, progressCh); err != nil {
		return fmt.Errorf("failed to reserve space for cache file %s: %w", filename, err)
	}

	fs, err := os.Create(indexFileName)
	if err != nil {
		return fmt.Errorf("failed to create index file %s: %w", indexFileName, err)
	}
	defer fs.Close()
	// Save the index file
	idx := &indexStore{
		ByteRanges: NewByteRanges(contentSize, blockSize),
		wr:         fs,
	}
	if err := idx.save(); err != nil {
		return fmt.Errorf("failed to create index file %s: %w", indexFileName, err)
	}
	return nil
}

// OpenCacheFiles opens the cache and index files for reading and writing.
func OpenCacheFiles(data, index string) (CacheFileReadWriter, CacheFileReadWriter, error) {
	dataFile, err := os.OpenFile(data, os.O_RDWR, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open cache file %s: %w", data, err)
	}
	indexFile, err := os.OpenFile(index, os.O_RDWR, 0600)
	if err != nil {
		dataFile.Close() // Close the data file if index file cannot be opened.
		return nil, nil, fmt.Errorf("failed to open index file %s: %w", index, err)
	}
	return dataFile, indexFile, nil
}

// CacheFilesExist checks if the cache and index files exist.
func CacheFilesExist(data, index string) (bool, bool, error) {
	dataExists := false
	indexExists := false
	if _, err := os.Stat(data); err == nil {
		dataExists = true
	} else if !os.IsNotExist(err) {
		return false, false, fmt.Errorf("failed to stat cache file %s: %w", data, err)
	}
	if _, err := os.Stat(index); err == nil {
		indexExists = true
	} else if !os.IsNotExist(err) {
		return false, false, fmt.Errorf("failed to stat index file %s: %w", index, err)
	}
	return dataExists, indexExists, nil
}

// CacheFileReadWriter is an interface that combines the functionality of
// io.Reader, io.ReaderAt, io.WriterAt, and io.Closer for reading and writing
// to the cache and index files.
// It also includes a Sync method to ensure that all writes are flushed to disk.
type CacheFileReadWriter interface {
	io.Reader
	io.Writer
	io.ReaderAt
	io.WriterAt
	io.Closer
	Name() string // Name returns the name of the file.
	Sync() error  // Sync ensures that all writes to the cache file are flushed to disk.
}

// NewLocalDownloadCache creates a new LocalDownloadCache instance.
// It opens the cache file and loads the index file containing the byte ranges
// that have been written to the cache. It returns an error if the files cannot
// be opened or if the index file cannot be loaded.
// The cache file is used to store the actual data, and the index file is used
// to track which byte ranges have been written to the cache.
// The cache and index files must already exist and are expected to be have
// been created using NewFilesForCache.
func NewLocalDownloadCache(dataReadWriter, indexReadWriter CacheFileReadWriter) (*LocalDownloadCache, error) {
	if dataReadWriter == nil || indexReadWriter == nil {
		return nil, fmt.Errorf("data and index read/writer arguments must be non-nil")
	}
	cache := &LocalDownloadCache{
		data:       dataReadWriter,
		indexStore: &indexStore{wr: indexReadWriter},
	}
	if err := cache.indexStore.load(); err != nil {
		return nil, fmt.Errorf("failed to load index file %s: %w", indexReadWriter.Name(), err)
	}

	cache.lastBlockSize = cache.indexStore.ContentLength() % int64(cache.indexStore.blockSize)
	cache.lastBlockOffset = cache.indexStore.ContentLength() - cache.lastBlockSize
	if cache.lastBlockSize == 0 {
		cache.lastBlockOffset -= int64(cache.indexStore.blockSize)
		cache.lastBlockSize = int64(cache.indexStore.blockSize)
	}
	return cache, nil
}

type indexStore struct {
	*ByteRanges
	wr CacheFileReadWriter
}

func (i *indexStore) save() error {
	data, err := json.Marshal(i.ByteRanges)
	if err != nil {
		return newInternalCacheError(fmt.Errorf("failed to marshal ranges to JSON: %w", err))
	}
	n, err := i.wr.WriteAt(data, 0)
	if err != nil {
		return newInternalCacheError(fmt.Errorf("failed to write index file %s: %w", i.wr.Name(), err))
	}
	if n != len(data) {
		return newInternalCacheError(fmt.Errorf("failed to write all data to the index file %s: wrote %d bytes, expected %d: %w", i.wr.Name(), n, len(data), err))
	}
	return nil
}

func (i *indexStore) load() error {
	buf, err := io.ReadAll(i.wr)
	if err != nil {
		return newInternalCacheError(fmt.Errorf("failed to read index file %s: %w", i.wr.Name(), err))
	}
	if err := json.Unmarshal(buf, &i.ByteRanges); err != nil {
		return newInternalCacheError(fmt.Errorf("failed to unmarshal index file %s: %w", i.wr.Name(), err))
	}
	return nil
}

// NextOutstanding implements DownloadCache. It returns the next, if any,
// uncached byte range starting from the specified index.
func (c *LocalDownloadCache) NextOutstanding(start int, br *ByteRange) int {
	return c.indexStore.NextClear(start, br)
}

// NextCached implements DownloadCache. It returns the next, if any,
// cached byte range starting from the specified index.
func (c *LocalDownloadCache) NextCached(start int, br *ByteRange) int {
	return c.indexStore.NextSet(start, br)
}

// Complete implements DownloadCache. It returns true if all byte ranges
// have been cached, meaning there are no more uncached ranges.
func (c *LocalDownloadCache) Complete() bool {
	var br ByteRange
	i := c.indexStore.NextClear(0, &br)
	return i == -1
}

func (c *LocalDownloadCache) CachedBytesAndBlocks() (bytes, blocks int64) {
	var br ByteRange
	for n := c.indexStore.NextSet(0, &br); n != -1; n = c.indexStore.NextSet(n, &br) {
		bytes += br.Size()
		blocks++
	}
	// Return the total number of bytes cached.
	return bytes, blocks
}

func (c *LocalDownloadCache) validateOffsetAndSize(off, size int64) error {
	if c.indexStore == nil {
		return newInternalCacheError(errors.New("index store is not initialized"))
	}
	if off < 0 || off > c.lastBlockOffset {
		return fmt.Errorf("%d must be between 0 and %d: %w", off, c.lastBlockOffset, ErrCacheInvalidOffset)
	}
	if off%int64(c.indexStore.blockSize) != 0 {
		return fmt.Errorf("offset %d is not aligned with block size %d: %w", off, c.indexStore.blockSize, ErrCacheInvalidOffset)
	}
	if size > int64(c.indexStore.blockSize) {
		return fmt.Errorf("data size %d must be less than or equal to %d: %w", size, c.indexStore.blockSize, ErrCacheInvalidBlockSize)
	}
	if off == c.lastBlockOffset {
		if size != c.lastBlockSize {
			return fmt.Errorf("data size %d for last block at %d: must be %d: %w", size, off, c.lastBlockSize, ErrCacheInvalidBlockSize)
		}
		return nil
	}
	if size != int64(c.indexStore.blockSize) {
		return fmt.Errorf("data size %d for offset %d: must be %d: %w", size, off, c.indexStore.blockSize, ErrCacheInvalidBlockSize)
	}
	return nil
}

// WriteAt implements DownloadCache.
func (c *LocalDownloadCache) WriteAt(data []byte, off int64) (int, error) {
	if err := c.validateOffsetAndSize(off, int64(len(data))); err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil || c.indexStore.wr == nil {
		return 0, newInternalCacheError(errors.New("cache files are not initialized"))
	}
	n, err := c.data.WriteAt(data, off)
	if err != nil {
		return 0, newInternalCacheError(fmt.Errorf("failed to write data to cache for offset %d: %w", off, err))
	}
	if n != len(data) {
		return n, newInternalCacheError(fmt.Errorf("failed to write all data to cache for offset %d: wrote %d bytes, expected %d: %w", off, n, len(data), io.ErrShortWrite)) // Ensure all data is written.
	}
	if err := c.data.Sync(); err != nil {
		return 0, newInternalCacheError(fmt.Errorf("failed to sync cache file after writing offset %d: %w", off, err))
	}
	c.indexStore.Set(off) // Mark the range as cached.
	if err := c.indexStore.save(); err != nil {
		return 0, newInternalCacheError(fmt.Errorf("failed to save index file %s after writing offset %d: %w", c.indexStore.wr.Name(), off, err))
	}
	return n, nil
}

func (c *LocalDownloadCache) validateOffset(off, size int64) error {
	if c.indexStore == nil {
		return newInternalCacheError(errors.New("index store is not initialized"))
	}
	if off < 0 || off >= c.indexStore.contentSize {
		return fmt.Errorf("%d must be between 0 and %d: %w", off, c.indexStore.contentSize, ErrCacheInvalidOffset)
	}
	if off+size > c.indexStore.contentSize {
		return fmt.Errorf("offset %d with size %d exceeds content size %d: %w", off, size, c.indexStore.contentSize, ErrCacheInvalidBlockSize)
	}
	return nil
}

func (c *LocalDownloadCache) validateCachedLocked(off, size int64) error {
	for i := off; i < off+size; i += int64(c.indexStore.blockSize) {
		if c.indexStore.IsClear(i) {
			return fmt.Errorf("offset %d is not cached: %w", i, ErrCacheUncachedRange)
		}
	}
	return nil
}

// ReadAt implements DownloadCache.
func (c *LocalDownloadCache) ReadAt(data []byte, off int64) (int, error) {
	if err := c.validateOffset(off, int64(len(data))); err != nil {
		return 0, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.data == nil || c.indexStore.wr == nil {
		return 0, newInternalCacheError(errors.New("cache files are not initialized"))
	}
	if err := c.validateCachedLocked(off, int64(len(data))); err != nil {
		return 0, err
	}
	n, err := c.data.ReadAt(data, off)
	if err != nil {
		return 0, newInternalCacheError(fmt.Errorf("failed to read all data from cache for offset %d: %w", off, err))
	}
	if n != len(data) {
		return n, newInternalCacheError(fmt.Errorf("failed to read all data from cache for offset %d: read %d bytes, expected %d: %w", off, n, len(data), io.ErrUnexpectedEOF))
	}
	return n, nil
}

// ContentLengthAndBlockSize implements DownloadCache.
func (c *LocalDownloadCache) ContentLengthAndBlockSize() (int64, int) {
	return c.indexStore.contentSize, c.indexStore.blockSize
}

func (c *LocalDownloadCache) syncAndClose(f CacheFileReadWriter) error {
	if err := f.Sync(); err != nil {
		f.Close() //nolint:errcheck
		return newInternalCacheError(fmt.Errorf("failed to sync %s: %w", f.Name(), err))
	}
	if err := f.Close(); err != nil {
		return newInternalCacheError(fmt.Errorf("failed to close %s: %w", f.Name(), err))
	}
	return nil
}

// Close implements DownloadCache.
func (c *LocalDownloadCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var errs errors.M
	errs.Append(c.syncAndClose(c.data))
	errs.Append(c.syncAndClose(c.indexStore.wr))
	return errs.Err()
}

func (c *LocalDownloadCache) isExtended(to int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	done := to > c.lastTailByteRange
	if done {
		c.lastTailByteRange = to
	}
	return done
}

// Tail implements DownloadCache. It returns the contiguous range of bytes
// that have been cached so far. If this has not grown since the last call to
// Tail, Tail will block until the tail is extended.
func (c *LocalDownloadCache) Tail(ctx context.Context) ByteRange {
	if tail, ok := c.indexStore.Tail(); ok && c.isExtended(tail.To) {
		return tail
	}
	ch := c.indexStore.Notify()
	select {
	case <-ctx.Done():
		return ByteRange{From: -1, To: -1} // Return an empty range if the context is done.
	case <-ch:
	}
	tail, _ := c.indexStore.Tail() // ok will always be true after a notification is received.
	c.mu.Lock()
	c.lastTailByteRange = tail.To
	c.mu.Unlock()
	return tail
}
