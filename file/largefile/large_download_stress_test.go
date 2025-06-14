// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/largefile"
	"cloudeng.io/net/ratecontrol"
)

type mockLargeFile struct {
	size      int64
	blockSize int
	failRatio int
}

func (m *mockLargeFile) ContentLengthAndBlockSize(context.Context) (int64, int, error) {
	return m.size, m.blockSize, nil // Mock implementation, returns size and block size
}
func (m *mockLargeFile) Checksum(context.Context) (largefile.ChecksumType, string, error) {
	return largefile.NoChecksum, "", nil // Mock implementation, no checksum
}

func (m *mockLargeFile) GetReader(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	//nolint:gosec // G404
	if m.failRatio > 0 && rand.Intn(10) < m.failRatio {
		return nil, &noRetryResponse{}, fmt.Errorf("mock failure for testing")
	}
	buf := make([]byte, to-from+1)
	val := from
	idx := 0
	for i := from; i <= to; i += 4 {
		binary.BigEndian.PutUint32(buf[idx:], uint32(val))
		val += 4
		idx += 4
	}
	return io.NopCloser(bytes.NewReader(buf)), &noRetryResponse{}, nil
}

type noBackoff struct{}

func (nb noBackoff) Retries() int {
	return 0
}

func (nb noBackoff) Wait(_ context.Context, _ any) (bool, error) {
	return false, nil
}

type jitterRateLimiter struct{}

func (j *jitterRateLimiter) Wait(context.Context) error {
	// Simulate a random delay to reorder downloads.
	//nolint:gosec // G404
	delay := time.Duration(rand.Int63n(int64(time.Millisecond * 10)))
	time.Sleep(delay)
	return nil
}

func (j *jitterRateLimiter) BytesTransferred(int) {
	// No-op for this mock, as we don't track bytes transferred.
}

func (j *jitterRateLimiter) Backoff() ratecontrol.Backoff {
	return &noBackoff{}
}

func validateCacheFile(t *testing.T, cacheFile string, expectedSize int64) {
	t.Helper()
	fileInfo, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("failed to stat cache file %s: %v", cacheFile, err)
	}
	if fileInfo.Size() != expectedSize {
		t.Errorf("cache file %v size %d does not match expected size %d", fileInfo.Name(), fileInfo.Size(), expectedSize)
	}
	cf, err := os.Open(cacheFile)
	if err != nil {
		t.Fatalf("failed to open cache file %s: %v", cacheFile, err)
	}
	defer cf.Close()
	expected := int32(0)
	for {
		var val int32
		if err := binary.Read(cf, binary.BigEndian, &val); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("failed to read from cache file %s: %v", cacheFile, err)
		}
		if got, want := val, expected; got != want {
			t.Errorf("cache file value %d at position %d does not match expected value %d", got, expected, want)
		}
		expected += 4
	}
	if got, want := expected, int32(expectedSize); got != want {
		t.Errorf("cache file contains wrong last value, got %d, want %d", got, want)
	}
}

func validateIndexFile(t *testing.T, indexFile string, cacheSize int64, blockSize int) {
	t.Helper()
	dr := loadIndexFile(t, indexFile, cacheSize, blockSize)
	s := 0
	for r := range dr.NextSet(0) {
		e := largefile.ByteRange{
			From: int64(s * blockSize),
			To:   min(int64((s+1)*blockSize-1), cacheSize-1),
		}
		if got, want := r, e; !reflect.DeepEqual(got, want) {
			t.Errorf("NextSet(%d) = %v, want %v", s, got, want)
		}
		s++
	}
	for r := range dr.NextClear(0) {
		t.Errorf("NextClear(0) returned unexpected range %v", r)
	}
}

