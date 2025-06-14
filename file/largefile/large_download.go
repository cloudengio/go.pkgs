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
	verifyChecksum bool // Whether to verify the checksum of downloaded data.
}

type DownloadOption func(*downloadOptions)

func WithDownloadConcurrency(n int) DownloadOption {
	return func(o *downloadOptions) {
		o.concurrency = n
	}
}

func WithDownloadRateController(rc ratecontrol.Limiter) DownloadOption {
	return func(o *downloadOptions) {
		o.rateController = rc
	}
}

func WithDownloadLogger(logger *slog.Logger) DownloadOption {
	return func(o *downloadOptions) {
		o.logger = logger
	}
}

func WithDownloadProgress(progress chan<- DownloadState) DownloadOption {
	return func(o *downloadOptions) {
		o.progressCh = progress
	}
}

func WithDownloadWaitForCompletion(wait bool) DownloadOption {
	return func(o *downloadOptions) {
		o.waitForCompletion = wait
	}
}

type DownloadState struct {
	CachedBytes      int64 // Total bytes cached.
	CachedBlocks     int64 // Total blocks cached.
	DownloadedBytes  int64 // Total bytes downloaded so far.
	DownloadedBlocks int64 // Total blocks downloaded so far.
	DownloadSize     int64 // Total size of the file in bytes.
	DownloadBlocks   int64 // Total number of blocks to download.
	Retries          int64 // Total number of retries made during the download.
	Iterations       int64 // Number of iterations requiredd to complete the download.
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
	atomic.AddInt64(&pt.Retries, int64(retries))
}

func (pt *progressTracker) incrementIterations() {
	atomic.AddInt64(&pt.Iterations, 1)
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
	requestCh chan request     // Channel for byte ranges to fetch.
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
	if d.rateController == nil {
		d.rateController = ratecontrol.New(ratecontrol.WithNoRateControl()) // Default to no rate control.
	}
	d.requestCh = make(chan request, d.concurrency) // Buffered channel for byte ranges to fetch.
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
				return nil, fmt.Errorf("application backoff giving up after %d retries: %w", backoff.Retries(), err)
			}
			retries++
			continue
		}
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
	for r := range dl.cache.Cached() {
		cachedBytes += r.Size()
		cachedBlocks++
	}
	if cachedBytes != 0 {
		dl.progress.incrementCache(cachedBlocks, cachedBytes)
	}

	start := time.Now()
	if !dl.waitForCompletion {
		st, err := dl.runOnce(ctx)
		st.Duration = time.Since(start)
		return st, err
	}
	for {
		st, err := dl.runOnce(ctx)
		if st.Complete && err == nil {
			st.Duration = time.Since(start)
			return st, nil
		}
		dl.progress.incrementIterations()
	}
}

func (dl *CachingDownloader) runOnce(ctx context.Context) (DownloadStatus, error) {
	g, ctx := errgroup.WithContext(ctx)
	g = errgroup.WithConcurrency(g, dl.concurrency) // +1 for the generator goroutine
	g.Go(func() error {
		defer close(dl.requestCh)
		return dl.generator(ctx, dl.requestCh)
	})
	for range dl.concurrency {
		g.Go(func() error {
			return dl.fetcher(ctx, dl.requestCh, dl.handleResponse)
		})
	}

	err := g.Wait()
	err = errors.Squash(err, context.Canceled, context.DeadlineExceeded)

	// Any errors encountered during the download are considered resumable, ie,
	// the download can be restarted with the same cache to continue from where it left off.
	resumeable := true && err != nil
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
		return &terminalError{err}
	}
	dl.progress.incrementCache(1, int64(len(resp.data.Bytes())))
	return nil
}

func (dl *CachingDownloader) generator(ctx context.Context, reqCh chan<- request) error {
	seq := 0
	for dl := range dl.cache.Outstanding() {
		select {
		case reqCh <- request{
			seq:       seq,
			ByteRange: dl,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
		seq++
	}
	return nil
}

type StreamingDownloadOption func(*downloadStreamingOptions)

func WithVerifyChecksum(verify bool) StreamingDownloadOption {
	return func(o *downloadStreamingOptions) {
		o.verifyChecksum = verify
	}
}

// StreamingDownloader is a downloader that streams data from a large file.
// The downloader uses concurrent byte range requests to fetch data and then
// serializes the responses into a single stream for reading.
type StreamingDownloader struct {
	*downloader
	verifyChecksum bool           // Whether to verify the checksum of downloaded data.
	pipeRd         io.ReadCloser  // Reader for streaming data.
	pipeWr         io.WriteCloser // Writer for streaming data.
	responseCh     chan response  // Channel for responses from fetchers.
}

// NewStreamingDownloader creates a new StreamingDownloader instance.
func NewStreamingDownloader(file Reader, opts ...StreamingDownloadOption) *StreamingDownloader {
	d := &StreamingDownloader{}
	var options downloadStreamingOptions
	for _, opt := range opts {
		opt(&options)
	}
	d.verifyChecksum = options.verifyChecksum
	d.downloader = newDownloader(file, options.downloadOptionsCommon)
	d.responseCh = make(chan response, d.concurrency) // Buffered channel for responses from fetchers.
	d.pipeRd, d.pipeWr = io.Pipe()                    // Create a pipe for streaming data.
	return d
}
