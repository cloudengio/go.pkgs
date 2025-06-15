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
	ContentLengthAndBlockSizeFunc func(ctx context.Context) (size int64, blockSize int, err error)
	GetReaderFunc                 func(ctx context.Context, from, to int64) (rd io.ReadCloser, retry largefile.RetryResponse, err error)
	GetReaderCalls                []struct{ From, To int64 }
	mu                            sync.Mutex
}

func (m *MockReader) Name() string {
	return "MockReader"
}

func (m *MockReader) ContentLengthAndBlockSize(ctx context.Context) (size int64, blockSize int, err error) {
	if m.ContentLengthAndBlockSizeFunc != nil {
		return m.ContentLengthAndBlockSizeFunc(ctx)
	}
	return 0, 0, errors.New("MockReader.ContentLengthAndBlockSizeFunc not implemented")
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

func (m *MockReader) Checksum(context.Context) (largefile.ChecksumType, string, error) {
	// Default implementation returns no checksum
	return largefile.NoChecksum, "", nil
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
	PutFunc                       func(r largefile.ByteRange, data []byte) error
	GetFunc                       func(r largefile.ByteRange, data []byte) error // Not used by CachingDownloader.Run
	CachedFunc                    func(int, *largefile.ByteRange) int            // Not used by CachingDownloader.Run
	PutCalls                      []struct {
		Range largefile.ByteRange
		Data  []byte
	}
	mu sync.Mutex
}

func (m *MockDownloadCache) ContentLengthAndBlockSize() (size int64, blockSize int) {
	if m.ContentLengthAndBlockSizeFunc != nil {
		return m.ContentLengthAndBlockSizeFunc()
	}
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

func (m *MockDownloadCache) Put(r largefile.ByteRange, data []byte) error {
	m.mu.Lock()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.PutCalls = append(m.PutCalls, struct {
		Range largefile.ByteRange
		Data  []byte
	}{r, dataCopy})
	m.mu.Unlock()
	if m.PutFunc != nil {
		return m.PutFunc(r, data)
	}
	return nil
}

func (m *MockDownloadCache) Get(r largefile.ByteRange, data []byte) error {
	if m.GetFunc != nil {
		return m.GetFunc(r, data)
	}
	return errors.New("MockDownloadCache.GetFunc not implemented")
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

func TestCachingDownloaderRun(t *testing.T) { //nolint:gocyclo
	ctx := context.Background()
	defaultContentSize := int64(256)
	defaultBlockSize := 64
	defaultBlocks := largefile.NumBlocks(defaultContentSize, defaultBlockSize)
	defaultState := largefile.DownloadState{
		CachedBytes:      defaultContentSize,
		CachedBlocks:     int64(defaultBlocks),
		DownloadedBytes:  defaultContentSize,
		DownloadedBlocks: int64(defaultBlocks),
		DownloadSize:     defaultContentSize,
		DownloadBlocks:   int64(defaultBlocks),
		Iterations:       1,
	}
	defaultIncompleteState := largefile.DownloadState{
		CachedBytes:      0,
		CachedBlocks:     0,
		DownloadedBytes:  0,
		DownloadedBlocks: 0,
		DownloadSize:     defaultContentSize,
		DownloadBlocks:   int64(defaultBlocks),
		Iterations:       1,
	}
	defaultConcurrency := 1

	firstRange := largefile.ByteRange{From: 0, To: int64(defaultBlockSize) - 1}

	newDefaultMockReader := func() *MockReader {
		return &MockReader{
			ContentLengthAndBlockSizeFunc: func(context.Context) (size int64, blockSize int, err error) {
				return defaultContentSize, defaultBlockSize, nil
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

	newDefaultMockCache := func() *MockDownloadCache {
		return &MockDownloadCache{
			ContentLengthAndBlockSizeFunc: func() (size int64, blockSize int) {
				return defaultContentSize, defaultBlockSize
			},
			PutFunc: func(largefile.ByteRange, []byte) error { return nil },
			OutstandingFunc: func(s int, b *largefile.ByteRange) int {
				return newByteRangeSeq()(s, b) // Default to no outstanding blocks
			},
		}
	}

	defaultOpts := func(c int) []largefile.DownloadOption {
		return []largefile.DownloadOption{
			largefile.WithDownloadConcurrency(c),
			largefile.WithDownloadRateController(ratecontrol.New()),
			largefile.WithDownloadLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		}
	}

	t.Run("cache not set", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		dl := largefile.NewCachingDownloader(mockReader, nil, defaultOpts(defaultConcurrency)...)
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error when cache is not set, got nil")
		}
		if !strings.Contains(err.Error(), "cache is not set") {
			t.Errorf("expected error message to contain 'cache is not set', got %q", err.Error())
		}
		if got, want := st, (largefile.DownloadStatus{}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("init fails - reader ContentLengthAndBlockSize error", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockReader.ContentLengthAndBlockSizeFunc = func(context.Context) (size int64, blockSize int, err error) {
			return 0, 0, errors.New("reader init error")
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error from init, got nil")
		}
		if !strings.Contains(err.Error(), "reader init error") {
			t.Errorf("expected error message to contain 'reader init error', got %q", err.Error())
		}
		if got, want := st, (largefile.DownloadStatus{}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("cache size mismatch", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.ContentLengthAndBlockSizeFunc = func() (size int64, blockSize int) {
			return defaultContentSize + 10, defaultBlockSize
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error for cache size mismatch, got nil")
		}
		if !strings.Contains(err.Error(), "cache size") || !strings.Contains(err.Error(), "does not match file size") {
			t.Errorf("expected error message for size mismatch, got %q", err.Error())
		}
		if got, want := st, (largefile.DownloadStatus{}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("no outstanding blocks", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache() // Default OutstandingFunc returns empty seq
		mockCache.CompleteFunc = func() bool {
			return true
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
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
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
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

		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
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
		} else if mockCache.PutCalls[0].Range != outstandingRange {
			t.Errorf("PutCall range mismatch: got %+v, want %+v", mockCache.PutCalls[0].Range, outstandingRange)
		}
		expectedData := make([]byte, defaultBlockSize)
		for i := 0; i < defaultBlockSize; i++ {
			expectedData[i] = byte(outstandingRange.From + int64(i))
		}
		if !bytes.Equal(mockCache.PutCalls[0].Data, expectedData) {
			t.Errorf("PutCall data mismatch: got %x, want %x", mockCache.PutCalls[0].Data, expectedData)
		}
		st.Duration = 0
		p := largefile.DownloadState{
			CachedBytes:      int64(defaultBlockSize),
			CachedBlocks:     1,
			DownloadedBytes:  int64(defaultBlockSize),
			DownloadedBlocks: 1,
			DownloadSize:     defaultContentSize,
			DownloadBlocks:   int64(defaultBlocks),
			Iterations:       1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("concurrency 0 - no outstanding", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache() // Default is no outstanding
		mockCache.CompleteFunc = func() bool {
			return true
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(0)...)
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run with concurrency 0 and no outstanding blocks failed: %v", err)
		}
		if len(mockReader.GetReaderCalls) != 0 {
			t.Errorf("expected 0 GetReaderCalls, got %d", len(mockReader.GetReaderCalls))
		}
		st.Duration = 0
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Resumeable: false, Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("concurrency 0 - with outstanding (should block or complete if ctx cancelled)", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache,
			largefile.WithDownloadRateController(&slowRateLimiter{}))

		runCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		st, err := dl.Run(runCtx)
		// Expect context deadline exceeded because generator sends to requestCh,
		// but no fetchers are there to receive, so it blocks until context times out.
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded with concurrency 0 and outstanding blocks, got %v", err)
		}
		st.Duration = 0
		st.CachedBlocks = 0
		st.CachedBytes = 0
		st.DownloadedBlocks = 0
		st.DownloadedBytes = 0
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Resumeable: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
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
		mockReader.ContentLengthAndBlockSizeFunc = func(context.Context) (size int64, blockSize int, err error) {
			return int64(numRanges * defaultBlockSize), defaultBlockSize, nil
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

		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(currentConcurrency)...)

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
		st.CachedBlocks, st.CachedBytes = 0, 0
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Resumeable: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	// Corrected "fetcher GetReader error"
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
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
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
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Resumeable: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	// Corrected "fetcher handleResponse (cache.Put) error"
	t.Run("fetcher handleResponse (cache.Put) error", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		putErr := errors.New("cache Put failed")
		mockCache.PutFunc = func(largefile.ByteRange, []byte) error {
			return putErr
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(defaultConcurrency)...)
		st, err := dl.Run(ctx)
		if err == nil {
			t.Fatal("expected error from cache.Put, got nil")
		}
		if !strings.Contains(err.Error(), putErr.Error()) {
			t.Errorf("expected error to contain %q, got %q", putErr.Error(), err.Error())
		}
		st.Duration = 0
		p := defaultIncompleteState
		if st.CacheErrors == 0 {
			t.Error("expected DownloadErrors to be > 0, got 0")
		}
		st.CacheErrors = 0
		p.DownloadedBlocks = 1
		p.DownloadedBytes = int64(defaultBlockSize)
		if got, want := st, (largefile.DownloadStatus{DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	// Corrected "concurrency 0 - with outstanding"
	t.Run("concurrency 0 - with outstanding (should block or complete if ctx cancelled)", func(t *testing.T) {
		mockReader := newDefaultMockReader()
		mockCache := newDefaultMockCache()
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(firstRange)(s, b)
		}
		dl := largefile.NewCachingDownloader(mockReader, mockCache, largefile.WithDownloadRateController(&slowRateLimiter{}))

		runCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		st, err := dl.Run(runCtx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context.DeadlineExceeded with concurrency 0 and outstanding blocks, got %v", err)
		}
		st.Duration = 0
		p := defaultIncompleteState
		if got, want := st, (largefile.DownloadStatus{Resumeable: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	// Corrected "all blocks downloaded successfully with progress"
	t.Run("all blocks downloaded successfully with progress", func(t *testing.T) {
		mockReader := newDefaultMockReader() // Ensure this is configured for the test's content/block size
		mockCache := newDefaultMockCache()
		mockCache.CompleteFunc = func() bool {
			return true
		}

		numBlocks := 4
		currentContentSize := int64(numBlocks * defaultBlockSize)
		mockReader.ContentLengthAndBlockSizeFunc = func(context.Context) (int64, int, error) {
			return currentContentSize, defaultBlockSize, nil
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
		progressCh := make(chan largefile.DownloadState, numBlocks+1)
		optsWithProgress := defaultOpts(2) // Concurrency 2
		optsWithProgress = append(optsWithProgress, largefile.WithDownloadProgress(progressCh))

		dl := largefile.NewCachingDownloader(mockReader, mockCache, optsWithProgress...)
		st, err := dl.Run(ctx)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		close(progressCh)

		var progressUpdates []largefile.DownloadState
		for p := range progressCh {
			progressUpdates = append(progressUpdates, p)
		}

		if len(progressUpdates) == 0 {
			t.Fatal("expected progress updates, got none")
		}

		sort.Slice(mockCache.PutCalls, func(i, j int) bool {
			return mockCache.PutCalls[i].Range.From < mockCache.PutCalls[j].Range.From
		})

		if len(mockCache.PutCalls) != numBlocks {
			t.Errorf("expected %d PutCalls, got %d", numBlocks, len(mockCache.PutCalls))
		}
		for i, r := range ranges {
			if i < len(mockCache.PutCalls) && mockCache.PutCalls[i].Range != r {
				t.Errorf("PutCall %d range mismatch: got %v, want %v", i, mockCache.PutCalls[i].Range, r)
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
		p := defaultState
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %+v, got %+v", want, got)
		}
	})

	// --- NEW/UPDATED TESTS ---

	// Replace/Unskip "multiple outstanding blocks success (concurrency > 1, with fixed pool)"
	// This test will now attempt a multi-block download with concurrency.
	// If the pool panic was due to a real issue, this might reveal it.
	// Otherwise, it tests successful concurrent download.
	t.Run("multiple outstanding blocks success (concurrency > 1)", func(t *testing.T) {
		numBlocks := 3
		concurrency := 2
		currentContentSize := int64(numBlocks * defaultBlockSize)

		mockReader := &MockReader{
			ContentLengthAndBlockSizeFunc: func(context.Context) (int64, int, error) {
				return currentContentSize, defaultBlockSize, nil
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
			PutFunc: func(_ largefile.ByteRange, _ []byte) error { return nil },
		}
		mockCache.CompleteFunc = func() bool {
			return true
		}

		expectedRanges := make([]largefile.ByteRange, numBlocks)
		for i := 0; i < numBlocks; i++ {
			expectedRanges[i] = largefile.ByteRange{
				From: int64(i * defaultBlockSize),
				To:   int64((i+1)*defaultBlockSize) - 1,
			}
		}
		mockCache.OutstandingFunc = func(s int, b *largefile.ByteRange) int {
			return newByteRangeSeq(expectedRanges...)(s, b)
		}

		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(concurrency)...)
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
			return mockCache.PutCalls[i].Range.From < mockCache.PutCalls[j].Range.From
		})

		for i, expectedRange := range expectedRanges {
			if i >= len(mockCache.PutCalls) {
				t.Errorf("missing PutCall for range index %d: %+v", i, expectedRange)
				continue
			}
			putCall := mockCache.PutCalls[i]
			if putCall.Range != expectedRange {
				t.Errorf("PutCall range mismatch at index %d: got %+v, want %+v", i, putCall.Range, expectedRange)
			}
			expectedDataSize := expectedRange.To - expectedRange.From + 1
			if int64(len(putCall.Data)) != expectedDataSize {
				t.Errorf("PutCall data length mismatch for range %+v: got %d, want %d", expectedRange, len(putCall.Data), expectedDataSize)
			}
		}
		st.Duration = 0
		p := largefile.DownloadState{
			CachedBytes:      currentContentSize,
			CachedBlocks:     int64(numBlocks),
			DownloadedBytes:  currentContentSize,
			DownloadedBlocks: int64(numBlocks),
			DownloadSize:     currentContentSize,
			DownloadBlocks:   int64(numBlocks),
			Iterations:       1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

	t.Run("file not a multiple of block size", func(t *testing.T) {
		numFullBlocks := 2
		partialBlockSize := defaultBlockSize / 2
		currentContentSize := int64(numFullBlocks*defaultBlockSize + partialBlockSize)
		totalBlocks := numFullBlocks + 1
		concurrency := 2

		mockReader := &MockReader{
			ContentLengthAndBlockSizeFunc: func(context.Context) (int64, int, error) {
				return currentContentSize, defaultBlockSize, nil
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
			PutFunc: func(_ largefile.ByteRange, _ []byte) error { return nil },
		}
		mockCache.CompleteFunc = func() bool {
			return true
		}

		expectedRanges := make([]largefile.ByteRange, totalBlocks)
		for i := 0; i < numFullBlocks; i++ {
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

		dl := largefile.NewCachingDownloader(mockReader, mockCache, defaultOpts(concurrency)...)
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
			return mockCache.PutCalls[i].Range.From < mockCache.PutCalls[j].Range.From
		})

		for i, expectedRange := range expectedRanges {
			if i >= len(mockCache.PutCalls) {
				t.Errorf("missing PutCall for range index %d: %+v", i, expectedRange)
				continue
			}
			putCall := mockCache.PutCalls[i]
			if putCall.Range != expectedRange {
				t.Errorf("PutCall range mismatch at index %d: got %+v, want %+v", i, putCall.Range, expectedRange)
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
		p := largefile.DownloadState{
			CachedBytes:      currentContentSize,
			CachedBlocks:     int64(totalBlocks),
			DownloadedBytes:  currentContentSize,
			DownloadedBlocks: int64(totalBlocks),
			DownloadSize:     currentContentSize,
			DownloadBlocks:   int64(totalBlocks),
			Iterations:       1,
		}
		if got, want := st, (largefile.DownloadStatus{Complete: true, DownloadState: p}); !reflect.DeepEqual(got, want) {
			t.Errorf("expected status %v, got %v", want, got)
		}
	})

} // End of TestCachingDownloader_Run
