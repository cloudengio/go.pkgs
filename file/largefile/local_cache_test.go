// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file/largefile"
)

// GenAI: gemini 2.5 wrote these tests, with some errors and a massive number of
// lint errors. It could not keep up with ongoing changes and edits in a useful
// way so much hand editing was required.

// Helper to create a temporary file with specific content.
func createTempFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("Failed to create temp file %s: %v", path, err)
	}
	return path
}

// Helper to collect iter.Seq[ByteRange] into a slice
func collectByteRanges(seq iter.Seq[largefile.ByteRange]) []largefile.ByteRange {
	var result []largefile.ByteRange
	for br := range seq {
		result = append(result, br)
	}
	return result
}

func collectCacheByteRanges(getter func(int, *largefile.ByteRange) int) []largefile.ByteRange {
	var result []largefile.ByteRange
	var br largefile.ByteRange
	for n := getter(0, &br); n >= 0; n = getter(n, &br) {
		result = append(result, br)
	}
	return result
}

func loadIndexFile(t *testing.T, indexFilePath string, contentSize int64, blockSize int) *largefile.ByteRanges {
	t.Helper()
	data, err := os.ReadFile(indexFilePath)
	if err != nil {
		t.Fatalf("Failed to read index file %s: %v", indexFilePath, err)
	}
	var br largefile.ByteRanges
	if err := json.Unmarshal(data, &br); err != nil {
		t.Fatalf("Failed to unmarshal index file %s: %v", indexFilePath, err)
	}
	if br.ContentLength() != contentSize {
		t.Errorf("index file content length %d does not match expected size %d", br.ContentLength(), contentSize)
	}
	if br.BlockSize() != blockSize {
		t.Errorf("index file block size %d does not match expected block size %d", br.BlockSize(), blockSize)
	}
	return &br
}

func writeAt(t *testing.T, cache largefile.DownloadCache, buf []byte, off int64) {
	t.Helper()
	n, err := cache.WriteAt(buf, off)
	if err != nil {
		t.Fatalf("WriteAt failed: %v", err)
	}
	if got, want := int64(n), int64(len(buf)); got != want {
		t.Errorf("WriteAt() = %v, want %v", got, want)
	}
}

func TestNewFilesForCache(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 1036
	const blockSize int = 128
	const concurrency int = 1

	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, concurrency, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache() error = %v", err)
	}

	// Check if cache file was created (ReserveSpace should handle this)
	fi, err := os.Stat(cacheFilePath)
	if err != nil {
		t.Fatalf("os.Stat(cacheFilePath) error = %v", err)
	}
	// Note: ReserveSpace might create a sparse file or a fully allocated one.
	// Its exact size on disk might vary, but logical size should be at least contentSize.
	// For this test, we primarily care it exists and the index is correct.
	if fi.Size() < contentSize && fi.Size() != 0 { // Some OS might report 0 for sparse files until written.
		// This check is tricky due to sparse files. If ReserveSpace guarantees a certain behavior, test that.
		// For now, existence is key.
		t.Logf("Cache file size is %d, expected at least %d (may be sparse)", fi.Size(), contentSize)
	}

	// Check if index file was created and is valid
	br := loadIndexFile(t, indexFilePath, contentSize, blockSize)
	// Initially, all ranges should be clear (outstanding)
	outstanding := collectByteRanges(br.AllClear(0))
	numExpectedBlocks := (contentSize + int64(blockSize) - 1) / int64(blockSize)
	if int64(len(outstanding)) != numExpectedBlocks {
		t.Errorf("Expected %d outstanding blocks, got %d", numExpectedBlocks, len(outstanding))
	}

	// Test with empty filename
	err = largefile.CreateNewFilesForCache(ctx, "", indexFilePath, contentSize, blockSize, concurrency, nil)
	if err == nil || !strings.Contains(err.Error(), "filename cannot be empty") {
		t.Errorf("Expected error for empty cache filename, got %v", err)
	}
	err = largefile.CreateNewFilesForCache(ctx, cacheFilePath, "", contentSize, blockSize, concurrency, nil)
	if err == nil || !strings.Contains(err.Error(), "filename cannot be empty") {
		t.Errorf("Expected error for empty index filename, got %v", err)
	}

	// Test removing existing files
	createTempFile(t, tmpDir, "existing_cache.dat", []byte("old data"))
	createTempFile(t, tmpDir, "existing_cache.idx", []byte("old index"))
	err = largefile.CreateNewFilesForCache(ctx, filepath.Join(tmpDir, "existing_cache.dat"), filepath.Join(tmpDir, "existing_cache.idx"), contentSize, blockSize, concurrency, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache() on existing files error = %v", err)
	}
}

