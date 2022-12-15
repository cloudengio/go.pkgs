// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/sync/errgroup"
)

type GenericOption func(*genericOptions)

type genericOptions struct {
	rateDelay        time.Duration
	backOffErr       error
	backOffStart     time.Duration
	backoffSteps     int
	concurrency      int
	progressInterval time.Duration
	progressCh       chan<- Progress
}

type genericCrawler struct {
	genericOptions
	ticker  time.Ticker
	crawled int64

	progressLast time.Time
	progressMu   sync.Mutex
}

func WithRequestsPerMinute(rpm int) GenericOption {
	return func(o *genericOptions) {
		if rpm > 60 {
			o.rateDelay = time.Second / time.Duration(rpm)
			return
		}
		o.rateDelay = time.Minute / time.Duration(rpm)
	}
}

func WithBackoffParameters(err error, start time.Duration, steps int) GenericOption {
	return func(o *genericOptions) {
		o.backOffErr = err
		o.backOffStart = start
		o.backoffSteps = steps
	}
}

func WithConcurrency(concurrency int) GenericOption {
	return func(o *genericOptions) {
		o.concurrency = concurrency
	}
}

func WithProgress(interval time.Duration, ch chan<- Progress) GenericOption {
	return func(o *genericOptions) {
		o.progressInterval = interval
		o.progressCh = ch
	}
}

func NewGeneric(opts ...GenericOption) T {
	hc := &genericCrawler{}
	for _, opt := range opts {
		opt(&hc.genericOptions)
	}
	if hc.concurrency == 0 {
		hc.concurrency = runtime.GOMAXPROCS(0)
	}
	if hc.rateDelay > 0 {
		hc.ticker = *time.NewTicker(hc.rateDelay)
	}
	return hc
}

func (gc *genericCrawler) Run(ctx context.Context,
	creator Creator,
	progress chan<- Progress,
	input <-chan []Item,
	output chan<- []Crawled) error {
	var grp errgroup.T
	for i := 0; i < gc.concurrency; i++ {
		i := i
		grp.Go(func() error {
			return gc.runner(ctx, i, creator, progress, input, output)
		})
	}
	err := grp.Wait()
	gc.ticker.Stop()
	close(output)
	if gc.progressCh != nil {
		close(gc.progressCh)
	}
	return err
}

func (gc *genericCrawler) updateDue() bool {
	gc.progressMu.Lock()
	defer gc.progressMu.Unlock()
	if time.Now().After(gc.progressLast.Add(gc.progressInterval)) {
		gc.progressLast = time.Now()
		return true
	}
	return false
}

func (gc *genericCrawler) updateProgess(crawled, outstanding int) {
	ncrawled := atomic.AddInt64(&gc.crawled, int64(crawled))
	if gc.progressCh != nil && gc.updateDue() {
		select {
		case gc.progressCh <- Progress{
			Crawled:     ncrawled,
			Outstanding: int64(outstanding),
		}:
		default:
		}
	}
}

func (gc *genericCrawler) runner(ctx context.Context, id int, creator Creator, progress chan<- Progress,
	input <-chan []Item,
	output chan<- []Crawled) error {

	for {
		var items []Item
		var ok bool
		select {
		case <-ctx.Done():
			return ctx.Err()
		case items, ok = <-input:
			if !ok {
				return nil
			}
		}
		crawled, err := gc.fetchItems(ctx, creator, items)
		if err != nil {
			return err
		}
		gc.updateProgess(len(crawled), len(input))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- crawled:
		}
	}
}

func (gc *genericCrawler) fetchItems(ctx context.Context, creator Creator, items []Item) ([]Crawled, error) {
	crawled := make([]Crawled, 0, len(items))
	for _, item := range items {
		item, err := gc.fetchItem(ctx, creator, item)
		if err != nil {
			return crawled, nil
		}
		crawled = append(crawled, item)
	}
	return crawled, nil
}

func (gc *genericCrawler) fetchItem(ctx context.Context, creator Creator, item Item) (Crawled, error) {
	if gc.ticker.C == nil {
		select {
		case <-ctx.Done():
			return Crawled{}, ctx.Err()
		default:
		}
	} else {
		select {
		case <-ctx.Done():
			return Crawled{}, ctx.Err()
		case <-gc.ticker.C:
		}
	}
	delay := gc.backOffStart
	steps := 0
	for {
		rd, err := item.Container.Open(item.Name)
		if err != nil {
			if !errors.Is(err, gc.backOffErr) || steps >= gc.backoffSteps {
				return Crawled{Item: item, Retries: steps, Err: err}, nil
			}
			select {
			case <-ctx.Done():
				return Crawled{}, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			steps++
			continue
		}
		wr, ni, err := creator.New(item.Name)
		if err != nil {
			return Crawled{Item: item, Retries: steps, Err: err}, nil
		}
		if _, err := io.Copy(wr, rd); err != nil {
			return Crawled{Item: item, Retries: steps, Err: err}, nil
		}

		return Crawled{Item: ni, Retries: steps}, nil
	}
}
