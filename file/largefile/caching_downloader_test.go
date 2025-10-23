// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
	"cloudeng.io/net/ratecontrol"
)

type noRetryResponse struct{}

func (r noRetryResponse) IsRetryable() bool {
	return false
}

func (r noRetryResponse) BackoffDuration() (bool, time.Duration) {
	return false, 0
}

// GenAI: gemini 2.5 wrote these tests, with some errors and a massive number of
// lint errors.

// MockReader implements largefile.Reader
type MockReader struct {
	ContentLengthAndBlockSizeFunc func() (size int64, blockSize int)
	GetReaderFunc                 func(ctx context.Context, from, to int64) (rd io.ReadCloser, retry largefile.RetryResponse, err error)
	GetReaderCalls                []struct{ From, To int64 }
	mu                            sync.Mutex
}

func (m *MockReader) Name() string {
	return "MockReader"
}

func (m *MockReader) ContentLengthAndBlockSize() (size int64, blockSize int) {
	if m.ContentLengthAndBlockSizeFunc != nil {
		return m.ContentLengthAndBlockSizeFunc()
	}
	return 0, 0
}

func (m *MockReader) GetReader(ctx context.Context, from, to int64) (rd io.ReadCloser, retry largefile.RetryResponse, err error) {
	m.mu.Lock()
	m.GetReaderCalls = append(m.GetReaderCalls, struct{ From, To int64 }{from, to})
	m.mu.Unlock()
	if m.GetReaderFunc != nil {
		return m.GetReaderFunc(ctx, from, to)
	}
	return nil, &noRetryResponse{}, errors.New("MockReader.GetReaderFunc not implemented")
}

func (m *MockReader) Digest() digests.Hash {
	// Default implementation returns no digest
	return digests.Hash{}
}

// ResetCalls resets the GetReaderCalls slice to allow for fresh tests

func (m *MockReader) ResetCalls() {
	m.mu.Lock()
	m.GetReaderCalls = nil
	m.mu.Unlock()
}

// MockDownloadCache implements largefile.DownloadCache
type MockDownloadCache struct {
	ContentLengthAndBlockSizeFunc func() (size int64, blockSize int)
	OutstandingFunc               func(int, *largefile.ByteRange) int
	CompleteFunc                  func() bool
	PutFunc                       func(data []byte, off int64) (int, error)
	TailFunc                      func(context.Context) largefile.ByteRange // Not used by CachingDownlun

	GetFunc    func(data []byte, off int64) (int, error) // Not used by CachingDownloader.Run
	CachedFunc func(int, *largefile.ByteRange) int       // Not used by CachingDownloader.Run
	PutCalls   []struct {
		Data []byte
		Off  int64
	}
	mu sync.Mutex
}

func (m *MockDownloadCache) ContentLengthAndBlockSize() (size int64, blockSize int) {
	if m.ContentLengthAndBlockSizeFunc != nil {
		return m.ContentLengthAndBlockSizeFunc()
	}
	return 0, 0
}

func (m *MockDownloadCache) CachedBytesAndBlocks() (int64, int64) {
	// Default implementation returns no cached bytes or blocks
	return 0, 0
}

func (m *MockDownloadCache) NextOutstanding(s int, b *largefile.ByteRange) int {
	if m.OutstandingFunc != nil {
		return m.OutstandingFunc(s, b)
	}
	return -1
}

func (m *MockDownloadCache) NextCached(s int, b *largefile.ByteRange) int {
	if m.CachedFunc != nil {
		return m.CachedFunc(s, b)
	}
	return -1
}

func (m *MockDownloadCache) WriteAt(data []byte, off int64) (int, error) {
	m.mu.Lock()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.PutCalls = append(m.PutCalls, struct {
		Data []byte
		Off  int64
	}{dataCopy, off})
	m.mu.Unlock()
	if m.PutFunc != nil {
		return m.PutFunc(data, off)
	}
	return len(data), nil
}

func (m *MockDownloadCache) ReadAt(data []byte, off int64) (int, error) {
	if m.GetFunc != nil {
		return m.GetFunc(data, off)
	}
	return 0, errors.New("MockDownloadCache.GetFunc not implemented")
}