func TestCacheStressTest(t *testing.T) {
	ctx := context.Background()
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	concurrency := 100
	cacheSize := int64(file.KB * 7)
	blockSize := 4 * 16 // Multiple of 4 to allow for writing uint32s to the test data

	if err := largefile.NewFilesForCache(ctx, cacheFile, indexFile, cacheSize, blockSize, concurrency); err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}

	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("failed to create and allocate space for %s: %v", cacheFile, err)
	}

	cSize, cBblockSize := cache.ContentLengthAndBlockSize()
	if cSize != cacheSize || cBblockSize != blockSize {
		t.Fatalf("cache content size %d and block size %d do not match expected values %d and %d", cSize, cBblockSize, cacheSize, blockSize)
	}

	t.Logf("Successfully created and allocated space for %s with size %v bytes in blocks of size %v", cacheFile, cSize, cBblockSize)

	dl := largefile.NewCachingDownloader(&mockLargeFile{size: cacheSize, blockSize: blockSize},
		cache,
		largefile.WithDownloadRateController(&jitterRateLimiter{}),
		largefile.WithDownloadConcurrency(concurrency))

	st, err := dl.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	t.Logf("Download completed successfully, cache file %s is ready for use", cacheFile)
	if err := cache.Close(); err != nil {
		t.Fatalf("failed to close cache: %v", err)
	}

	nBlocks := int64(largefile.NumBlocks(cacheSize, blockSize))
	expected := largefile.DownloadState{
		CachedBytes:      cacheSize,
		CachedBlocks:     nBlocks,
		DownloadedBytes:  cacheSize,
		DownloadedBlocks: nBlocks,
		DownloadSize:     cacheSize,
		DownloadBlocks:   nBlocks,
	}
	if st.Duration == 0 {
		t.Errorf("expected non-zero duration in download status, got %v", st.Duration)
	}
	st.Duration = 0
	if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: expected}); !reflect.DeepEqual(got, want) {
		t.Errorf("expected status %v, got %v", want, got)
	}

	validateCacheFile(t, cacheFile, cacheSize)
	validateIndexFile(t, indexFile, cacheSize, blockSize)
	if !cache.Complete() {
		t.Errorf("cache is not complete, expected complete cache")
	}

	// Make sure cache is used.
	dl = largefile.NewCachingDownloader(&mockLargeFile{size: cacheSize, blockSize: blockSize},
		cache,
		largefile.WithDownloadRateController(&jitterRateLimiter{}),
		largefile.WithDownloadConcurrency(concurrency))
	st, err = dl.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if st.Duration == 0 {
		t.Errorf("expected non-zero duration in download status, got %v", st.Duration)
	}
	st.Duration = 0
	cachedState := largefile.DownloadState{
		CachedBytes:    cacheSize,
		CachedBlocks:   nBlocks,
		DownloadSize:   cacheSize,
		DownloadBlocks: nBlocks,
	}
	if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: cachedState}); !reflect.DeepEqual(got, want) {
		t.Errorf("expected status %v, got %v", want, got)
	}

	validateCacheFile(t, cacheFile, cacheSize)

}

func TestCacheRestart(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	concurrency := 100
	cacheSize := int64(file.KB * 7)
	blockSize := 4 * 16 // Multiple of 4 to allow for writing uint32s to the test data

	if err := largefile.NewFilesForCache(ctx, cacheFile, indexFile, cacheSize, blockSize, concurrency); err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("failed to create and allocate space for %s: %v", cacheFile, err)
	}
	cSize, cBblockSize := cache.ContentLengthAndBlockSize()
	if cSize != cacheSize || cBblockSize != blockSize {
		t.Fatalf("cache content size %d and block size %d do not match expected values %d and %d", cSize, cBblockSize, cacheSize, blockSize)
	}

	t.Logf("Successfully created and allocated space for %s with size %v bytes in blocks of size %v", cacheFile, cSize, cBblockSize)

	prevState := largefile.DownloadState{
		DownloadSize:   cacheSize,
		DownloadBlocks: int64(largefile.NumBlocks(cacheSize, blockSize)),
	}
	for i, failRatio := range []int{9, 4, 0} {
		mf := &mockLargeFile{size: cacheSize, blockSize: blockSize, failRatio: failRatio}

		dl := largefile.NewCachingDownloader(mf,
			cache,
			largefile.WithDownloadRateController(&jitterRateLimiter{}),
			largefile.WithDownloadConcurrency(concurrency))

		st, err := dl.Run(ctx)
		t.Logf("Run %d completed with status: %+v\n", i, st)
		if err != nil {
			if got, want := st.Resumeable, true; got != want {
				t.Errorf("Run %d expected resumeable error, got %v", i, err)
			}
			if got, want := cache.Complete(), false; got != want {
				t.Errorf("Run %d expected cache to be incomplete, got complete: %v", i, got)
			}
		} else {
			if got, want := st.Complete, true; got != want {
				t.Errorf("Run %d expected complete status, got %v", i, st)
			}
			if got, want := cache.Complete(), true; got != want {
				t.Errorf("Run %d expected cache to be complete, got complete: %v", i, got)
			}
		}
		if err := cache.Close(); err != nil {
			t.Fatalf("failed to close cache: %v", err)
		}
		cache, err = largefile.NewLocalDownloadCache(cacheFile, indexFile)
		if err != nil {
			t.Fatalf("failed to create and allocate space for %s: %v", cacheFile, err)
		}
		if st.CachedBlocks <= prevState.CachedBlocks {
			t.Errorf("Run %d expected more cached blocks, got %d, want > %d", i, st.CachedBlocks, prevState.CachedBlocks)
		}
		if st.CachedBytes <= prevState.CachedBytes {
			t.Errorf("Run %d expected more cached bytes, got %d, want > %d", i, st.CachedBytes, prevState.CachedBytes)
		}
		prevState = st.DownloadState
	}

	if !cache.Complete() {
		t.Errorf("cache is not complete after retries, expected complete cache")
	}

	validateCacheFile(t, cacheFile, cacheSize)
	validateIndexFile(t, indexFile, cacheSize, blockSize)

}
