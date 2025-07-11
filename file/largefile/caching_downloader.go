// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

// CachingDownloader is a downloader that caches streamed downloaded data to
// a local cache and supports resuming downloads from where they left off.
type CachingDownloader struct {
	*downloader
	cache DownloadCache
}

// NewCachingDownloader creates a new CachingDownloader instance.
func NewCachingDownloader(file Reader, cache DownloadCache, opts ...DownloadOption) (*CachingDownloader, error) {
	dl := &CachingDownloader{
		cache: cache,
	}
	var options downloadOptions
	for _, opt := range opts {
		opt(&options)
	}
	dl.downloader = newDownloader(file, options)
	if dl.cache == nil {
		return nil, fmt.Errorf("cache is not set for CachingDownloader")
	}
	if dl.hash.Algo != "" {
		return nil, fmt.Errorf("digest calculation is not supported for CachingDownloader")
	}
	csize, cblock := dl.cache.ContentLengthAndBlockSize()
	if csize != dl.size || cblock != dl.blockSize {
		return nil, fmt.Errorf("cache size (%d) or block size (%d) does not match file size (%d) or block size (%d)", csize, cblock, dl.size, dl.blockSize)
	}
	return dl, nil
}

// DownloadStatus holds the status for a completed download operation, including
// the progress made, whether the download is resumable, completed and
// the total duration of operation.
type DownloadStatus struct {
	DownloadStats
	Resumable bool          // Indicates if the download can be re-run.
	Complete  bool          // Indicates if the download completed successfully.
	Duration  time.Duration // Total duration of the download.
}

// Run executes the downloaded process. If the downloader encounters any errors
// it will return an
func (dl *CachingDownloader) Run(ctx context.Context) (DownloadStatus, error) {

	cachedBytes, cachedBlocks := dl.cache.CachedBytesAndBlocks()
	dl.progress.incrementCacheOrStream(cachedBytes, cachedBlocks)

	start := time.Now()
	var finalStats DownloadStats
	for {
		st, err := dl.runOnce(ctx)
		if st.Complete && err == nil {
			return dl.finalize(st, finalStats.updateAfterIteration(dl.progress.DownloadStats), start, nil)
		}
		dl.logger.Info("runOnce: download not complete, retrying", "iterations", st.Iterations, "error", err)
		if !dl.waitForCompletion {
			return dl.finalize(st, finalStats.updateAfterIteration(dl.progress.DownloadStats), start, err)
		}
		finalStats = finalStats.updateAfterIteration(dl.progress.DownloadStats)
		select {
		case <-ctx.Done():
			return dl.finalize(st, finalStats, start, ctx.Err())
		default:
		}
	}
}

func (dl *CachingDownloader) finalize(status DownloadStatus, state DownloadStats, start time.Time, err error) (DownloadStatus, error) {
	status.Duration = time.Since(start)
	status.DownloadStats = state
	if dl.progressCh != nil {
		dl.progress.DownloadStats = status.DownloadStats
		// Send the final download state to the progress channel, taking
		// care to ensure that a timeout is used, but also giving the
		// receiver a chance to read the final state.
		select {
		case dl.progressCh <- dl.progress.DownloadStats:
		case <-time.After(dl.progressTimeout):
		}
		close(dl.progressCh) // Ensure the progress channel is closed when done.
	}
	return status, err
}

func retryHandler(_ context.Context, _ request, err error) error {
	return err
}

func (dl *CachingDownloader) runOnce(ctx context.Context) (DownloadStatus, error) {
	reqCh := make(chan request, dl.concurrency) // Buffered channel for requests to fetch.
	g, ctx := errgroup.WithContext(ctx)
	g = errgroup.WithConcurrency(g, dl.concurrency+1) // +1 for the generator goroutine
	g.Go(func() error {
		defer close(reqCh)
		return dl.generator(ctx, reqCh)
	})
	for range dl.concurrency {
		g.Go(func() error {
			return dl.fetcher(ctx, reqCh, retryHandler, dl.handleResponse)
		})
	}

	err := g.Wait()
	err = errors.Squash(err, context.Canceled, context.DeadlineExceeded)

	// Any errors encountered during the download are considered resumable, ie,
	// the download can be restarted with the same cache to continue from where it left off.
	resumable := err != nil
	if errors.Is(err, ErrInternalError) {
		// If the error is a terminal error, we consider it non-resumable.
		resumable = false
	}

	st := DownloadStatus{
		DownloadStats: dl.progress.DownloadStats,
		Complete:      dl.cache.Complete() && err == nil,
		Resumable:     resumable,
	}
	return st, err
}

func (dl *CachingDownloader) handleResponse(_ context.Context, resp response) error {
	defer dl.bufPool.Put(resp.data) // Return the buffer to the pool after use.
	n, err := dl.cache.WriteAt(resp.data.Bytes(), resp.From)
	if err != nil {
		dl.progress.incrementCacheErrors()
		dl.logger.Info("handleResponse: cache write failed", "byteRange", resp.ByteRange, "error", err)
		return newInternalDownloadError(err)
	}
	dl.progress.incrementCacheOrStream(int64(n), 1)
	return nil
}

func (dl *CachingDownloader) generator(ctx context.Context, reqCh chan<- request) error {
	var br ByteRange
	// Start with the first uncached byte range.
	for n := dl.cache.NextOutstanding(0, &br); n != -1; n = dl.cache.NextOutstanding(n, &br) {
		select {
		case reqCh <- request{ByteRange: br}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