func (m *MockDownloadCache) Complete() bool {
	if m.CompleteFunc != nil {
		return m.CompleteFunc()
	}
	return false
}

func (m *MockDownloadCache) ResetCalls() {
	m.mu.Lock()
	m.PutCalls = nil
	m.mu.Unlock()
}

func (m *MockDownloadCache) Tail(ctx context.Context) largefile.ByteRange {
	if m.TailFunc != nil {
		return m.TailFunc(ctx)
	}
	return largefile.ByteRange{}
}

const (
	defaultContentSize = int64(256)
	defaultBlockSize   = 64
	defaultConcurrency = 1
)

var (
	firstRange = largefile.ByteRange{From: 0, To: int64(defaultBlockSize) - 1}
)

func newDefaultMockReader() *MockReader {
	return &MockReader{
		ContentLengthAndBlockSizeFunc: func() (size int64, blockSize int) {
			return defaultContentSize, defaultBlockSize
		},
		GetReaderFunc: func(_ context.Context, from, to int64) (rd io.ReadCloser, retry largefile.RetryResponse, err error) {
			dataSize := int(to-from) + 1
			d := make([]byte, dataSize)
			for i := 0; i < dataSize; i++ {
				d[i] = byte(from + int64(i)) // Unique data per block start
			}
			return io.NopCloser(bytes.NewReader(d)), noRetryResponse{}, nil
		},
	}
}

func newDefaultMockCache() *MockDownloadCache {
	return &MockDownloadCache{
		ContentLengthAndBlockSizeFunc: func() (size int64, blockSize int) {
			return defaultContentSize, defaultBlockSize
		},
		PutFunc: func(data []byte, _ int64) (int, error) { return len(data), nil },
		OutstandingFunc: func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq()(s, b) // Default to no outstanding blocks
		},
	}
}

func defaultOpts(c int) []largefile.DownloadOption {
	return []largefile.DownloadOption{
		largefile.WithDownloadConcurrency(c),
		largefile.WithDownloadRateController(ratecontrol.New()),
		largefile.WithDownloadLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}
}

func newByteRangeSeq(ranges ...largefile.ByteRange) func(s int, b *largefile.ByteRange) int {
	return func(s int, b *largefile.ByteRange) int {
		if s >= len(ranges) {
			return -1
		}
		*b = ranges[s]
		return s + 1
	}
}

type slowRateLimiter struct{}

func (s *slowRateLimiter) Wait(ctx context.Context) error {
	select {
	case <-time.After(time.Minute): // Simulate a slow rate limiter
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *slowRateLimiter) BytesTransferred(int) {
	// No-op for this mock, as we don't track bytes transferred.
}

func (s *slowRateLimiter) Backoff() ratecontrol.Backoff {
	return &noBackoff{}
}

func newLogger() (*slog.Logger, *strings.Builder) {
	out := &strings.Builder{}
	return slog.New(slog.NewJSONHandler(out, nil)), out
}

func TestCachingDownloaderSetup(t *testing.T) {
	t.Run("cache not set", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		_, err := largefile.NewCachingDownloader(mockReader, nil, defaultOpts(defaultConcurrency)...)
		if err == nil {
			t.Fatal("expected error when cache is not set, got nil")
		}
		if !strings.Contains(err.Error(), "cache is not set") {
			t.Errorf("expected error message to contain 'cache is not set', got %q", err.Error())
		}
	})

	t.Run("cache size mismatch", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.ContentLengthAndBlockSizeFunc = func() (size int64, blockSize int) {
			return defaultContentSize + 10, defaultBlockSize
		}
		_, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		if err == nil {
			t.Fatal("expected error for cache size mismatch, got nil")
		}
		if !strings.Contains(err.Error(), "cache size") || !strings.Contains(err.Error(), "does not match file size") {
			t.Errorf("expected error message for size mismatch, got %q", err.Error())
		}
	})
}