func TestNewLocalDownloadCacheExisting(t *testing.T) {
	ctx := t.Context()
	const contentSize int64 = 1036
	const blockSize int = 128
	const concurrency int = 1

	t.Run("load existing valid cache and index", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheFilePath := filepath.Join(tmpDir, "cache.dat")
		indexFilePath := filepath.Join(tmpDir, "cache.idx")

		// Initialize files using NewFilesForCache
		err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, concurrency, nil)
		if err != nil {
			t.Fatalf("NewFilesForCache() failed: %v", err)
		}

		// Modify the index to simulate some cached data
		br := loadIndexFile(t, indexFilePath, contentSize, blockSize)
		br.Set(int64(blockSize)) // Mark second block as set (offset = blockSize)
		idxData, _ := json.Marshal(br)
		if err := os.WriteFile(indexFilePath, idxData, 0600); err != nil {
			t.Fatalf("Failed to write modified index: %v", err)
		}

		cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
		if err != nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}

		cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
		if err != nil {
			t.Fatalf("NewLocalDownloadCache() error = %v", err)
		}
		defer cache.Close() // Assuming a Close method to close the file handle

		cs, bs := cache.ContentLengthAndBlockSize()
		if cs != contentSize {
			t.Errorf("ContentSize mismatch: got %v, want %v", cs, contentSize)
		}
		if bs != blockSize {
			t.Errorf("BlockSize mismatch: got %v, want %v", bs, blockSize)
		}
		cachedRanges := collectCacheByteRanges(cache.NextCached)
		if len(cachedRanges) != 1 || cachedRanges[0].From != int64(blockSize) {
			t.Errorf("Expected second block to be cached, got %v", cachedRanges)
		}
	})

}

