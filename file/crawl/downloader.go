// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/sync/errgroup"
)

// DownloadProgress is used to communicate the progress of a download run.
type DownloadProgress struct {
	// Downloaded is the total number of items downloaded so far.
	Downloaded int64
	// Outstanding is the current size of the input channel for items to
	// be downloaded.
	Outstanding int64
}

type loggerOpts struct {
	logLevel  int
	logOut    io.Writer
	logPrefix string
}

func (l *loggerOpts) log(level int, format string, args ...interface{}) {
	if l.logLevel >= level && l.logOut != nil {
		fmt.Fprintf(l.logOut, l.logPrefix+strings.TrimSuffix(format, "\n")+"\n", args...)
	}
}

type DownloaderOption func(*downloaderOptions)

type downloaderOptions struct {
	rateDelay        time.Duration
	backOffErr       error
	backOffStart     time.Duration
	backoffSteps     int
	concurrency      int
	progressInterval time.Duration
	progressCh       chan<- DownloadProgress
	progressClose    bool
	loggerOpts
}

type downloader struct {
	downloaderOptions
	ticker     time.Ticker
	downloaded int64 // updated using atomic.

	progressMu   sync.Mutex
	progressLast time.Time // GUARDED_BY(progressMu)
}

func WithRequestsPerMinute(rpm int) DownloaderOption {
	return func(o *downloaderOptions) {
		if rpm > 60 {
			o.rateDelay = time.Second / time.Duration(rpm)
			return
		}
		o.rateDelay = time.Minute / time.Duration(rpm)
	}
}

func WithBackoffParameters(err error, start time.Duration, steps int) DownloaderOption {
	return func(o *downloaderOptions) {
		o.backOffErr = err
		o.backOffStart = start
		o.backoffSteps = steps
	}
}

func WithNumDownloaders(concurrency int) DownloaderOption {
	return func(o *downloaderOptions) {
		o.concurrency = concurrency
	}
}

func WithDownloadProgress(interval time.Duration, ch chan<- DownloadProgress, close bool) DownloaderOption {
	return func(o *downloaderOptions) {
		o.progressInterval = interval
		o.progressCh = ch
		o.progressClose = close
	}
}

func WithDownloadLogging(level uint32, out io.Writer) DownloaderOption {
	return func(o *downloaderOptions) {
		o.logLevel = int(level)
		o.logOut = out
	}
}

func NewDownloader(opts ...DownloaderOption) Downloader {
	dl := &downloader{}
	for _, opt := range opts {
		opt(&dl.downloaderOptions)
	}
	if dl.concurrency == 0 {
		dl.concurrency = runtime.GOMAXPROCS(0)
	}
	if dl.rateDelay > 0 {
		dl.ticker = *time.NewTicker(dl.rateDelay)
	}
	dl.logPrefix = fmt.Sprintf("downloader(%p): ", dl)
	return dl
}

func (dl *downloader) Run(ctx context.Context,
	creator Creator,
	input <-chan Request,
	output chan<- Downloaded) error {
	dl.log(0, "Run starting: concurrency: %v", dl.concurrency)

	var grp errgroup.T
	for i := 0; i < dl.concurrency; i++ {
		i := i
		grp.Go(func() error {
			return dl.runner(ctx, i, creator, dl.progressCh, input, output)
		})
	}
	err := grp.Wait()

	dl.ticker.Stop()
	close(output)
	if dl.progressCh != nil && dl.progressClose {
		close(dl.progressCh)
	}
	dl.log(0, "Run done: err: %v", err)
	return err
}

func (dl *downloader) updateDue() bool {
	dl.progressMu.Lock()
	defer dl.progressMu.Unlock()
	if time.Now().After(dl.progressLast.Add(dl.progressInterval)) {
		dl.progressLast = time.Now()
		return true
	}
	return false
}

func (dl *downloader) updateProgess(downloaded, outstanding int) {
	ndownloaded := atomic.AddInt64(&dl.downloaded, int64(downloaded))
	if dl.progressCh != nil && dl.updateDue() {
		select {
		case dl.progressCh <- DownloadProgress{
			Downloaded:  ndownloaded,
			Outstanding: int64(outstanding),
		}:
		default:
		}
	}
}

func (dl *downloader) runner(ctx context.Context, id int, creator Creator, progress chan<- DownloadProgress,
	input <-chan Request,
	output chan<- Downloaded) error {

	for {
		var request Request
		var ok bool
		dl.log(1, "id: %v: waiting for input", id)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case request, ok = <-input:
			if !ok {
				return nil
			}
		}
		if len(request.Names) == 0 {
			// ignore empty requests.
			continue
		}
		dl.log(1, "id: %v: downloading: #%v objects", id, len(request.Names))
		downloaded, err := dl.downloadObjects(ctx, id, creator, request)
		dl.log(0, "id: %v: downloaded: #%v objects, err: %v", id, len(downloaded.Downloads), err)
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

func (dl *downloader) downloadObjects(ctx context.Context, id int, creator Creator, request Request) (Downloaded, error) {
	download := Downloaded{
		Request:   request,
		Container: creator.Container(),
		Downloads: make([]DownloadStatus, 0, len(request.Names)),
	}
	for _, name := range request.Names {
		dl.log(1, "id: %v: downloading: %v", id, name)
		status, err := dl.downloadObject(ctx, creator, request.Container, name)
		dl.log(1, "id: %v: downloaded: %v, err: %v", id, name, err)
		if err != nil {
			return download, err
		}
		download.Downloads = append(download.Downloads, status)
	}
	return download, nil
}

func (dl *downloader) downloadObject(ctx context.Context, creator Creator, container fs.FS, name string) (DownloadStatus, error) {
	status := DownloadStatus{}
	if dl.ticker.C == nil {
		select {
		case <-ctx.Done():
			return status, ctx.Err()
		default:
		}
	} else {
		select {
		case <-ctx.Done():
			return status, ctx.Err()
		case <-dl.ticker.C:
		}
	}
	delay := dl.backOffStart
	steps := 0
	for {
		rd, err := container.Open(name)
		status.Retries = steps
		status.Err = err
		if err != nil {
			if !errors.Is(err, dl.backOffErr) || steps >= dl.backoffSteps {
				return status, nil
			}
			select {
			case <-ctx.Done():
				return status, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			steps++
			continue
		}
		wr, ni, err := creator.New(name)
		if err != nil {
			status.Err = err
			return status, nil
		}
		if _, err := io.Copy(wr, rd); err != nil {
			status.Err = err
			return status, nil
		}
		status.Name = ni
		return status, nil
	}
}