func defaultsForTest() (defaultBlocks int, defaultIncompleteStats largefile.DownloadStats) {
	defaultBlocks = largefile.NumBlocks(defaultContentSize, defaultBlockSize)
	defaultIncompleteStats = largefile.DownloadStats{
		CachedOrStreamedBytes:  0,
		CachedOrStreamedBlocks: 0,
		DownloadedBytes:        0,
		DownloadedBlocks:       0,
		DownloadSize:           defaultContentSize,
		DownloadBlocks:         int64(defaultBlocks),
		Iterations:             1,
	}
	return
}

func TestCachingDownloaderSimple(t *testing.T) {
	ctx := t.Context()
	defaultBlocks, defaultIncompleteStats := defaultsForTest()

	t.Run("no outstanding blocks", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache() // Default OutstandingFunc returns empty seq
		mockCache.CompleteFunc = func() bool {
			return true
		}
		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed for no outstanding blocks: %v", err)
		}
		if len(mockReader.GetReaderCalls) != 0 {
			t.Errorf("expected 0 GetReaderCalls, got %d", len(mockReader.GetReaderCalls))
		}
		if len(mockCache.PutCalls) != 0 {
			t.Errorf("expected 0 PutCalls, got %d", len(mockCache.PutCalls))
		}
		st.Duration = 0
		p := defaultIncompleteStats
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("one outstanding block - success", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		outstandingRange := firstRange
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(outstandingRange)(s, b)
		}
		mockCache.CompleteFunc = func() bool {
			return true
		}

		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed for one block: %v", err)
		}
		if len(mockReader.GetReaderCalls) != 1 {
			t.Errorf("expected 1 GetReaderCall, got %d", len(mockReader.GetReaderCalls))
		} else if !reflect.DeepEqual(mockReader.GetReaderCalls[0], struct{ From, To int64 }{outstandingRange.From, outstandingRange.To}) {
			t.Errorf("GetReaderCall mismatch: got %+v, want %+v", mockReader.GetReaderCalls[0], outstandingRange)
		}
		if len(mockCache.PutCalls) != 1 {
			t.Errorf("expected 1 PutCall, got %d", len(mockCache.PutCalls))
		} else if mockCache.PutCalls[0].Off != outstandingRange.From {
			t.Errorf("PutCall range mismatch: got %+v, want %+v", mockCache.PutCalls[0].Off, outstandingRange.From)
		}
		expectedData := make([]byte, defaultBlockSize)
		for i := 0; i < defaultBlockSize; i++ {
			expectedData[i] = byte(outstandingRange.From + int64(i))
		}
		if !bytes.Equal(mockCache.PutCalls[0].Data, expectedData) {
			t.Errorf("PutCall data mismatch: got %x, want %x", mockCache.PutCalls[0].Data, expectedData)
		}
		st.Duration = 0
		p := largefile.DownloadStats{
			CachedOrStreamedBytes:  int64(defaultBlockSize),
			CachedOrStreamedBlocks: 1,
			DownloadedBytes:        int64(defaultBlockSize),
			DownloadedBlocks:       1,
			DownloadSize:           defaultContentSize,
			DownloadBlocks:         int64(defaultBlocks),
			Iterations:             1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

}

