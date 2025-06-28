// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/net/ratecontrol"
)

type DownloadState struct {
	CachedOrStreamedBytes  int64 // Total bytes cached.
	CachedOrStreamedBlocks int64 // Total blocks cached.
	CacheErrors            int64 // Total number of errors encountered while caching.
	DownloadedBytes        int64 // Total bytes downloaded so far.
	DownloadedBlocks       int64 // Total blocks downloaded so far.
	DownloadSize           int64 // Total size of the file in bytes.
	DownloadBlocks         int64 // Total number of blocks to download.
	DownloadRetries        int64 // Total number of retries made during the download.
	DownloadErrors         int64 // Total number of errors encountered during the download.
	Iterations             int64 // Number of iterations requiredd to complete the download.
}

func (ds DownloadState) updateAfterIteration(nds DownloadState) DownloadState {
	// Update the download state after a 'wait for completion' iteration,
	// do not update iterations and the overall download size and blocks
	return DownloadState{
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

type progressTracker struct {
	DownloadState
	ch chan<- DownloadState
}

func (pt *progressTracker) send() {
	if pt.ch == nil {
		return
	}
	select {
	case pt.ch <- DownloadState{
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
	}:
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
	bufPool   sync.Pool        // Pool for byte slices to reduce allocations.
}

type terminalError struct {
	err error // The error that caused the terminal state.
}

func (te *terminalError) Error() string {
	return te.err.Error()
}

func (te *terminalError) Unwrap() error {
	return te.err
}

func (te *terminalError) Is(target error) bool {
	_, ok := target.(*terminalError)
	return ok
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
	dl.bufPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, dl.blockSize))
		},
	}
	dl.progress = &progressTracker{
		DownloadState: DownloadState{
			DownloadSize:   dl.size,
			DownloadBlocks: int64(NumBlocks(dl.size, dl.blockSize)),
		},
		ch: dl.progressCh,
	}
	return dl
}

type response struct {
	data      *bytes.Buffer
	ByteRange // The byte range that was fetched.
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

func (dl *downloader) handleGet(ctx context.Context, req request, handler responseHandler) error {
	if err := dl.rateController.Wait(ctx); err != nil {
		return fmt.Errorf("ratecontrol failed: %w", err)
	}
	start := time.Now()
	rd, err := dl.get(ctx, req)
	if err != nil {
		return err
	}
	buf := dl.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	copied, err := io.Copy(buf, rd)
	dl.rateController.BytesTransferred(int(copied))
	dl.progress.incrementDownload(1, copied)
	if err != nil {
		return fmt.Errorf("failed to copy data for byte range %v: %w", req.ByteRange, err)
	}
	if copied != req.Size() {
		return fmt.Errorf("copied %d bytes for byte range %v, expected %d bytes", copied, req.ByteRange, req.Size())
	}
	var resp response
	resp.ByteRange = req.ByteRange
	resp.duration = time.Since(start)
	resp.data = buf
	if err := handler(ctx, resp); err != nil {
		return fmt.Errorf("failed to handle response for byte range %v: %w", req.ByteRange, err)
	}
	return nil
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
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, &terminalError{}) {
					return err
				}
				errs.Append(retryHandler(ctx, req, err))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
