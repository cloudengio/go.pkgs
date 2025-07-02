// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"cloudeng.io/algo/container/heap"
	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

func (a response) Less(b response) bool {
	return a.ByteRange.From < b.ByteRange.From
}

// StreamingStatus holds status for a streaming download.
type StreamingStatus struct {
	DownloadState
	OutOfOrder    int64         // Total number of out-of-order responses encountered.
	MaxOutOfOrder int64         // Maximum number of out-of-order responses at any point.
	Duration      time.Duration // Total duration of the download.
}

// StreamingDownloader is a downloader that streams data from a large file.
// The downloader uses concurrent byte range requests to fetch data and then
// serializes the responses into a single stream for reading.
type StreamingDownloader struct {
	*downloader
	pipeRd       *io.PipeReader // Reader for streaming data.
	pipeWr       *io.PipeWriter // Writer for streaming data.
	responseCh   chan response  // Channel for responses from fetchers.
	requestCh    chan request   // Channel for inflight requests.
	outstanding  *ByteRanges    // Byte ranges to be downloaded.
	retryTracker retryTracker
	mu           sync.Mutex // Mutex to protect access to the heap.
	heap         heap.Heap[response]
	tracking     ByteRange
	outOfOrder   int64
	maxHeapSize  int64 // Maximum size of the heap during the download.
}

// NewStreamingDownloader creates a new StreamingDownloader instance.
func NewStreamingDownloader(file Reader, opts ...DownloadOption) *StreamingDownloader {
	dl := &StreamingDownloader{}
	var options downloadOptions
	for _, opt := range opts {
		opt(&options)
	}
	dl.downloader = newDownloader(file, options)
	dl.responseCh = make(chan response, dl.concurrency)   // Buffered channel for responses from fetchers.
	dl.requestCh = make(chan request, dl.concurrency)     // Buffered channel for requests to fetch.
	dl.pipeRd, dl.pipeWr = io.Pipe()                      // Create a pipe for streaming data.
	dl.outstanding = NewByteRanges(dl.size, dl.blockSize) // Create byte ranges for downloading.

	dl.tracking = ByteRange{From: -1, To: -1} // Initialize tracking range to an invalid state.

	if dl.waitForCompletion {
		// Track byte ranges that need to be re-issued by the generator goroutine.
		dl.retryTracker.ByteRangesTracker = NewByteRangesTracker(dl.size, dl.blockSize)
	}
	return dl
}

func (dl *StreamingDownloader) Run(ctx context.Context) (StreamingStatus, error) {
	start := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	g = errgroup.WithConcurrency(g, dl.concurrency+1) // +1 for the generator goroutine
	g.Go(func() error {
		defer close(dl.requestCh)
		return dl.generator(ctx)
	})
	for range dl.concurrency {
		g.Go(func() error {
			return dl.fetcher(ctx, dl.requestCh, dl.retryError, dl.handleResponse)
		})
	}
	err := g.Wait()
	err = errors.Squash(err, context.Canceled, context.DeadlineExceeded)

	st := StreamingStatus{
		DownloadState: dl.progress.DownloadState,
		Duration:      time.Since(start),
		OutOfOrder:    dl.outOfOrder,
		MaxOutOfOrder: dl.maxHeapSize,
	}
	cerr := dl.pipeWr.CloseWithError(err)
	if err != nil {
		return st, err
	}
	return st, cerr // Close the writer to signal end of stream.
}

// Read implements io.Reader.
func (dl *StreamingDownloader) Read(buf []byte) (int, error) {
	return dl.pipeRd.Read(buf)
}

// Reader returns an io.Reader.
func (dl *StreamingDownloader) Reader() io.Reader {
	return dl.pipeRd
}

// ContentLength returns the content length header for the file being
// downloaded.
func (dl *StreamingDownloader) ContentLength() int64 {
	return dl.size
}

func (dl *StreamingDownloader) reissue(ctx context.Context, from int64) {
	// Reissue the requests for the byte ranges that have been successfully downloaded.
	// This is useful for retrying failed requests or re-issuing requests that were
	// not completed.
	var br ByteRange
	idx := dl.retryTracker.Block(from)
	for n := dl.retryTracker.nextSetAndClear(idx, &br); n != -1; n = dl.retryTracker.nextSetAndClear(n, &br) {
		select {
		case dl.requestCh <- request{ByteRange: br}:
		case <-ctx.Done():
			return
		}
	}
}

