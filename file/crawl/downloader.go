// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/sync/errgroup"
)

type loggerOpts struct {
	logEnabled bool
	logLevel   int
	logOut     io.Writer
	logPrefix  string
}

func (l *loggerOpts) log(level int, format string, args ...interface{}) {
	if l.logEnabled && l.logLevel >= level {
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

func WithDownloadProgress(interval time.Duration, ch chan<- DownloadProgress) DownloaderOption {
	return func(o *downloaderOptions) {
		o.progressInterval = interval
		o.progressCh = ch
	}
}

func WithDownloadLogging(level uint32, out io.Writer) DownloaderOption {
	return func(o *downloaderOptions) {
		o.logEnabled = true
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
	input <-chan []Request,
	output chan<- []Downloaded) error {
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
	if dl.progressCh != nil {
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
	input <-chan []Request,
	output chan<- []Downloaded) error {

	for {
		var items []Request
		var ok bool
		dl.log(1, "id: %v: waiting for input", id)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case items, ok = <-input:
			if !ok {
				return nil
			}
		}
		dl.log(1, "id: %v: downloading: #%v objects", id, len(items))
		fetched, err := dl.fetchItems(ctx, id, creator, items)
		dl.log(0, "id: %v: downloaded: #%v objects, err: %v", id, len(fetched), err)
		if err != nil {
			return err
		}
		dl.updateProgess(len(fetched), len(input))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- fetched:
		}
		dl.log(1, "id: %v: objects downloaded: #%v", id, len(fetched))
	}
}

func (dl *downloader) fetchItems(ctx context.Context, id int, creator Creator, items []Request) ([]Downloaded, error) {
	fetched := make([]Downloaded, 0, len(items))
	for _, item := range items {
		dl.log(1, "id: %v: downloading: %v", id, item.Name)
		item, err := dl.fetchItem(ctx, creator, item)
		dl.log(1, "id: %v: downloaded: %v, err: %v", id, item.Name, err)
		if err != nil {
			return fetched, nil
		}
		fetched = append(fetched, item)
	}
	return fetched, nil
}

func (dl *downloader) fetchItem(ctx context.Context, creator Creator, item Request) (Downloaded, error) {
	if dl.ticker.C == nil {
		select {
		case <-ctx.Done():
			return Downloaded{}, ctx.Err()
		default:
		}
	} else {
		select {
		case <-ctx.Done():
			return Downloaded{}, ctx.Err()
		case <-dl.ticker.C:
		}
	}
	delay := dl.backOffStart
	steps := 0
	dlr := Downloaded{Request: item.Object}
	for {
		rd, err := item.Container.Open(item.Name)
		dlr.Retries = steps
		dlr.Err = err
		if err != nil {
			if !errors.Is(err, dl.backOffErr) || steps >= dl.backoffSteps {
				return dlr, nil
			}
			select {
			case <-ctx.Done():
				return Downloaded{}, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			steps++
			continue
		}
		wr, ni, err := creator.New(item.Name)
		if err != nil {
			dlr.Err = err
			return dlr, nil
		}
		if _, err := io.Copy(wr, rd); err != nil {
			dlr.Err = err
			return dlr, nil
		}
		dlr.Object = ni
		return dlr, nil
	}
}