func TestNewLocalDownloadCacheErrors(t *testing.T) {
	const contentSize int64 = 1036
	const blockSize int = 128

	t.Run("index file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheFilePath := filepath.Join(tmpDir, "cache.dat")
		indexFilePath := filepath.Join(tmpDir, "nonexistent.idx")
		// Create a dummy cache file so OpenFile doesn't fail for that reason
		createTempFile(t, tmpDir, "cache.dat", []byte{})

		_, _, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
		if err == nil {
			t.Fatal("NewLocalDownloadCache() error = nil, wantErr true for missing index")
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("NewLocalDownloadCache() error = %q, want 'failed to load ranges' due to 'no such file or directory'", err.Error())
		}
	})

	t.Run("cache file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheFilePath := filepath.Join(tmpDir, "nonexistent.dat")
		indexFilePath := filepath.Join(tmpDir, "cache.idx")
		// Create a dummy index file
		br := largefile.NewByteRanges(contentSize, blockSize)
		idxData, _ := json.Marshal(br)
		createTempFile(t, tmpDir, "cache.idx", idxData)

		_, _, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
		if err == nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("NewLocalDownloadCache() error = %q, want 'failed to open cache file' due to 'no such file or directory'", err.Error())
		}
	})

	t.Run("cannot open cache file (e.g. is a directory)", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheFilePath := filepath.Join(tmpDir, "cache.dat") // This will be a dir
		indexFilePath := filepath.Join(tmpDir, "cache.idx")
		if err := os.Mkdir(cacheFilePath, 0755); err != nil {
			t.Fatalf("Failed to create dir for test: %v", err)
		}
		// Create a dummy index file
		br := largefile.NewByteRanges(contentSize, blockSize)
		idxData, _ := json.Marshal(br)
		createTempFile(t, tmpDir, "cache.idx", idxData)

		_, _, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
		if err == nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}
		// Error message might be OS-dependent ("is a directory" on Unix, "Access is denied" or similar on Win)
		if !strings.Contains(err.Error(), "failed to open cache file") {
			t.Errorf("NewLocalDownloadCache() error = %q, want to contain 'failed to open cache file'", err.Error())
		}
	})

	t.Run("corrupt index file", func(t *testing.T) {
		tmpDir := t.TempDir()
		cacheFilePath := filepath.Join(tmpDir, "cache.dat")
		indexFilePath := filepath.Join(tmpDir, "cache.idx")
		createTempFile(t, tmpDir, "cache.dat", []byte{}) // Dummy cache file
		createTempFile(t, tmpDir, "cache.idx", []byte("this is not json"))
		cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
		if err != nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}
		defer cacheFile.Close()
		defer indexFile.Close()

		_, err = largefile.NewLocalDownloadCache(cacheFile, indexFile)
		if err == nil {
			t.Fatal("NewLocalDownloadCache() error = nil, wantErr true for corrupt index")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal index file") {
			t.Errorf("NewLocalDownloadCache() error = %q, want to contain 'failed to unmarshal index file'", err.Error())
		}
	})
}

func TestLocalDownloadCachePutGetRoundtrip(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	const contentSize int64 = 256
	const blockSize int = 64 // 4 blocks
	const concurrency int = 1
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, concurrency, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	// Test data for one block
	blockData := make([]byte, blockSize)
	for i := range blockData {
		blockData[i] = byte(i)
	}

	// Put the second block (offset 64)
	putAt := int64(64)
	_, err = cache.WriteAt(blockData, putAt)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Verify index file was saved and reflects the change
	idx := loadIndexFile(t, indexFilePath, contentSize, blockSize)
	if !idx.IsSet(putAt) {
		t.Errorf("Saved ranges in index do not reflect Put operation for offset %d", putAt)
	}

	// Get the data back
	readBuffer := make([]byte, blockSize)
	_, err = cache.ReadAt(readBuffer, putAt)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(readBuffer, blockData) {
		t.Errorf("Get() data mismatch: got %x, want %x", readBuffer, blockData)
	}

	// Try to Get a non-cached range
	nonCachedRange := int64(0)
	_, err = cache.ReadAt(readBuffer, nonCachedRange)
	if err == nil {
		t.Fatal("Get() on non-cached range expected error, got nil")
	}
	if !strings.Contains(err.Error(), "is not cached") {
		t.Errorf("Get() on non-cached range error = %q, want to contain 'is not cached'", err.Error())
	}

	// Put another block
	blockData2 := make([]byte, blockSize)
	for i := range blockData2 {
		blockData2[i] = byte(i + 100)
	}
	putAt2 := int64(0)
	_, err = cache.WriteAt(blockData2, putAt2)
	if err != nil {
		t.Fatalf("Put() for second block error = %v", err)
	}

	// Get first block
	_, err = cache.ReadAt(readBuffer, putAt2)
	if err != nil {
		t.Fatalf("Get() for second block error = %v", err)
	}
	if !bytes.Equal(readBuffer, blockData2) {
		t.Errorf("Get() data mismatch for second block: got %x, want %x", readBuffer, blockData2)
	}

	// Test Put with invalid range (e.g., beyond content size)
	_, err = cache.WriteAt(make([]byte, blockSize), contentSize)
	if err == nil {
		t.Fatal("Put() with invalid range expected error, got nil")
	}
	if !errors.Is(err, largefile.ErrCacheInvalidOffset) {
		t.Errorf("Put() with invalid range error = %q, want to contain 'invalid range'", err.Error())
	}

	// Test Put with data size mismatch
	mismatchData := make([]byte, blockSize-1)
	_, err = cache.WriteAt(mismatchData, putAt)
	if err == nil {
		t.Fatal("Put() with data size mismatch expected error, got nil")
	}
	if !errors.Is(err, largefile.ErrCacheInvalidBlockSize) {
		t.Errorf("Put() with data size mismatch error = %q, want to contain 'data length ... does not match range size'", err.Error())
	}
}

