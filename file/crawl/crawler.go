// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"runtime"
	"sync"

	"cloudeng.io/file"
	"cloudeng.io/file/download"
)

// Option is used to configure the behaviour of a newly created Crawler.
type Option func(o *options)

type options struct {
	concurrency int
	depth       int
}

func WithNumExtractors(concurrency int) Option {
	return func(o *options) {
		o.concurrency = concurrency
	}
}

func WithCrawlDepth(depth int) Option {
	return func(o *options) {
		o.depth = depth
	}
}

type crawler struct {
	options
}

func New(opts ...Option) T {
	cr := &crawler{}
	for _, opt := range opts {
		opt(&cr.options)
	}
	if cr.concurrency == 0 {
		cr.concurrency = runtime.GOMAXPROCS(0)
	}
	return cr
}

func (cr *crawler) Run(ctx context.Context,
	factory DownloaderFactory,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	output chan<- Crawled) error {

	dlCurrent, dlRequests, dlDownloaded := factory(ctx, 0)

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		dlCurrent.Run(ctx, writeFS, dlRequests, dlDownloaded)
		wg.Done()
	}()

	if cr.depth == 0 {
		err := cr.crawlAtDepth(ctx, 0, extractor, writeFS, input, dlDownloaded, dlRequests, nil, output)
		return err
	}

	/*
		dlNext, dlcnextIn, dlcNextOut := factory(ctx, 1)

		for depth := 0; depth < cr.depth; depth++ {

			if err := cr.crawlAtDepth(ctx, depth, x, extractor, writeFS, input, output); err != nil {
				return err
			}
		}*/
	return nil
}

func (cr *crawler) crawlAtDepth(ctx context.Context,
	depth int,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	dlOutput <-chan download.Downloaded,
	dlInput, dlExtractedInput chan<- download.Request,
	output chan<- Crawled) error {

	wg := &sync.WaitGroup{}
	wg.Add(4)
	go func() {
		pipe(ctx, input, dlInput)
		wg.Done()
	}()

	extractorErrCh := make(chan error, 1)

	ep := newExtractorPool(extractor, depth, cr.concurrency)
	epIn := make(chan download.Downloaded, cap(output))
	epOut := make(chan Crawled, cap(output))
	go func() {
		extractorErrCh <- ep.run(ctx, epIn, epOut)
		wg.Done()
	}()

	go func() {
		// Pipe output of download pool to extractor pool.
		pipe(ctx, dlOutput, epIn)
		wg.Done()
	}()

	go func() {
		cr.handleExtractedLinks(ctx, epOut, dlInput, output)
		wg.Done()
	}()

	ep.stop()
	err := <-extractorErrCh
	wg.Wait()
	return err
}

func pipe[T any](ctx context.Context, inputCh <-chan T, outputCh chan<- T) {
	for {
		var in T
		var ok bool
		select {
		case <-ctx.Done():
			return
		case in, ok = <-inputCh:
			if !ok {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case outputCh <- in:
		}
	}
}

func (cr *crawler) handleExtractedLinks(ctx context.Context, crawledCh <-chan Crawled, downloaderCh chan<- download.Request, output chan<- Crawled) {
	for {
		var crawled Crawled
		var ok bool
		select {
		case <-ctx.Done():
			return
		case crawled, ok = <-crawledCh:
			if ok {
				return
			}
		}
		// Forward crawled downloads to the user.
		select {
		case <-ctx.Done():
			return
		case output <- crawled:
		}
		if downloaderCh == nil {
			continue
		}
		for _, ol := range crawled.Outlinks {
			select {
			case <-ctx.Done():
				return
			case downloaderCh <- ol:
			}
		}
	}
}
