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
	"cloudeng.io/sync/errgroup"
)

type downloadOptionsCommon struct {
	concurrency    int
	rateController ratecontrol.Limiter
	progressCh     chan<- DownloadState // Channel to report download progress.
	logger         *slog.Logger
}

type downloadOptions struct {
	downloadOptionsCommon // Common options for downloading.
	waitForCompletion     bool
}

type downloadStreamingOptions struct {
	downloadOptionsCommon
	verifyDigest bool // Whether to verify the digest of downloaded data.
}

type DownloadOption func(*downloadOptions)

// WithDownloadConcurrency sets the number of concurrent download goroutines.
func WithDownloadConcurrency(n int) DownloadOption {
	return func(o *downloadOptions) {
		o.concurrency = n
	}
}

// WithDownloadRateController sets the rate controller for the download.
func WithDownloadRateController(rc ratecontrol.Limiter) DownloadOption {
	return func(o *downloadOptions) {
		o.rateController = rc
	}
}

// WithDownloadLogger sets the logger for the download.
func WithDownloadLogger(logger *slog.Logger) DownloadOption {
	return func(o *downloadOptions) {
		o.logger = logger
	}
}

// WithDownloadProgress sets the channel to report download progress.
func WithDownloadProgress(progress chan<- DownloadState) DownloadOption {
	return func(o *downloadOptions) {
		o.progressCh = progress
	}
}

// WithDownloadWaitForCompletion sets whether the download should iterate,
// until the download is successfully completed, or return after one iteration.
// An iteration represents a single pass through the download process whereby
// every outstsanding byte range is attempted to be downloaded once with retries.
// A download will either complete after any specified retries or be left
// outstanding for the next iteration.
func WithDownloadWaitForCompletion(wait bool) DownloadOption {
	return func(o *downloadOptions) {
		o.waitForCompletion = wait
	}
}

type DownloadState struct {
	CachedBytes      int64 // Total bytes cached.
	CachedBlocks     int64 // Total blocks cached.
	CacheErrors      int64 // Total number of errors encountered while caching.
	DownloadedBytes  int64 // Total bytes downloaded so far.
	DownloadedBlocks int64 // Total blocks downloaded so far.
	DownloadSize     int64 // Total size of the file in bytes.
	DownloadBlocks   int64 // Total number of blocks to download.
	DownloadRetries  int64 // Total number of retries made during the download.
	DownloadErrors   int64 // Total number of errors encountered during the download.
	Iterations       int64 // Number of iterations requiredd to complete the download.
}

func (ds DownloadState) updateAfterIteration(nds DownloadState) DownloadState {
	// Update the download state after a 'wait for completion' iteration,
	// do not update iterations and the overall download size and blocks
	return DownloadState{
		CachedBytes:      ds.CachedBytes + nds.CachedBytes,
		CachedBlocks:     ds.CachedBlocks + nds.CachedBlocks,
		CacheErrors:      ds.CacheErrors + nds.CacheErrors,
		DownloadedBytes:  ds.DownloadedBytes + nds.DownloadedBytes,
		DownloadedBlocks: ds.DownloadedBlocks + nds.DownloadedBlocks,
		DownloadRetries:  ds.DownloadRetries + nds.DownloadRetries,
		DownloadErrors:   ds.DownloadErrors + nds.DownloadErrors,
		DownloadSize:     nds.DownloadSize,
		DownloadBlocks:   nds.DownloadBlocks,
		Iterations:       ds.Iterations + 1, // Increment iterations.
	}
}

type progressTracker struct {
	DownloadState
	ch chan<- DownloadState
}

func (pt *progressTracker) incrementCache(blocks int, size int64) {
	atomic.AddInt64(&pt.CachedBytes, size)
	atomic.AddInt64(&pt.CachedBlocks, int64(blocks))
}

func (pt *progressTracker) incrementRetries(retries int) {
	atomic.AddInt64(&pt.DownloadRetries, int64(retries))
}

func (pt *progressTracker) incrementDownloadErrors() {
	atomic.AddInt64(&pt.DownloadErrors, 1)
}

func (pt *progressTracker) incrementCacheErrors() {
	atomic.AddInt64(&pt.CacheErrors, 1)
}

func (pt *progressTracker) incrementDownload(blocks int, size int64) {
	tbytes := atomic.AddInt64(&pt.DownloadedBytes, size)
	tblocks := atomic.AddInt64(&pt.DownloadedBlocks, int64(blocks))
	if pt.ch == nil {
		return
	}
	select {
	case pt.ch <- DownloadState{
		DownloadedBytes:  tbytes,
		DownloadedBlocks: tblocks,
		DownloadSize:     pt.DownloadSize,
		DownloadBlocks:   pt.DownloadBlocks,
	}:
	default:
	}
}

