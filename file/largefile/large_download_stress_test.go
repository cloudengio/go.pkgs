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
	"strings"
	"testing"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/file/largefile"
	"cloudeng.io/net/ratecontrol"
)

type mockLargeFile struct {
	size      int64
	blockSize int
	failRatio int
	withRetry bool
}

func (m *mockLargeFile) Name() string {
	return "mockLargeFile" // Mock implementation, returns a fixed name
}

func (m *mockLargeFile) ContentLengthAndBlockSize(context.Context) (int64, int, error) {
	return m.size, m.blockSize, nil // Mock implementation, returns size and block size
}
func (m *mockLargeFile) Digest(context.Context) string {
	return ""
}

type retryResponse struct{}

func (r retryResponse) IsRetryable() bool {
	return true
}

func (r retryResponse) BackoffDuration() (bool, time.Duration) {
	return false, 0
}

func (m *mockLargeFile) GetReader(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	//nolint:gosec // G404
	if m.failRatio > 0 && rand.Intn(10) < m.failRatio {
		if m.withRetry {
			return nil, &retryResponse{}, fmt.Errorf("mock failure for testing")
		}
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

type noBackoff struct{ retries int }

func (nb noBackoff) Retries() int {
	return nb.retries
}

func (nb noBackoff) Wait(_ context.Context, _ any) (bool, error) {
	return false, nil
}

type jitterRateLimiter struct{ retries int }

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
	return &noBackoff{retries: j.retries}
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
	for r := range dr.AllSet(0) {
		e := largefile.ByteRange{
			From: int64(s * blockSize),
			To:   min(int64((s+1)*blockSize-1), cacheSize-1),
		}
		if got, want := r, e; !reflect.DeepEqual(got, want) {
			t.Errorf("NextSet(%d) = %v, want %v", s, got, want)
		}
		s++
	}
	for r := range dr.AllClear(0) {
		t.Errorf("NextClear(0) returned unexpected range %v", r)
	}
}

func TestCacheStressTest(t *testing.T) {
	ctx := context.Background()
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	for _, concurrency := range []int{1, 10, 100} {
		t.Run(fmt.Sprintf("concurrency=%d", concurrency), func(t *testing.T) {
			t.Logf("Running stress test with concurrency %d", concurrency)
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
				Iterations:       1,
			}
			if st.Duration == 0 {
				t.Errorf("expected non-zero duration in download status, got %v", st.Duration)
			}
			st.Duration = 0
			if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: expected}); !reflect.DeepEqual(got, want) {
				t.Errorf("expected status %+v, got %+v", want, got)
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
				Iterations:     1,
			}
			if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: cachedState}); !reflect.DeepEqual(got, want) {
				t.Errorf("expected status %+v, got %+v", want, got)
			}

			validateCacheFile(t, cacheFile, cacheSize)
		})
	}

}

func TestCacheRestart(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	concurrency := 10
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

	t.Logf("Successfully created and allocated space for %s with size %v bytes in blocks of size %v", cacheFile, cSize, cBblockSize)

	prevState := largefile.DownloadState{
		DownloadSize:   cacheSize,
		DownloadBlocks: int64(largefile.NumBlocks(cacheSize, blockSize)),
	}
	logger, logOut := newLogger()
	totalErrors := int64(0)
	totalRetries := int64(0)
	// Run the downloader multiple times with different failure ratios to ensure
	// partial downloads that will be progressively filled by rerunning the downloader.
	for i, failRatio := range []int{9, 4, 0} {
		mf := &mockLargeFile{size: cacheSize, blockSize: blockSize, failRatio: failRatio}

		dl := largefile.NewCachingDownloader(mf,
			cache,
			largefile.WithDownloadLogger(logger),
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
		if st.Iterations != 1 {
			t.Errorf("Run %d expected 1 iteration, got %d", i, st.Iterations)
		}
		totalErrors += st.DownloadErrors
		totalRetries += st.DownloadRetries
		prevState = st.DownloadState
	}

	if !cache.Complete() {
		t.Errorf("cache is not complete after retries, expected complete cache")
	}

	if got, want := totalRetries, int64(0); got < want {
		t.Errorf("expected at least %d retries, got %d", want, got)
	}

	validateCacheFile(t, cacheFile, cacheSize)
	validateIndexFile(t, indexFile, cacheSize, blockSize)

	if got, want := totalErrors, strings.Count(logOut.String(), `"error":"mock failure for testing"`); got != int64(want) {
		t.Errorf("got %d mock failure messages, did not match number of errors reported: %d", want, got)
	}

}

func downloadFile(ctx context.Context, t *testing.T, cacheSize int64, blockSize, failRatio int, withRetry bool, opts ...largefile.DownloadOption) largefile.DownloadStatus {
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	if err := largefile.NewFilesForCache(ctx, cacheFile, indexFile, cacheSize, blockSize, 2); err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("failed to create and allocate space for %s: %v", cacheFile, err)
	}

	mf := &mockLargeFile{size: cacheSize, blockSize: blockSize, failRatio: failRatio, withRetry: withRetry}

	dl := largefile.NewCachingDownloader(mf, cache, opts...)

	st, err := dl.Run(ctx)
	t.Logf("Run completed with status: %+v\n", st)
	if err != nil {
		if got, want := st.Resumeable, true; got != want {
			t.Errorf("Run expected resumeable error, got %v", err)
		}
		if got, want := cache.Complete(), false; got != want {
			t.Errorf("Run expected cache to be incomplete, got complete: %v", got)
		}
	} else {
		if got, want := st.Complete, true; got != want {
			t.Errorf("Run expected complete status, got %v", st)
		}
		if got, want := cache.Complete(), true; got != want {
			t.Errorf("Run expected cache to be complete, got complete: %v", got)
		}
	}
	if err := cache.Close(); err != nil {
		t.Fatalf("failed to close cache: %v", err)
	}

	if !cache.Complete() {
		t.Errorf("cache is not complete after retries, expected complete cache")
	}

	if err == nil {
		validateCacheFile(t, cacheFile, cacheSize)
		validateIndexFile(t, indexFile, cacheSize, blockSize)
	}

	return st
}