func TestCachingDownloaderOutstandingBlocks(t *testing.T) {
	ctx := t.Context()

	t.Run("multiple outstanding blocks success (concurrency > 1)", func(t *testing.T) {
		numBlocks := 3
		concurrency := 2
		currentContentSize := int64(numBlocks * defaultBlockSize)

		mockReader := &MockReader{
			ContentLengthAndBlockSizeFunc: func() (int64, int) {
				return currentContentSize, defaultBlockSize
			},
			GetReaderFunc: func(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
				dataSize := int(to - from + 1)
				d := make([]byte, dataSize)
				for i := 0; i < dataSize; i++ {
					d[i] = byte(from + int64(i)) // Unique data per block
				}
				return io.NopCloser(bytes.NewReader(d)), noRetryResponse{}, nil
			},
		}
		mockCache := &MockDownloadCache{
			ContentLengthAndBlockSizeFunc: func() (int64, int) {
				return currentContentSize, defaultBlockSize
			},
			PutFunc: func(data []byte, _ int64) (int, error) { return len(data), nil },
		}
		mockCache.CompleteFunc = func() bool {
			return true
		}

		expectedRanges := make([]largefile.ByteRange, numBlocks)
		for i := range numBlocks {
			expectedRanges[i] = largefile.ByteRange{
				From: int64(i * defaultBlockSize),
				To:   int64((i+1)*defaultBlockSize) - 1,
			}
		}
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(expectedRanges...)(s, b)
		}

		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(concurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed for multiple blocks: %v", err)
		}

		if len(mockReader.GetReaderCalls) != numBlocks {
			t.Errorf("expected %d GetReaderCalls, got %d", numBlocks, len(mockReader.GetReaderCalls))
		}
		if len(mockCache.PutCalls) != numBlocks {
			t.Errorf("expected %d PutCalls, got %d", numBlocks, len(mockCache.PutCalls))
		}

		// Sort PutCalls by From offset for deterministic checking
		sort.Slice(mockCache.PutCalls, func(i, j int) bool {
			return mockCache.PutCalls[i].Off < mockCache.PutCalls[j].Off
		})

		for i, expectedRange := range expectedRanges {
			if i >= len(mockCache.PutCalls) {
				t.Errorf("missing PutCall for range index %d: %+v", i, expectedRange)
				continue
			}
			putCall := mockCache.PutCalls[i]
			if putCall.Off != expectedRange.From {
				t.Errorf("PutCall range mismatch at index %d: got %+v, want %+v", i, putCall.Off, expectedRange)
			}
			expectedDataSize := expectedRange.To - expectedRange.From + 1
			if int64(len(putCall.Data)) != expectedDataSize {
				t.Errorf("PutCall data length mismatch for range %+v: got %d, want %d", expectedRange, len(putCall.Data), expectedDataSize)
			}
		}
		st.Duration = 0
		p := largefile.DownloadStats{
			CachedOrStreamedBytes:  currentContentSize,
			CachedOrStreamedBlocks: int64(numBlocks),
			DownloadedBytes:        currentContentSize,
			DownloadedBlocks:       int64(numBlocks),
			DownloadSize:           currentContentSize,
			DownloadBlocks:         int64(numBlocks),
			Iterations:             1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

}

func TestCachingDownloaderMultipleBlockSizes(t *testing.T) {
	ctx := t.Context()

	numFullBlocks := 2
	partialBlockSize := defaultBlockSize / 2
	currentContentSize := int64(numFullBlocks*defaultBlockSize + partialBlockSize)
	totalBlocks := numFullBlocks + 1
	concurrency := 2

	mockReader := &MockReader{
		ContentLengthAndBlockSizeFunc: func() (int64, int) {
			return currentContentSize, defaultBlockSize
		},
		GetReaderFunc: func(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
			dataSize := int(to - from + 1)
			d := make([]byte, dataSize)
			for i := 0; i < dataSize; i++ {
				d[i] = byte(from + int64(i))
			}
			return io.NopCloser(bytes.NewReader(d)), noRetryResponse{}, nil
		},
	}
	mockCache := &MockDownloadCache{
		ContentLengthAndBlockSizeFunc: func() (int64, int) {
			return currentContentSize, defaultBlockSize
		},
		PutFunc: func(data []byte, _ int64) (int, error) { return len(data), nil },
	}
	mockCache.CompleteFunc = func() bool {
		return true
	}

	expectedRanges := make([]largefile.ByteRange, totalBlocks)
	for i := range numFullBlocks {
		expectedRanges[i] = largefile.ByteRange{
			From: int64(i * defaultBlockSize),
			To:   int64((i+1)*defaultBlockSize) - 1,
		}
	}
	expectedRanges[numFullBlocks] = largefile.ByteRange{
		From: int64(numFullBlocks * defaultBlockSize),
		To:   currentContentSize - 1, // Last byte of the file
	}

	mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
		return newByteRangeSeq(expectedRanges...)(s, b)
	}

	dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(concurrency)...)
	if err != nil {
		t.Fatalf("NewCachingDownloader failed: %v", err)
	}
	st, err := dl.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed for non-multiple block size file: %v", err)
	}

	if len(mockReader.GetReaderCalls) != totalBlocks {
		t.Errorf("expected %d GetReaderCalls, got %d", totalBlocks, len(mockReader.GetReaderCalls))
	}
	if len(mockCache.PutCalls) != totalBlocks {
		t.Errorf("expected %d PutCalls, got %d", totalBlocks, len(mockCache.PutCalls))
	}

	sort.Slice(mockCache.PutCalls, func(i, j int) bool {
		return mockCache.PutCalls[i].Off < mockCache.PutCalls[j].Off
	})

	for i, expectedRange := range expectedRanges {
		if i >= len(mockCache.PutCalls) {
			t.Errorf("missing PutCall for range index %d: %+v", i, expectedRange)
			continue
		}
		putCall := mockCache.PutCalls[i]
		if putCall.Off != expectedRange.From {
			t.Errorf("PutCall range mismatch at index %d: got %+v, want %+v", i, putCall.Off, expectedRange)
		}
		expectedDataSize := expectedRange.To - expectedRange.From + 1
		if int64(len(putCall.Data)) != expectedDataSize {
			t.Errorf("PutCall data length mismatch for range %+v: got %d, want %d", expectedRange, len(putCall.Data), expectedDataSize)
		}
		// Verify data content for the last partial block specifically
		if i == numFullBlocks { // Last block
			expectedPartialData := make([]byte, partialBlockSize)
			for j := 0; j < partialBlockSize; j++ {
				expectedPartialData[j] = byte(expectedRange.From + int64(j))
			}
			if !bytes.Equal(putCall.Data, expectedPartialData) {
				t.Errorf("PutCall data content mismatch for partial block: got %x, want %x", putCall.Data, expectedPartialData)
			}
		}
	}
	st.Duration = 0
	p := largefile.DownloadStats{
		CachedOrStreamedBytes:  currentContentSize,
		CachedOrStreamedBlocks: int64(totalBlocks),
		DownloadedBytes:        currentContentSize,
		DownloadedBlocks:       int64(totalBlocks),
		DownloadSize:           currentContentSize,
		DownloadBlocks:         int64(totalBlocks),
		Iterations:             1,
	}
	if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
		t.Errorf("expected status %v, got %v", want, got)
	}
}