type downloader struct {
	downloadOptionsCommon
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

func newDownloader(file Reader, opts downloadOptionsCommon) *downloader {
	d := &downloader{file: file, downloadOptionsCommon: opts}
	if d.logger == nil {
		d.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	d.logger = d.logger.With("pkg", "cloudeng.io/file/largefile", "download", file.Name())

	if d.rateController == nil {
		d.rateController = ratecontrol.New(ratecontrol.WithNoRateControl()) // Default to no rate control.
	}
	if d.concurrency <= 0 {
		d.concurrency = runtime.NumCPU() // Default to number of CPU cores.
	}
	return d
}

func (dl *downloader) init(ctx context.Context) error {
	var err error
	dl.size, dl.blockSize, err = dl.file.ContentLengthAndBlockSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to get file size: %w", err)
	}
	dl.bufPool = sync.Pool{
		New: func() interface{} {
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
	return nil
}

func (dl *downloader) get(ctx context.Context, req request) (io.ReadCloser, error) {
	backoff := dl.rateController.Backoff()
	retries := 0
	for {
		rd, retry, err := dl.file.GetReader(ctx, req.From, req.To)
		if err == nil {
			dl.progress.incrementRetries(retries)
			return rd, nil
		}
		if retry.IsRetryable() {
			if done, _ := backoff.Wait(ctx, retry); done {
				dl.logger.Info("getReader: backoff exhausted", "byteRange", req.ByteRange, "retries", retries, "error", err)
				return nil, fmt.Errorf("application backoff giving up after %d retries: %w", backoff.Retries(), err)
			}
			retries++
			dl.progress.incrementRetries(retries)
			continue
		}
		dl.progress.incrementDownloadErrors()
		dl.logger.Info("getReader: non retryable error", "byteRange", req.ByteRange, "retries", retries, "error", err)
		return nil, fmt.Errorf("failed to get byte range %v: %w", req.ByteRange, err)
	}
}

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
	resp.seq = req.seq
	resp.ByteRange = req.ByteRange
	resp.duration = time.Since(start)
	resp.data = buf
	if err := handler(ctx, resp); err != nil {
		return fmt.Errorf("failed to handle response for byte range %v: %w", req.ByteRange, err)
	}
	return nil
}

func (dl *downloader) fetcher(ctx context.Context, in <-chan request, handler responseHandler) error {
	errs := &errors.M{}
	for {
		select {
		case req, ok := <-in:
			if !ok {
				return errs.Err()
			}
			if err := dl.handleGet(ctx, req, handler); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, &terminalError{}) {
					return err
				}
				errs.Append(err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// CachingDownloader is a downloader that caches streamed downloaded data to
// a local cache and supports resuming downloads from where they left off.
type CachingDownloader struct {
	*downloader
	waitForCompletion bool // Whether to wait for the download to complete or return after one iteration.
	cache             DownloadCache
}

// NewCachingDownloader creates a new CachingDownloader instance.
func NewCachingDownloader(file Reader, cache DownloadCache, opts ...DownloadOption) *CachingDownloader {
	d := &CachingDownloader{
		cache: cache,
	}
	var options downloadOptions
	for _, opt := range opts {
		opt(&options)
	}
	d.waitForCompletion = options.waitForCompletion
	d.downloader = newDownloader(file, options.downloadOptionsCommon)
	return d
}

type response struct {
	data      *bytes.Buffer
	ByteRange // The byte range that was fetched.
	seq       int
	duration  time.Duration
}

type request struct {
	seq       int
	ByteRange // The byte range to fetch.
}

// DownloadStatus holds the status of a download operation, including
// the progress made, whether the download is resumable, completed and
// the total duration of operation.
type DownloadStatus struct {
	DownloadState
	Resumeable bool          // Indicates if the download can be re-run.
	Complete   bool          // Indicates if the download completed successfully.
	Duration   time.Duration // Total duration of the download.
}

// Run executes the downloaded process. If the downloader encounters any errors
// it will return an
func (dl *CachingDownloader) Run(ctx context.Context) (DownloadStatus, error) {
	if dl.cache == nil {
		return DownloadStatus{}, fmt.Errorf("cache is not set for CachingDownloader")
	}
	if err := dl.init(ctx); err != nil {
		return DownloadStatus{}, err
	}
	csize, cblock := dl.cache.ContentLengthAndBlockSize()
	if csize != dl.size || cblock != dl.blockSize {
		return DownloadStatus{}, fmt.Errorf("cache size (%d) or block size (%d) does not match file size (%d) or block size (%d)", csize, cblock, dl.size, dl.blockSize)
	}

	var cachedBytes int64
	var cachedBlocks int
	var br ByteRange
	for n := dl.cache.NextCached(0, &br); n != -1; n = dl.cache.NextCached(n, &br) {
		cachedBytes += br.Size()
		cachedBlocks++
	}
	if cachedBytes != 0 {
		dl.progress.incrementCache(cachedBlocks, cachedBytes)
	}

	start := time.Now()
	var finalState DownloadState
	for {
		st, err := dl.runOnce(ctx)
		if st.Complete && err == nil {
			st.DownloadState = finalState.updateAfterIteration(dl.progress.DownloadState)
			st.Duration = time.Since(start)
			return st, nil
		}
		dl.logger.Info("runOnce: download not complete, retrying", "iterations", st.Iterations, "error", err)
		if !dl.waitForCompletion {
			st.DownloadState = finalState.updateAfterIteration(dl.progress.DownloadState)
			st.Duration = time.Since(start)
			return st, err
		}
		finalState = finalState.updateAfterIteration(dl.progress.DownloadState)
		select {
		case <-ctx.Done():
			st.Duration = time.Since(start)
			st.DownloadState = finalState
			return st, ctx.Err()
		default:
		}
	}
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
			return dl.fetcher(ctx, reqCh, dl.handleResponse)
		})
	}

	err := g.Wait()
	err = errors.Squash(err, context.Canceled, context.DeadlineExceeded)

	// Any errors encountered during the download are considered resumable, ie,
	// the download can be restarted with the same cache to continue from where it left off.
	resumeable := err != nil
	if errors.Is(err, &terminalError{}) {
		// If the error is a terminal error, we consider it non-resumable.
		resumeable = false
	}

	st := DownloadStatus{
		DownloadState: dl.progress.DownloadState,
		Complete:      dl.cache.Complete() && err == nil,
		Resumeable:    resumeable,
	}
	return st, err
}

func (dl *CachingDownloader) handleResponse(_ context.Context, resp response) error {
	defer dl.bufPool.Put(resp.data) // Return the buffer to the pool after use.
	if err := dl.cache.Put(resp.ByteRange, resp.data.Bytes()); err != nil {
		dl.progress.incrementCacheErrors()
		dl.logger.Info("handleResponse: cache write failed", "byteRange", resp.ByteRange, "error", err)
		return &terminalError{err}
	}
	dl.progress.incrementCache(1, int64(len(resp.data.Bytes())))
	return nil
}

func (dl *CachingDownloader) generator(ctx context.Context, reqCh chan<- request) error {
	seq := 0
	var br ByteRange
	// Start with the first uncached byte range.
	for n := dl.cache.NextOutstanding(0, &br); n != -1; n = dl.cache.NextOutstanding(n, &br) {
		select {
		case reqCh <- request{
			seq:       seq,
			ByteRange: br,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
		seq++
	}
	return nil
}

type StreamingDownloadOption func(*downloadStreamingOptions)

func WithVerifyDigest(verify bool) StreamingDownloadOption {
	return func(o *downloadStreamingOptions) {
		o.verifyDigest = verify
	}
}

// StreamingDownloader is a downloader that streams data from a large file.
// The downloader uses concurrent byte range requests to fetch data and then
// serializes the responses into a single stream for reading.
type StreamingDownloader struct {
	*downloader
	verifyDigest bool           // Whether to verify the digest of downloaded data.
	pipeRd       io.ReadCloser  // Reader for streaming data.
	pipeWr       io.WriteCloser // Writer for streaming data.
	responseCh   chan response  // Channel for responses from fetchers.
}

// NewStreamingDownloader creates a new StreamingDownloader instance.
func NewStreamingDownloader(file Reader, opts ...StreamingDownloadOption) *StreamingDownloader {
	d := &StreamingDownloader{}
	var options downloadStreamingOptions
	for _, opt := range opts {
		opt(&options)
	}
	d.verifyDigest = options.verifyDigest
	d.downloader = newDownloader(file, options.downloadOptionsCommon)
	d.responseCh = make(chan response, d.concurrency) // Buffered channel for responses from fetchers.
	d.pipeRd, d.pipeWr = io.Pipe()                    // Create a pipe for streaming data.
	return d
}