func TestRateControl(t *testing.T) {
	ctx := context.Background()
	cacheSize := int64(file.KB * 7)
	blockSize := 1024
	concurrency := 10
	// No rate control, should be faster than the slower download.
	st := downloadFile(ctx, t, cacheSize, blockSize, 0, false,
		largefile.WithDownloadRateController(ratecontrol.New(ratecontrol.WithNoRateControl())),
		largefile.WithDownloadConcurrency(concurrency))

	slower := ratecontrol.New(ratecontrol.WithBytesPerTick(time.Millisecond*100, 10))
	sst := downloadFile(ctx, t, cacheSize, blockSize, 0, false,
		largefile.WithDownloadRateController(slower),
		largefile.WithDownloadConcurrency(concurrency))

	if (2 * st.Duration) > sst.Duration {
		t.Errorf("expected faster download with no rate control, got %v vs %v", st.Duration, sst.Duration)
	}
	t.Logf("no rate control download duration: %v, slower download duration: %v\n", st.Duration, sst.Duration)
}

func TestCacheRetriesAndRunToCompletion(t *testing.T) {
	ctx := context.Background()

	cacheSize := int64(file.KB * 7)
	blockSize := 1024
	concurrency := 10

	logger, logOut := newLogger()

	// Test run to completion with no retries.
	st := downloadFile(ctx, t, cacheSize, blockSize, 7, false,
		largefile.WithDownloadRateController(ratecontrol.New(ratecontrol.WithNoRateControl())),
		largefile.WithDownloadConcurrency(concurrency),
		largefile.WithDownloadWaitForCompletion(true),
		largefile.WithDownloadLogger(logger),
	)
	if got, want := st.Iterations, strings.Count(logOut.String(), "download not complete")+1; got != int64(want) {
		t.Errorf("expected %d iterations, got %d", want, got)
	}

	logOut.Reset() // Reset log output for the next test

	// Test a single iteration with many retries.
	rl := &jitterRateLimiter{retries: 1000}
	st = downloadFile(ctx, t, cacheSize, blockSize, 7, true,
		largefile.WithDownloadRateController(rl),
		largefile.WithDownloadConcurrency(concurrency),
		largefile.WithDownloadLogger(logger),
	)
	if st.Iterations != 1 {
		t.Errorf("expected 1 iteration with retries, got %d", st.Iterations)
	}
	if st.DownloadErrors != 0 {
		t.Errorf("expected 0 download errors with retries, got %d", st.DownloadErrors)
	}
	if st.DownloadRetries == 0 {
		t.Errorf("expected non-zero download retries, got %d", st.DownloadRetries)
	}

	if got, want := 0, len(logOut.String()); got != want {
		t.Errorf("expected no log messages, got %d", want)
	}

	// Test a download with run to completion that will always fail, it will
	// hang until the context is cancelled
	tmpDirAllCached := t.TempDir()
	cacheFile := filepath.Join(tmpDirAllCached, "cache.dat")
	indexFile := filepath.Join(tmpDirAllCached, "cache.idx")

	if err := largefile.NewFilesForCache(ctx, cacheFile, indexFile, cacheSize, blockSize, 2); err != nil {
		t.Fatalf("NewFilesForCache failed: %v", err)
	}
	cache, err := largefile.NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("failed to create and allocate space for %s: %v", cacheFile, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	mf := &mockLargeFile{size: cacheSize, blockSize: blockSize, failRatio: 10} // all retries will fail

	dl := largefile.NewCachingDownloader(mf, cache,
		largefile.WithDownloadConcurrency(concurrency),
		largefile.WithDownloadLogger(logger),
		largefile.WithDownloadWaitForCompletion(true))

	st, err = dl.Run(ctx)
	if err == nil {
		t.Fatalf("expected error due to context cancellation, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded error, got %v", err)
	}
}