func TestLocalDownloadCache_Iterators(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 256 // 4 blocks of 64
	const blockSize int = 64
	const concurrency int = 1
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, concurrency, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}

	cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	// Put blocks 0 and 2
	_, err = cache.WriteAt(make([]byte, 64), 0)
	if err != nil {
		t.Fatalf("Put for block 0 failed: %v", err)
	}
	_, err = cache.WriteAt(make([]byte, 64), 128)
	if err != nil {
		t.Fatalf("Put for block 2 failed: %v", err)
	}

	t.Run("Cached", func(t *testing.T) {
		gotRanges := collectCacheByteRanges(cache.NextCached)
		wantRanges := []largefile.ByteRange{
			{From: 0, To: 63},
			{From: 128, To: 191},
		}
		if !reflect.DeepEqual(gotRanges, wantRanges) {
			t.Errorf("Cached() got %v, want %v", gotRanges, wantRanges)
		}
	})

	t.Run("Outstanding", func(t *testing.T) {
		gotRanges := collectCacheByteRanges(cache.NextOutstanding)
		wantRanges := []largefile.ByteRange{
			{From: 64, To: 127},  // Block 1
			{From: 192, To: 255}, // Block 3
		}
		if !reflect.DeepEqual(gotRanges, wantRanges) {
			t.Errorf("Outstanding() got %v, want %v", gotRanges, wantRanges)
		}
	})

	t.Run("all cached", func(t *testing.T) {
		tmpDirAllCached := t.TempDir()
		cacheFileAll := filepath.Join(tmpDirAllCached, "cache.dat")
		indexFileAll := filepath.Join(tmpDirAllCached, "cache.idx")

		err := largefile.CreateNewFilesForCache(ctx, cacheFileAll, indexFileAll, 128, 64, concurrency, nil)
		if err != nil {
			t.Fatalf("NewFilesForCache for all_cached failed: %v", err)
		}
		cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFileAll, indexFileAll)
		if err != nil {
			t.Fatalf("Failed to open cache files: %v", err)
		}
		cacheAll, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
		if err != nil {
			t.Fatalf("NewLocalDownloadCache for all_cached failed: %v", err)
		}
		defer cacheAll.Close()

		if _, err := cacheAll.WriteAt(make([]byte, 64), 0); err != nil {
			t.Fatalf("Put for first block in all_cached failed: %v", err)
		}
		if _, err := cacheAll.WriteAt(make([]byte, 64), 64); err != nil {
			t.Fatalf("Put for second block in all_cached failed: %v", err)
		}

		gotCached := collectCacheByteRanges(cacheAll.NextCached)
		wantCached := []largefile.ByteRange{{From: 0, To: 63}, {From: 64, To: 127}}
		if !reflect.DeepEqual(gotCached, wantCached) {
			t.Errorf("All Cached() got %v, want %v", gotCached, wantCached)
		}

		gotOutstanding := collectCacheByteRanges(cacheAll.NextOutstanding)
		if len(gotOutstanding) != 0 { // Expect empty slice
			t.Errorf("All Cached - Outstanding() got %v, want empty", gotOutstanding)
		}
	})
}

func TestLocalDownloadCache_ContentLengthAndBlockSize(t *testing.T) {
	ctx := context.Background()
	const contentSize int64 = 512
	const blockSize int = 32
	const concurrency int = 1
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, concurrency, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close() // Add Close

	cs, bs := cache.ContentLengthAndBlockSize()
	if cs != contentSize {
		t.Errorf("ContentLength() got %v, want %v", cs, contentSize)
	}
	if bs != blockSize {
		t.Errorf("BlockSize() got %v, want %v", bs, blockSize)
	}
}

