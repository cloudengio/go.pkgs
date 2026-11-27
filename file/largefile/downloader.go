// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/algo/ratecontrol"
	"cloudeng.io/errors"
)

// DownloadStats statistics on the download process and is used for progress
// reporting.
type DownloadStats struct {
	CachedOrStreamedBytes  int64 // Total bytes cached or streamed.
	CachedOrStreamedBlocks int64 // Total blocks cached or streamed.
	CacheErrors            int64 // Total number of errors encountered while caching.
	DownloadedBytes        int64 // Total bytes downloaded so far.
	DownloadedBlocks       int64 // Total blocks downloaded so far.
	DownloadSize           int64 // Total size of the file in bytes.
	DownloadBlocks         int64 // Total number of blocks to download.
	DownloadRetries        int64 // Total number of retries made during the download.
	DownloadErrors         int64 // Total number of errors encountered during the download.
	Iterations             int64 // Number of iterations requiredd to complete the download.
}

func (ds DownloadStats) updateAfterIteration(nds DownloadStats) DownloadStats {
	// Update the download state after a 'wait for completion' iteration,
	// do not update iterations and the overall download size and blocks
	return DownloadStats{
		CachedOrStreamedBytes:  ds.CachedOrStreamedBytes + nds.CachedOrStreamedBytes,
		CachedOrStreamedBlocks: ds.CachedOrStreamedBlocks + nds.CachedOrStreamedBlocks,
		CacheErrors:            ds.CacheErrors + nds.CacheErrors,
		DownloadedBytes:        ds.DownloadedBytes + nds.DownloadedBytes,
		DownloadedBlocks:       ds.DownloadedBlocks + nds.DownloadedBlocks,
		DownloadRetries:        ds.DownloadRetries + nds.DownloadRetries,
		DownloadErrors:         ds.DownloadErrors + nds.DownloadErrors,
		DownloadSize:           nds.DownloadSize,
		DownloadBlocks:         nds.DownloadBlocks,
		Iterations:             ds.Iterations + 1, // Increment iterations.
	}
}

// progressTracker accumulates download statistics, all of which are
// updated atomically apart from the download size and number of blocks,
// which are fixed.
type progressTracker struct {
	DownloadStats
	mu sync.Mutex // Serializes send so that updates are delivered in order.
	ch chan<- DownloadStats
}

