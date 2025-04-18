// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"runtime"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file/download"
	"cloudeng.io/sync/errgroup"
)

// Option is used to configure the behaviour of a newly created Crawler.
type Option func(o *options)

type options struct {
	concurrency int
	depth       int
}

// WithNumExtractors sets the number of extractors to run.
func WithNumExtractors(concurrency int) Option {
	return func(o *options) {
		o.concurrency = concurrency
	}
}

// WithCrawlDepth sets the depth of the crawl.
func WithCrawlDepth(depth int) Option {
	return func(o *options) {
		o.depth = depth
	}
}

type crawler struct {
	options
}

// New creates a new instance of T that implements a multilevel, concurrent
// crawl. The crawl is implemented as a chain of downloaders and extractors,
// one per depth requested. This allows for concurrency within each level
// of the crawl as well as across each level.
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

type downloaderState struct {
	dl         download.T
	requests   chan download.Request
	downloaded chan download.Downloaded
}

func (cr *crawler) Run(ctx context.Context,
	factory DownloaderFactory,
	extractor Outlinks,
	input <-chan download.Request,
	output chan<- Crawled) error {

	downloaders, dlgrp := cr.createAndRunDownloaders(ctx, factory)

	if cr.depth == 0 {
		var errs errors.M
		reqs, dld := downloaders[0].requests, downloaders[0].downloaded
		err := cr.crawlAtDepth(ctx, 0, extractor, input, dld, reqs, nil, output)
		errs.Append(err)
		errs.Append(dlgrp.Wait())
		close(output)
		return errs.Err()
	}

	crlgrp := errgroup.T{}
	chainedInput := input
	for depth := 0; depth <= cr.depth; depth++ {
		reqs, dld := downloaders[depth].requests, downloaders[depth].downloaded
		currentInput := chainedInput
		var nextInput chan download.Request
		if depth < cr.depth {
			nextInput = make(chan download.Request, cap(reqs))
		}
		crlgrp.Go(func() error {
			return cr.crawlAtDepth(ctx, depth, extractor, currentInput, dld, reqs, nextInput, output)
		})
		chainedInput = nextInput
	}

	var errs errors.M
	errs.Append(dlgrp.Wait())
	errs.Append(crlgrp.Wait())
	close(output)
	return errs.Err()
}

func (cr *crawler) createAndRunDownloaders(ctx context.Context, factory DownloaderFactory) ([]*downloaderState, *errgroup.T) {
	downloaders := make([]*downloaderState, cr.depth+1)
	for i := range downloaders {
		dl, reqs, dled := factory.New(ctx, i)
		downloaders[i] = &downloaderState{
			dl:         dl,
			requests:   reqs,
			downloaded: dled,
		}
	}
	grp := &errgroup.T{}
	for _, dls := range downloaders {
		grp.Go(func() error {
			return dls.dl.Run(ctx, dls.requests, dls.downloaded)
		})
	}
	return downloaders, grp
}

func (cr *crawler) crawlAtDepth(ctx context.Context,
	depth int,
	extractor Outlinks,
	input <-chan download.Request,
	dlOutput <-chan download.Downloaded,
	dlInput, dlExtractedInput chan<- download.Request,
	output chan<- Crawled) error {

	wg := &sync.WaitGroup{}
	wg.Add(4)
	go func() {
		pipe(ctx, input, dlInput)
		close(dlInput)
		wg.Done()
	}()

	extractorErrCh := make(chan error, 1)

	ep := newExtractorPool(extractor, depth, cr.concurrency)

	epIn := make(chan download.Downloaded, cap(output))
	epOut := make(chan Crawled, cap(output))

	go func() {
		// Pipe output of download pool to extractor pool.
		pipe(ctx, dlOutput, epIn)
		close(epIn)
		wg.Done()
	}()

	go func() {
		extractorErrCh <- ep.run(ctx, epIn, epOut)
		close(epOut)
		wg.Done()
	}()

	go func() {
		cr.handleExtractedLinks(ctx, epOut, dlExtractedInput, output)
		wg.Done()
	}()

	wg.Wait()
	if dlExtractedInput != nil {
		close(dlExtractedInput)
	}
	err := <-extractorErrCh
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
			if !ok {
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
