// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"bytes"
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
)

// sliceReader serves ranges of data, tracking the readers that it hands out.
type sliceReader struct {
	data      []byte
	blockSize int
	opened    atomic.Int64
	closed    atomic.Int64
	hook      func(from int64) // called before a reader is returned.
	extra     bool             // return one byte too many.
	short     bool             // return one byte too few.
}

type countingCloser struct {
	io.Reader
	r *sliceReader
}

func (c countingCloser) Close() error {
	c.r.closed.Add(1)
	return nil
}

func (s *sliceReader) Name() string { return "slice" }
func (s *sliceReader) ContentLengthAndBlockSize() (int64, int) {
	return int64(len(s.data)), s.blockSize
}
func (s *sliceReader) Digest() digests.Hash { return digests.Hash{} }

func (s *sliceReader) GetReader(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	if s.hook != nil {
		s.hook(from)
	}
	if s.extra {
		to++
	}
	if s.short {
		to--
	}
	s.opened.Add(1)
	end := min(to+1, int64(len(s.data)))
	return countingCloser{bytes.NewReader(s.data[from:end]), s}, nil, nil
}

func testData(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i % 251)
	}
	return data
}

func streamAll(t *testing.T, dl *largefile.StreamingDownloader) ([]byte, largefile.StreamingStatus, error) {
	t.Helper()
	type result struct {
		st  largefile.StreamingStatus
		err error
	}
	ch := make(chan result, 1)
	go func() {
		st, err := dl.Run(context.Background())
		ch <- result{st, err}
	}()
	got, rerr := io.ReadAll(dl.Reader())
	res := <-ch
	if res.err != nil {
		return got, res.st, res.err
	}
	return got, res.st, rerr
}

func TestDownloaderClosesReaders(t *testing.T) {
	data := testData(10*100 + 7)
	for name, mod := range map[string]func(*sliceReader){
		"ok":    func(*sliceReader) {},
		"short": func(s *sliceReader) { s.short = true },
		"extra": func(s *sliceReader) { s.extra = true },
	} {
		r := &sliceReader{data: data, blockSize: 100}
		mod(r)
		dl := largefile.NewStreamingDownloader(r, largefile.WithDownloadConcurrency(2))
		got, _, err := streamAll(t, dl)
		if name == "ok" {
			if err != nil || !bytes.Equal(got, data) {
				t.Errorf("%v: got %v bytes, %v", name, len(got), err)
			}
		} else if err == nil {
			t.Errorf("%v: expected an error", name)
		}
		if o, c := r.opened.Load(), r.closed.Load(); o != c || o == 0 {
			t.Errorf("%v: opened %v readers, closed %v", name, o, c)
		}
	}
}

func TestRangeForIndex(t *testing.T) {
	invalid := largefile.ByteRange{From: -1, To: -1}
	for _, tc := range []struct {
		idx  int
		size int64
		bs   int
		want largefile.ByteRange
	}{
		{0, 25, 10, largefile.ByteRange{From: 0, To: 9}},
		{2, 25, 10, largefile.ByteRange{From: 20, To: 24}},
		{3, 25, 10, invalid},
		{-1, 25, 10, invalid},
		{0, 0, 10, invalid},
		{0, 10, 0, invalid},
		{1 << 20, 1 << 50, 1 << 20, largefile.ByteRange{From: 1 << 40, To: 1<<40 + 1<<20 - 1}},
	} {
		if got := largefile.RangeForIndex(tc.idx, tc.size, tc.bs); got != tc.want {
			t.Errorf("RangeForIndex(%v, %v, %v) = %v, want %v", tc.idx, tc.size, tc.bs, got, tc.want)
		}
	}
	if got := largefile.NumBlocks(10, 0); got != 0 {
		t.Errorf("NumBlocks with a zero block size: got %v", got)
	}
}

func TestStreamingWindow(t *testing.T) {
	const (
		blockSize = 100
		nblocks   = 100
		window    = 6
	)
	data := testData(blockSize * nblocks)
	var highest atomic.Int64
	var atRelease int64
	r := &sliceReader{data: data, blockSize: blockSize}
	r.hook = func(from int64) {
		if from != 0 {
			for {
				h := highest.Load()
				if from <= h || highest.CompareAndSwap(h, from) {
					return
				}
			}
		}
		// Hold up the first block whilst the others race ahead.
		time.Sleep(200 * time.Millisecond)
		atRelease = highest.Load()
	}
	dl := largefile.NewStreamingDownloader(r,
		largefile.WithDownloadConcurrency(4), largefile.WithDownloadWindow(window))
	got, st, err := streamAll(t, dl)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("got %v bytes, %v", len(got), err)
	}
	if got, want := atRelease, int64((window-1)*blockSize); got > want {
		t.Errorf("fetched block at offset %v whilst the first was outstanding, want at most %v", got, want)
	}
	if st.MaxOutOfOrder >= window {
		t.Errorf("buffered %v blocks, want fewer than %v", st.MaxOutOfOrder, window)
	}
}

type failFirst struct {
	*sliceReader
	mu   sync.Mutex
	seen map[int64]bool
}

func (f *failFirst) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	f.mu.Lock()
	first := !f.seen[from]
	f.seen[from] = true
	f.mu.Unlock()
	if first {
		return nil, nil, io.ErrUnexpectedEOF
	}
	return f.sliceReader.GetReader(ctx, from, to)
}

// TestStreamingWindowRetries ensures that a window as small as a single
// block does not prevent failed blocks from being retried.
func TestStreamingWindowRetries(t *testing.T) {
	data := testData(100 * 30)
	for _, window := range []int{1, 2, 5} {
		r := &failFirst{sliceReader: &sliceReader{data: data, blockSize: 100}, seen: map[int64]bool{}}
		dl := largefile.NewStreamingDownloader(r,
			largefile.WithDownloadConcurrency(3),
			largefile.WithDownloadWindow(window),
			largefile.WithDownloadWaitForCompletion(true))
		got, _, err := streamAll(t, dl)
		if err != nil || !bytes.Equal(got, data) {
			t.Errorf("window %v: got %v bytes, %v", window, len(got), err)
		}
	}
}

// TestStreamingFailure ensures that a failed block ends the download
// when not waiting for completion, regardless of the window.
func TestStreamingFailure(t *testing.T) {
	data := testData(100 * 30)
	r := &failFirst{sliceReader: &sliceReader{data: data, blockSize: 100}, seen: map[int64]bool{}}
	dl := largefile.NewStreamingDownloader(r,
		largefile.WithDownloadConcurrency(2), largefile.WithDownloadWindow(1))
	if _, _, err := streamAll(t, dl); err == nil {
		t.Error("expected an error")
	}
}