func (dl *StreamingDownloader) generator(ctx context.Context) error {
	// Prime the request channel with the initial byte ranges to download.
	var br ByteRange
	for n := dl.outstanding.NextClear(0, &br); n != -1; n = dl.outstanding.NextClear(n, &br) {
		select {
		case dl.requestCh <- request{ByteRange: br}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if !dl.waitForCompletion {
		// If we are not waiting for completion, we can return early.
		return nil
	}
	// Wait for requests that need to be re-issued.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-dl.outstanding.Notify():
			if tail, ok := dl.outstanding.Tail(); ok && tail.To+1 == dl.size {
				// All ranges have been requested, so we can stop generating requests.
				return nil
			}
		case <-dl.retryTracker.notify():
			dl.mu.Lock()
			from := max(dl.tracking.From, 0)
			dl.mu.Unlock()
			dl.reissue(ctx, from)
		}
	}
}

func (dl *StreamingDownloader) writeToPipeLocked(resp response) error {
	if dl.tracking.From == -1 && resp.From != 0 {
		return &internalError{fmt.Errorf("first response must start at 0, got %d", resp.From)}
	}
	if resp.From != dl.tracking.To+1 {
		return &internalError{fmt.Errorf("out of order response: expected %d, got %d", dl.tracking.To+1, resp.From)}
	}
	dl.tracking = resp.ByteRange // Update the tracking range to the current response.
	data := resp.data.Bytes()
	n, err := dl.pipeWr.Write(data)
	if err != nil {
		return err
	}
	if n < len(data) {
		return io.ErrShortWrite
	}
	if dl.hash.Hash != nil {
		if _, err := dl.hash.Write(data); err != nil {
			return err
		}
	}
	dl.outstanding.Set(resp.From)
	dl.progress.incrementCacheOrStream(int64(n), 1) // Increment the progress tracker for cached or streamed bytes.
	return nil
}

func (dl *StreamingDownloader) pushLocked(resp response) {
	dl.heap.Push(resp)
	dl.outOfOrder++
	dl.maxHeapSize = max(dl.maxHeapSize, int64(dl.heap.Len()))
}

func (dl *StreamingDownloader) retryError(_ context.Context, req request, err error) error {
	if !dl.waitForCompletion {
		return err
	}
	if err == nil {
		return nil // No error, nothing to retry.
	}
	dl.retryTracker.set(req.From)
	return nil
}

func (dl *StreamingDownloader) handleResponse(_ context.Context, resp response) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	lastWritten, ok := dl.outstanding.Tail()
	lastTo := lastWritten.To
	if !ok {
		lastTo = -1
	}
	if resp.From != lastTo+1 { // lastTo is -1 if there are no written ranges yet.
		dl.pushLocked(resp) // Push the response to the heap since it can't be written yet.
		return nil
	}
	// first response, so write to pipe.
	if err := dl.writeToPipeLocked(resp); err != nil {
		return err
	}
	return dl.drainCacheLocked(resp.ByteRange.To + 1) // Drain the cache to write any responses that are now in order.
}

func (dl *StreamingDownloader) drainCacheLocked(nextOffset int64) error {
	for dl.heap.Len() > 0 {
		head := dl.heap.Pop()
		if nextOffset == head.From {
			if err := dl.writeToPipeLocked(head); err != nil {
				return err
			}
			nextOffset = head.ByteRange.To + 1
			continue
		}
		dl.pushLocked(head) // Push back the response if it is not the next expected.
		break
	}
	return nil
}

type retryTracker struct {
	sync.RWMutex
	*ByteRangesTracker
	ch      chan struct{} // Channel to notify when the byte ranges are updated.
	pending bool
}

func (rt *retryTracker) set(from int64) {
	rt.Lock()
	defer rt.Unlock()
	rt.ByteRangesTracker.Set(from)
	rt.pending = true
	rt.kickLocked() // Notify that the byte ranges have been updated.
}

func (rt *retryTracker) notify() <-chan struct{} {
	rt.Lock()
	defer rt.Unlock()
	if rt.pending {
		rt.pending = false // Reset the pending flag.
		closedCh := make(chan struct{})
		close(closedCh) // Close the channel to notify that the byte ranges have been updated.
		return closedCh
	}
	if rt.ch == nil {
		rt.ch = make(chan struct{})
	}
	return rt.ch
}

func (rt *retryTracker) nextSetAndClear(start int, br *ByteRange) int {
	rt.Lock()
	defer rt.Unlock()
	n := rt.ByteRangesTracker.NextSet(start, br)
	if n != -1 {
		rt.ByteRangesTracker.Clear(br.From)
	}
	return n
}

func (rt *retryTracker) kickLocked() {
	if rt.ch == nil {
		return
	}
	close(rt.ch) // Close the channel to notify that the byte ranges have been updated.
	rt.ch = nil
}