func TestLocalDownloadCache_Tail(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	const contentSize int64 = 256
	const blockSize int = 64 // 4 blocks
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	t.Run("Tail returns -1,-1 if nothing cached", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		got := cache.Tail(ctx)
		if got.From != -1 || got.To != -1 {
			t.Errorf("Tail() got %v, want {-1,-1}", got)
		}
	})

	t.Run("Tail returns correct range after each block", func(t *testing.T) {
		block := make([]byte, blockSize)
		for i := range block {
			block[i] = byte(i)
		}
		for i := range 4 {
			_, err := cache.WriteAt(block, int64(i*blockSize))
			if err != nil {
				t.Fatalf("WriteAt failed: %v", err)
			}
			got := cache.Tail(context.Background())
			want := largefile.ByteRange{From: 0, To: int64((i+1)*blockSize - 1)}
			if got != want {
				t.Errorf("Tail() after block %d got %v, want %v", i, got, want)
			}
		}
	})

	t.Run("Tail blocks until next contiguous block is cached", func(t *testing.T) {
		// New cache with a gap at block 1
		tmpDir2 := t.TempDir()
		cacheFilePath2 := filepath.Join(tmpDir2, "cache.dat")
		indexFilePath2 := filepath.Join(tmpDir2, "cache.idx")
		_ = largefile.CreateNewFilesForCache(ctx, cacheFilePath2, indexFilePath2, contentSize, blockSize, 1, nil)
		cacheFile2, indexFile2, _ := largefile.OpenCacheFiles(cacheFilePath2, indexFilePath2)
		cache2, _ := largefile.NewLocalDownloadCache(cacheFile2, indexFile2)
		defer cache2.Close()

		block := make([]byte, blockSize)
		writeAt(t, cache2, block, 0)                  // block 0
		writeAt(t, cache2, block, int64(2*blockSize)) // block 2

		// Tail should return up to block 0
		got := cache2.Tail(context.Background())
		want := largefile.ByteRange{From: 0, To: int64(blockSize - 1)}
		if got != want {
			t.Errorf("Tail() with gap got %v, want %v", got, want)
		}

		// Start Tail in a goroutine, should block until block 1 is cached
		done := make(chan largefile.ByteRange, 1)
		go func() {
			done <- cache2.Tail(context.Background())
		}()
		time.Sleep(20 * time.Millisecond)
		select {
		case <-done:
			t.Fatal("Tail() returned before gap was filled")
		default:
		}
		writeAt(t, cache2, block, int64(blockSize)) // fill the gap

		select {
		case got := <-done:
			want := largefile.ByteRange{From: 0, To: int64(3*blockSize - 1)}
			if got != want {
				t.Errorf("Tail() after filling gap got %v, want %v", got, want)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Tail() did not return after filling gap")
		}
	})

	t.Run("Tail returns immediately if already complete", func(t *testing.T) {
		// All blocks already cached from previous test
		got := cache.Tail(context.Background())
		want := largefile.ByteRange{From: 0, To: contentSize - 1}
		if got != want {
			t.Errorf("Tail() got %v, want %v", got, want)
		}
	})

	t.Run("Tail returns -1,-1 if context cancelled before extension", func(t *testing.T) {
		tmpDir3 := t.TempDir()
		cacheFilePath3 := filepath.Join(tmpDir3, "cache.dat")
		indexFilePath3 := filepath.Join(tmpDir3, "cache.idx")
		_ = largefile.CreateNewFilesForCache(ctx, cacheFilePath3, indexFilePath3, contentSize, blockSize, 1, nil)
		cacheFile3, indexFile3, _ := largefile.OpenCacheFiles(cacheFilePath3, indexFilePath3)
		cache3, _ := largefile.NewLocalDownloadCache(cacheFile3, indexFile3)
		defer cache3.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		got := cache3.Tail(ctx)
		if got.From != -1 || got.To != -1 {
			t.Errorf("Tail() got %v, want {-1,-1}", got)
		}
	})
}

