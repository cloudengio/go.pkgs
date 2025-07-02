// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"log/slog"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/net/ratecontrol"
)

type downloaderOptions struct {
	concurrency       int
	rateController    ratecontrol.Limiter
	progressCh        chan<- DownloadState // Channel to report download progress.
	progressTimeout   time.Duration        // Delay between progress updates.
	logger            *slog.Logger
	waitForCompletion bool
}

type downloadOptions struct {
	downloaderOptions
	hash digests.Hash // Optional hash for computing the downloaded data as it is streamed.
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

// WithDownloadProgressDelay sets a timeout for certain progress updates,
// such as those sent when a download is completed to allow for the caller
// to process any pending updates. Routine updates, such as those sent
// whenever a new byte range is downloaded are sent on a best-effort basis.
func WithDownloadProgressTimeout(timeout time.Duration) DownloadOption {
	return func(o *downloadOptions) {
		o.progressTimeout = timeout
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

// WithDownloadDigest sets the hash function to be used for computing the digest of the downloaded data
// as it is streamed.
func WithDownloadDigest(h digests.Hash) DownloadOption {
	return func(o *downloadOptions) {
		o.hash = h
	}
}
