// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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

type Option func(*options)

type options struct {
	rateDelay        time.Duration
	backOffErr       error
	backOffStart     time.Duration
	backoffSteps     int
	concurrency      int
	progressInterval time.Duration
	progressCh       chan<- Progress
	progressClose    bool
}

type downloader struct {
	options
	ticker     time.Ticker
	downloaded int64 // updated using atomic.

	progressMu   sync.Mutex
	progressLast time.Time // GUARDED_BY(progressMu)
}

func WithRequestsPerMinute(rpm int) Option {
	return func(o *options) {
		if rpm > 60 {
			o.rateDelay = time.Second / time.Duration(rpm)
			return
		}
		o.rateDelay = time.Minute / time.Duration(rpm)
	}
}

func WithBackoffParameters(err error, start time.Duration, steps int) Option {
	return func(o *options) {
		o.backOffErr = err
		o.backOffStart = start
		o.backoffSteps = steps
	}
}

func WithNumDownloaders(concurrency int) Option {
	return func(o *options) {
		o.concurrency = concurrency
	}
}

func WithProgress(interval time.Duration, ch chan<- Progress, close bool) Option {
	return func(o *options) {
		o.progressInterval = interval
		o.progressCh = ch
		o.progressClose = close
	}
}

func New(opts ...Option) T {
	dl := &downloader{}
	for _, opt := range opts {
		opt(&dl.options)
	}
	if dl.concurrency == 0 {
		dl.concurrency = runtime.GOMAXPROCS(0)
	}
	if dl.rateDelay > 0 {
		dl.ticker = *time.NewTicker(dl.rateDelay)
	}
	return dl
}

func (dl *downloader) Run(ctx context.Context,
	creator Creator,
	input <-chan Request,
	output chan<- Downloaded) error {

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
		case dl.progressCh <- Progress{
			Downloaded:  ndownloaded,
			Outstanding: int64(outstanding),
		}:
		default:
		}
	}
}

func (dl *downloader) runner(ctx context.Context, id int, creator Creator, progress chan<- Progress, input <-chan Request, output chan<- Downloaded) error {

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
		downloaded, err := dl.downloadObjects(ctx, id, creator, request)
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
		Downloads: make([]Result, 0, len(request.Names())),
	}
	for _, name := range request.Names() {
		status, err := dl.downloadObject(ctx, creator, request.Container(), name)
		if err != nil {
			return download, err
		}
		download.Downloads = append(download.Downloads, status)
	}
	return download, nil
}

func (dl *downloader) downloadObject(ctx context.Context, creator Creator, container fs.FS, name string) (Result, error) {
	result := Result{}
	if dl.ticker.C == nil {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}
	} else {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-dl.ticker.C:
		}
	}
	delay := dl.backOffStart
	steps := 0
	for {
		rd, err := container.Open(name)
		result.Retries = steps
		result.Err = err
		if err != nil {
			if !errors.Is(err, dl.backOffErr) || steps >= dl.backoffSteps {
				return result, nil
			}
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			steps++
			continue
		}
		wr, ni, err := creator.New(name)
		if err != nil {
			result.Err = err
			return result, nil
		}
		if _, err := io.Copy(wr, rd); err != nil {
			result.Err = err
			return result, nil
		}
		result.Name = ni
		return result, nil
	}
}
