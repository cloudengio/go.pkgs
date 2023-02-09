// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package download provides a simple download mechanism that uses the
// fs.FS container API to implement the actual download. This allows
// rate control, retries and download management to be separated from the
// mechanism of the actual download. Downloaders can be provided for
// http/https, AWS S3 or any other local or cloud storage system for which
// an fs.FS implementation exists.
package download

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/net/ratecontrol"
	"cloudeng.io/sync/errgroup"
)

// Progress is used to communicate the progress of a download run.
type Progress struct {
	// Downloaded is the total number of items downloaded so far.
	Downloaded int64
	// Outstanding is the current size of the input channel for items
	// yet to be downloaded.
	Outstanding int64
}

// Option is used to configure the behaviour of a newly created Downloader.
type Option func(*options)

type options struct {
	rateController        *ratecontrol.Controller
	rateControllerOptions []ratecontrol.Option // backwards compatibility
	rateContoller         ratecontrol.Controller
	backoffErr            error
	concurrency           int
	progressInterval      time.Duration
	progressCh            chan<- Progress
	progressClose         bool
}

type downloader struct {
	options
	ticker     time.Ticker
	downloaded int64 // updated using atomic.

	progressMu   sync.Mutex
	progressLast time.Time // GUARDED_BY(progressMu)
}

// WithNumDownloaders controls the number of concurrent downloads used.
// If not specified the default of runtime.GOMAXPROCS(0) is used.
func WithNumDownloaders(concurrency int) Option {
	return func(o *options) {
		o.concurrency = concurrency
	}
}

// WithRateController sets the rate controller to use to enforce rate
// control. Backoff will be triggered if the supplied error is returned
// by the container (file.FS) implementation.
func WithRateController(retryErr error, rc *ratecontrol.Controller) Option {
	return func(o *options) {
		o.backoffErr = retryErr
		o.rateController = rc
	}
}

// WithProgress requests that progress messages are sent over the
// supplid channel. If close is true the progress channel will be closed
// when the downloader has finished. Close should be set to false if the same
// channel is shared across multiplied downloader instances.
func WithProgress(interval time.Duration, ch chan<- Progress, close bool) Option {
	return func(o *options) {
		o.progressInterval = interval
		o.progressCh = ch
		o.progressClose = close
	}
}

// New creates a new instance of a download.T.
func New(opts ...Option) T {
	dl := &downloader{}
	dl.concurrency = runtime.GOMAXPROCS(0)
	for _, opt := range opts {
		opt(&dl.options)
	}
	if dl.rateController == nil {
		dl.rateController = ratecontrol.New(dl.rateControllerOptions...)
	}
	return dl
}

// Run implements T.Run.
func (dl *downloader) Run(ctx context.Context,
	input <-chan Request,
	output chan<- Downloaded) error {

	var grp errgroup.T
	for i := 0; i < dl.concurrency; i++ {
		i := i
		grp.Go(func() error {
			return dl.runner(ctx, i, dl.progressCh, input, output)
		})
	}
	err := grp.Wait()

	dl.ticker.Stop()
	close(output)
	if dl.progressCh != nil && dl.progressClose {
		close(dl.progressCh)
	}
	return err
}

func (dl *downloader) updateDue() bool {
	dl.progressMu.Lock()
	defer dl.progressMu.Unlock()
	now := time.Now()
	if now.After(dl.progressLast.Add(dl.progressInterval)) {
		dl.progressLast = now
		return true
	}
	return false
}

func (dl *downloader) updateProgess(downloaded, outstanding int) {
	ndownloaded := atomic.AddInt64(&dl.downloaded, int64(downloaded))
	if dl.progressCh != nil && dl.updateDue() {
		select {
		case dl.progressCh <- Progress{
			Downloaded:  ndownloaded,
			Outstanding: int64(outstanding),
		}:
		default:
		}
	}
}

func (dl *downloader) runner(ctx context.Context, id int, progress chan<- Progress, input <-chan Request, output chan<- Downloaded) error {
	for {
		var request Request
		var ok bool
		select {
		case <-ctx.Done():
			return ctx.Err()
		case request, ok = <-input:
			if !ok {
				return nil
			}
		}
		if len(request.Names()) == 0 {
			// ignore empty requests.
			continue
		}
		downloaded, err := dl.downloadObjects(ctx, id, request)
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- downloaded:
		}
		dl.updateProgess(len(downloaded.Downloads), len(input))
	}
}

func (dl *downloader) downloadObjects(ctx context.Context, id int, request Request) (Downloaded, error) {
	download := Downloaded{
		Request:   request,
		Downloads: make([]Result, 0, len(request.Names())),
	}
	for _, name := range request.Names() {
		status, err := dl.downloadObject(ctx, request.Container(), name, request.FileMode())
		if err != nil {
			return download, err
		}
		download.Downloads = append(download.Downloads, status)
	}
	return download, nil
}

func (dl *downloader) downloadObject(ctx context.Context, downloadFS file.FS, name string, mode fs.FileMode) (Result, error) {
	result := Result{}
	dl.rateController.Wait(ctx)
	backoff := dl.rateController.Backoff()
	for {
		rd, err := downloadFS.OpenCtx(ctx, name)
		result.Retries = backoff.Retries()
		result.Err = err
		result.Name = name
		if err != nil {
			if errors.Is(err, dl.backoffErr) {
				if done, err := backoff.Wait(ctx); done {
					return result, err
				}
				continue
			}
			return result, nil
		}
		fi, err := rd.Stat()
		if err != nil {
			result.Err = err
			return result, err
		}
		result.FileInfo = fi
		buf := make([]byte, 0, int(fi.Size()))
		wr := bytes.NewBuffer(buf)
		n, err := io.Copy(wr, rd)
		if err != nil {
			result.Err = err
			return result, nil
		}
		if n != fi.Size() {
			result.Err = fmt.Errorf("short copy of downloaded object: %v: %v != %v", name, n, fi.Size())
			return result, nil
		}
		result.Contents = wr.Bytes()
		return result, nil
	}
}