func TestLocalDownloadCache_ReadAt_ArbitraryRanges(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	const contentSize int64 = (4 * 64) + 10
	const blockSize int = 64 // 4 blocks
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	err := largefile.CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil)
	if err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := largefile.OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("Failed to open cache files: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	defer cache.Close()

	// Write blocks 0 and 1
	block := make([]byte, blockSize)
	for i := range block {
		block[i] = byte(i)
	}
	_, err = cache.WriteAt(block, 0)
	if err != nil {
		t.Fatalf("WriteAt block 0 failed: %v", err)
	}
	_, err = cache.WriteAt(block, int64(blockSize))
	if err != nil {
		t.Fatalf("WriteAt block 1 failed: %v", err)
	}

	// Try to read across both cached blocks (should succeed)
	readBuf := make([]byte, 2*blockSize)
	n, err := cache.ReadAt(readBuf, 0)
	if err != nil {
		t.Errorf("ReadAt across two cached blocks failed: %v", err)
	}
	if n != 2*blockSize {
		t.Errorf("ReadAt read %d bytes, want %d", n, 2*blockSize)
	}
	if !bytes.Equal(readBuf[:blockSize], block) || !bytes.Equal(readBuf[blockSize:2*blockSize], block) {
		t.Errorf("ReadAt data mismatch across two blocks")
	}

	// Try to read a range that partially overlaps an uncached block (should fail)
	readBuf = make([]byte, blockSize+16)
	_, err = cache.ReadAt(readBuf, int64(blockSize))
	if err == nil {
		t.Error("ReadAt should fail when reading into uncached block, but got nil error")
	}

	// Write block 2
	_, err = cache.WriteAt(block, int64(2*blockSize))
	if err != nil {
		t.Fatalf("WriteAt block 2 failed: %v", err)
	}

	// Try to read a range that spans blocks 1 and 2 (should succeed)
	readBuf = make([]byte, blockSize*2)
	n, err = cache.ReadAt(readBuf, int64(blockSize))
	if err != nil {
		t.Errorf("ReadAt across blocks 1 and 2 failed: %v", err)
	}
	if n != 2*blockSize {
		t.Errorf("ReadAt read %d bytes, want %d", n, 2*blockSize)
	}

	// Try to read a range that starts at an uncached block (should fail)
	readBuf = make([]byte, blockSize)
	_, err = cache.ReadAt(readBuf, int64(3*blockSize))
	if err == nil {
		t.Error("ReadAt should fail when reading from uncached block, but got nil error")
	}

	readBuf = make([]byte, 10)
	_, err = cache.ReadAt(readBuf, (int64(blockSize) * 4))
	if err == nil {
		t.Error("ReadAt should fail when reading from uncached block, but got nil error")
	}

	// Write partial last block
	_, err = cache.WriteAt(block[:10], (int64(blockSize) * 4)) // Write to the end of the file
	if err != nil {
		t.Fatalf("WriteAt trailing failed: %v", err)
	}

	readBuf = make([]byte, blockSize+10)
	_, err = cache.ReadAt(readBuf, (int64(blockSize) * 3))
	if err == nil {
		t.Error("ReadAt should fail when reading from uncached block, but got nil error")
	}

	_, err = cache.WriteAt(block, (int64(blockSize) * 3)) // Write to the end of the file
	if err != nil {
		t.Fatalf("WriteAt trailing failed: %v", err)
	}

	n, err = cache.ReadAt(readBuf, (int64(blockSize) * 3))
	if err != nil {
		t.Error("ReadAt should be able to read overlapping trailing data, but got error: " + err.Error() + "'")
	}
	if n != blockSize+10 {
		t.Errorf("ReadAt read %d bytes, want %d", n, 10+10)
	}
}