// send reports the current statistics on a best-effort basis.
func (pt *progressTracker) send() {
	if pt.ch == nil {
		return
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()
	stats := DownloadStats{
		CachedOrStreamedBytes:  atomic.LoadInt64(&pt.CachedOrStreamedBytes),
		CachedOrStreamedBlocks: atomic.LoadInt64(&pt.CachedOrStreamedBlocks),
		CacheErrors:            atomic.LoadInt64(&pt.CacheErrors),
		DownloadedBytes:        atomic.LoadInt64(&pt.DownloadedBytes),
		DownloadedBlocks:       atomic.LoadInt64(&pt.DownloadedBlocks),
		DownloadSize:           pt.DownloadSize,
		DownloadBlocks:         pt.DownloadBlocks,
		DownloadRetries:        atomic.LoadInt64(&pt.DownloadRetries),
		DownloadErrors:         atomic.LoadInt64(&pt.DownloadErrors),
		Iterations:             atomic.LoadInt64(&pt.Iterations),
	}
	select {
	case pt.ch <- stats:
	default:
	}
}

func (pt *progressTracker) incrementCacheOrStream(bytes, blocks int64) {
	atomic.AddInt64(&pt.CachedOrStreamedBytes, bytes)
	atomic.AddInt64(&pt.CachedOrStreamedBlocks, blocks)
	pt.send()
}

func (pt *progressTracker) incrementRetries(retries int) {
	atomic.AddInt64(&pt.DownloadRetries, int64(retries))
	pt.send()
}

func (pt *progressTracker) incrementDownloadErrors() {
	atomic.AddInt64(&pt.DownloadErrors, 1)
	pt.send()
}

func (pt *progressTracker) incrementCacheErrors() {
	atomic.AddInt64(&pt.CacheErrors, 1)
	pt.send()
}

func (pt *progressTracker) incrementDownload(blocks int, size int64) {
	atomic.AddInt64(&pt.DownloadedBytes, size)
	atomic.AddInt64(&pt.DownloadedBlocks, int64(blocks))
	pt.send() // Send the updated state to the channel.
}

type downloader struct {
	downloadOptions
	progress  *progressTracker // Progress tracker for the download.
	file      Reader           // The large file to download.
	size      int64            // Total size of the file in bytes.
	blockSize int              // Size of each block in bytes.
	bufPool   sync.Pool        // Pool of *[]byte, each blockSize long, to reduce allocations.
}

func newDownloader(file Reader, opts downloadOptions) *downloader {
	dl := &downloader{file: file, downloadOptions: opts}
	if dl.logger == nil {
		dl.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	dl.logger = dl.logger.With("pkg", "cloudeng.io/file/largefile", "download", file.Name())
	if dl.rateController == nil {
		dl.rateController = ratecontrol.New(ratecontrol.WithNoRateControl()) // Default to no rate control.
	}
	if dl.concurrency <= 0 {
		dl.concurrency = runtime.NumCPU() // Default to number of CPU cores.
	}
	dl.size, dl.blockSize = dl.file.ContentLengthAndBlockSize()
	if dl.blockSize <= 0 {
		dl.blockSize = 4096 // Default block size is 4 KiB.
		dl.logger.Warn("block size not set, using default", "blockSize", dl.blockSize)
	}
	if dl.progressTimeout == 0 {
		dl.progressTimeout = time.Second
	}
	dl.bufPool = sync.Pool{
		New: func() any {
			b := make([]byte, dl.blockSize)
			return &b
		},
	}
	dl.progress = &progressTracker{
		DownloadSize:   dl.size,
		DownloadBlocks: int64(NumBlocks(dl.size, dl.blockSize)),
		ch:             dl.progressCh,
	}
	return dl
}

type response struct {
	data      []byte  // The data fetched, which is backed by buf.
	buf       *[]byte // The pooled storage for data, see release.
	ByteRange         // The byte range that was fetched.
	duration  time.Duration
}

type request struct {
	ByteRange // The byte range to fetch.
}

func (dl *downloader) get(ctx context.Context, req request) (io.ReadCloser, error) {
	backoff := dl.rateController.Backoff()
	retries := 0
	for {
		rd, retry, err := dl.file.GetReader(ctx, req.From, req.To)
		if err == nil {
			return rd, nil
		}
		if retry != nil && retry.IsRetryable() {
			if done, _ := backoff.Wait(ctx, retry); done {
				dl.logger.Info("getReader: backoff exhausted", "byteRange", req.ByteRange, "retries", retries, "error", err)
				return nil, fmt.Errorf("application backoff giving up after %d retries: %w", backoff.Retries(), err)
			}
			retries++
			dl.progress.incrementRetries(1)
			continue
		}
		dl.progress.incrementDownloadErrors()
		dl.logger.Info("getReader: non retryable error", "byteRange", req.ByteRange, "retries", retries, "error", err)
		return nil, fmt.Errorf("failed to get byte range %v: %w", req.ByteRange, err)
	}
}

type retryErrorHandler func(ctx context.Context, req request, err error) error
type responseHandler func(ctx context.Context, resp response) error

// release returns the storage used by resp to the pool.
func (dl *downloader) release(resp response) {
	dl.bufPool.Put(resp.buf)
}

// read reads exactly the bytes of req from rd into a pooled buffer.
func (dl *downloader) read(rd io.Reader, req request) (response, error) {
	buf := dl.bufPool.Get().(*[]byte)
	data := (*buf)[:req.Size()]
	n, err := io.ReadFull(rd, data)
	dl.rateController.BytesTransferred(n)
	dl.progress.incrementDownload(1, int64(n))
	if err == nil {
		// There must be nothing more to read.
		var extra [1]byte
		if m, _ := io.ReadFull(rd, extra[:]); m != 0 {
			err = fmt.Errorf("more than the expected %d bytes", req.Size())
		}
	}
	if err != nil {
		dl.bufPool.Put(buf)
		return response{}, fmt.Errorf("failed to read byte range %v, read %d of %d bytes: %w", req.ByteRange, n, req.Size(), err)
	}
	return response{data: data, buf: buf, ByteRange: req.ByteRange}, nil
}

func (dl *downloader) handleGet(ctx context.Context, req request, handler responseHandler) error {
	if err := dl.rateController.Wait(ctx); err != nil {
		return err // Context was canceled or deadline exceeded, return immediately.
	}
	start := time.Now()
	rd, err := dl.get(ctx, req)
	if err != nil {
		return err
	}
	defer rd.Close()
	resp, err := dl.read(rd, req)
	if err != nil {
		return err
	}
	resp.duration = time.Since(start)
	return handler(ctx, resp)
}

func (dl *downloader) fetcher(ctx context.Context, in <-chan request, retryHandler retryErrorHandler, handler responseHandler) error {
	errs := &errors.M{}
	for {
		select {
		case req, ok := <-in:
			if !ok {
				return errs.Err()
			}
			if err := dl.handleGet(ctx, req, handler); err != nil {
				if errors.Is(err, context.Canceled) ||
					errors.Is(err, context.DeadlineExceeded) ||
					errors.Is(err, ErrInternalError) {
					return err
				}
				errs.Append(retryHandler(ctx, req, err))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