func TestCachingDownloaderConcurrencyAndCancellation(t *testing.T) {
	ctx := context.Background()
	defaultBlocks := largefile.NumBlocks(defaultContentSize, defaultBlockSize)
	defaultIncompleteStats := largefile.DownloadStats{
		CachedOrStreamedBytes:  0,
		CachedOrStreamedBlocks: 0,
		DownloadedBytes:        0,
		DownloadedBlocks:       0,
		DownloadSize:           defaultContentSize,
		DownloadBlocks:         int64(defaultBlocks),
		Iterations:             1,
	}

	t.Run("concurrency 0 - no outstanding", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache() // Default is no outstanding
		mockCache.CompleteFunc = func() bool {
			return true
		}
		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(0)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run with concurrency 0 and no outstanding blocks failed: %v", err)
		}
		if len(mockReader.GetReaderCalls) != 0 {
			t.Errorf("expected 0 GetReaderCalls, got %d", len(mockReader.GetReaderCalls))
		}
		st.Duration = 0
		p := defaultIncompleteStats
		if got, want := st, (largefile.DownloadStatus{Resumable: false, Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("concurrency 0 - with outstanding (should block or complete if ctx cancelled)", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		dl, err := largefile.NewCachingDownloader(mockReader, mockCache,
			largefile.WithDownloadRateController(&slowRateLimiter{}))
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		runCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		st, err := dl.Run(runCtx)
		// Expect context deadline exceeded because generator sends to requestCh,
		// but no fetchers are there to receive, so it blocks until context times out.
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded with concurrency 0 and outstanding blocks, got %v", err)
		}
		st.Duration = 0
		st.CachedOrStreamedBlocks = 0
		st.CachedOrStreamedBytes = 0
		st.DownloadedBlocks = 0
		st.DownloadedBytes = 0
		p := defaultIncompleteStats
		if got, want := st, (largefile.DownloadStatus{Resumable: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	t.Run("generator context cancelled", func(t *testing.T) {
		mockReader := newDefaultMockReader() // Use appropriate mock setup
		mockCache := newDefaultMockCache()
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		currentConcurrency := 2
		numRanges := currentConcurrency * 2
		ranges := make([]largefile.ByteRange, numRanges)
		for i := 0; i < numRanges; i++ {
			ranges[i] = largefile.ByteRange{From: int64(i * defaultBlockSize), To: int64((i+1)*defaultBlockSize) - 1} // Corrected To
		}
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(ranges...)(s, b)
		}
		// ... (rest of this test case as you have it) ...
		// Ensure mockReader.ContentLengthAndBlockSizeFunc and mockCache.ContentLengthAndBlockSizeFunc
		// are set to match defaultContentSize (or a size that accommodates numRanges) and defaultBlockSize.
		mockReader.ContentLengthAndBlockSizeFunc = func() (size int64, blockSize int) {
			return int64(numRanges * defaultBlockSize), defaultBlockSize
		}
		mockCache.ContentLengthAndBlockSizeFunc = func() (size int64, blockSize int) {
			return int64(numRanges * defaultBlockSize), defaultBlockSize
		}

		getBlocked := make(chan struct{})
		mockReader.GetReaderFunc = func(c context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
			expectedDataSize := to - from + 1
			select {
			case <-getBlocked:
				return io.NopCloser(bytes.NewReader(make([]byte, expectedDataSize))), noRetryResponse{}, nil
			case <-c.Done():
				return nil, noRetryResponse{}, c.Err()
			}
		}

		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(currentConcurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
			close(getBlocked)
		}()

		st, err := dl.Run(cancelCtx)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
		if st.DownloadErrors == 0 {
			t.Error("expected DownloadErrors to be > 0, got 0")
		}
		st.DownloadErrors = 0
		st.Duration = 0
		// The values vary depending on when the context was cancelled,
		// so we set them to 0.
		st.DownloadedBlocks, st.DownloadedBytes = 0, 0
		st.CachedOrStreamedBlocks, st.CachedOrStreamedBytes = 0, 0
		p := defaultIncompleteStats
		if got, want := st, (largefile.DownloadStatus{Resumable: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})
}

func TestCachingDownloaderErrorHandling(t *testing.T) {
	ctx := context.Background()
	defaultBlocks := largefile.NumBlocks(defaultContentSize, defaultBlockSize)
	defaultIncompleteStats := largefile.DownloadStats{
		CachedOrStreamedBytes:  0,
		CachedOrStreamedBlocks: 0,
		DownloadedBytes:        0,
		DownloadedBlocks:       0,
		DownloadSize:           defaultContentSize,
		DownloadBlocks:         int64(defaultBlocks),
		Iterations:             1,
	}

	t.Run("fetcher GetReader error", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		fetchErr := errors.New("fetch failed")
		mockReader.GetReaderFunc = func(_ context.Context, _, _ int64) (rd io.ReadCloser, retry largefile.RetryResponse, err error) {
			return nil, &noRetryResponse{}, fetchErr
		}
		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error from fetcher, got nil")
		}
		if !strings.Contains(err.Error(), fetchErr.Error()) {
			t.Errorf("expected error to contain %q, got %q", fetchErr.Error(), err.Error())
		}
		st.Duration = 0
		if st.DownloadErrors == 0 {
			t.Error("expected DownloadErrors to be > 0, got 0")
		}
		st.DownloadErrors = 0
		p := defaultIncompleteStats
		if got, want := st, (largefile.DownloadStatus{Resumable: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	t.Run("fetcher handleResponse (cache.Put) error", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		putErr := errors.New("cache Put failed")
		mockCache.PutFunc = func(_ []byte, _ int64) (int, error) {
			return 0, putErr
		}
		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error from cache.Put, got nil")
		}
		if !strings.Contains(err.Error(), putErr.Error()) {
			t.Errorf("expected error to contain %q, got %q", putErr.Error(), err.Error())
		}
		st.Duration = 0
		p := defaultIncompleteStats
		if st.CacheErrors == 0 {
			t.Error("expected DownloadErrors to be > 0, got 0")
		}
		st.CacheErrors = 0
		p.DownloadedBlocks = 1
		p.DownloadedBytes = int64(defaultBlockSize)
		if got, want := st, (largefile.DownloadStatus{DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})
}

func TestCachingDownloaderProgress(t *testing.T) {
	ctx := context.Background()
	t.Run("all blocks downloaded successfully with progress", func(t *testing.T) {
		mockReader := newDefaultMockReader() // Ensure this is configured for the test's content/block size
		mockCache := newDefaultMockCache()
		mockCache.CompleteFunc = func() bool {
			return true
		}

		numBlocks := 4
		currentContentSize := int64(numBlocks * defaultBlockSize)
		mockReader.ContentLengthAndBlockSizeFunc = func() (int64, int) {
			return currentContentSize, defaultBlockSize
		}
		mockCache.ContentLengthAndBlockSizeFunc = func() (int64, int) {
			return currentContentSize, defaultBlockSize
		}

		ranges := make([]largefile.ByteRange, numBlocks)
		for i := range numBlocks {
			// Corrected To
			ranges[i] = largefile.ByteRange{From: int64(i * defaultBlockSize), To: int64((i+1)*defaultBlockSize) - 1}
		}
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(ranges...)(s, b)
		}
		// ... (rest of this test case as you have it) ...
		progressCh := make(chan largefile.DownloadStats, numBlocks*2)
		optsWithProgress := defaultOpts(2) // Concurrency 2
		optsWithProgress = append(optsWithProgress, largefile.WithDownloadProgress(progressCh))

		ch := make(chan struct{})
		var progressUpdates []largefile.DownloadStats
		go func() {
			for p := range progressCh {
				progressUpdates = append(progressUpdates, p)
			}
			close(ch)
		}()

		dl, err := largefile.NewCachingDownloader(mockReader, mockCache, optsWithProgress...)
		if err != nil {
			t.Fatalf("NewCachingDownloader failed: %v", err)
		}
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		<-ch
		if len(progressUpdates) == 0 {
			t.Fatal("expected progress updates, got none")
		}

		sort.Slice(mockCache.PutCalls, func(i, j int) bool {
			return mockCache.PutCalls[i].Off < mockCache.PutCalls[j].Off
		})

		if len(mockCache.PutCalls) != numBlocks {
			t.Errorf("expected %d PutCalls, got %d", numBlocks, len(mockCache.PutCalls))
		}
		for i, r := range ranges {
			if i < len(mockCache.PutCalls) && mockCache.PutCalls[i].Off != r.From {
				t.Errorf("PutCall %d range mismatch: got %v, want %v", i, mockCache.PutCalls[i].Off, r.From)
			}
		}

		finalProgress := progressUpdates[len(progressUpdates)-1]
		if finalProgress.DownloadedBytes != currentContentSize {
			t.Errorf("final progress DownloadedBytes: got %d, want %d", finalProgress.DownloadedBytes, currentContentSize)
		}
		if finalProgress.DownloadedBlocks != int64(numBlocks) {
			t.Errorf("final progress DownloadedBlocks: got %d, want %d", finalProgress.DownloadedBlocks, int64(numBlocks))
		}
		if finalProgress.DownloadSize != currentContentSize {
			t.Errorf("final progress DownloadSize: got %d, want %d", finalProgress.DownloadSize, currentContentSize)
		}
		if finalProgress.DownloadBlocks != int64(numBlocks) {
			t.Errorf("final progress DownloadBlocks: got %d, want %d", finalProgress.DownloadBlocks, int64(numBlocks))
		}
		st.Duration = 0
		p := largefile.DownloadStats{
			CachedOrStreamedBytes:  currentContentSize,
			CachedOrStreamedBlocks: int64(numBlocks),
			DownloadedBytes:        currentContentSize,
			DownloadedBlocks:       int64(numBlocks),
			DownloadSize:           currentContentSize,
			DownloadBlocks:         int64(numBlocks),
			Iterations:             1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadStats: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})
}
