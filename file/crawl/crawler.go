// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/file/download"
	"cloudeng.io/sync/errgroup"
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

type downloaderState struct {
	dl         download.T
	requests   chan download.Request
	downloaded chan download.Downloaded
}

func (cr *crawler) Run(ctx context.Context,
	factory DownloaderFactory,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	output chan<- Crawled) error {

	downloaders, dlgrp := cr.createAndRunDownloaders(ctx, factory, writeFS)

	if cr.depth == 0 {
		var errs errors.M
		reqs, dld := downloaders[0].requests, downloaders[0].downloaded
		err := cr.crawlAtDepth(ctx, 0, extractor, writeFS, input, dld, reqs, nil, output)
		errs.Append(err)
		errs.Append(dlgrp.Wait())
		close(output)
		return errs.Err()
	}

	crlgrp := errgroup.T{}
	nextInput := input
	fmt.Printf("input ch: %v\n", nextInput)
	for depth := 0; depth <= cr.depth; depth++ {
		reqs, dld := downloaders[depth].requests, downloaders[depth].downloaded
		var nextReqs chan download.Request
		if depth < cr.depth {
			nextReqs = downloaders[depth+1].requests
		}
		// capture current values for use by the closure below.
		ni := nextInput
		depth := depth
		crlgrp.Go(func() error {
			return cr.crawlAtDepth(ctx, depth, extractor, writeFS, ni, dld, reqs, nextReqs, output)
		})
		nextInput = nextReqs
	}

	var errs errors.M

	fmt.Printf("# downloaders: %v\n", len(downloaders))

	/*	time.Sleep(time.Second)
		gs, _ := goroutines.Get()
		fmt.Printf("RUNNING... %v\n", goroutines.Format(gs...))*/

	errs.Append(dlgrp.Wait())
	errs.Append(crlgrp.Wait())
	return errs.Err()
}

func (cr *crawler) createAndRunDownloaders(ctx context.Context, factory DownloaderFactory, writeFS file.WriteFS) ([]*downloaderState, *errgroup.T) {
	downloaders := make([]*downloaderState, cr.depth+1)
	for i := range downloaders {
		dl, reqs, dled := factory(ctx, i)
		downloaders[i] = &downloaderState{
			dl:         dl,
			requests:   reqs,
			downloaded: dled,
		}
	}
	grp := &errgroup.T{}
	for _, dls := range downloaders {
		dls := dls
		grp.Go(func() error {
			return dls.dl.Run(ctx, writeFS, dls.requests, dls.downloaded)
		})
	}
	return downloaders, grp
}

func (cr *crawler) crawlAtDepth(ctx context.Context,
	depth int,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	dlOutput <-chan download.Downloaded,
	dlInput, dlExtractedInput chan<- download.Request,
	output chan<- Crawled) error {

	fmt.Printf("crawl at depth: %v: starting\n", depth)
	defer fmt.Printf("crawl at depth: %v: done\n", depth)
	wg := &sync.WaitGroup{}
	wg.Add(4)
	go func() {
		pipe(ctx, "downloader input", depth, input, dlInput)
		close(dlInput)
		wg.Done()
	}()

	extractorErrCh := make(chan error, 1)

	ep := newExtractorPool(extractor, depth, cr.concurrency)

	epIn := make(chan download.Downloaded, cap(output))
	epOut := make(chan Crawled, cap(output))

	crawlCompleteCh := make(chan struct{})

	go func() {
		// Pipe output of download pool to extractor pool.
		pipe(ctx, "downloader output", depth, dlOutput, epIn)
		// The crawl is complete when dlOutput is closed.
		close(crawlCompleteCh)
		wg.Done()
	}()

	go func() {
		extractorErrCh <- ep.run(ctx, epIn, epOut)
		fmt.Printf("extractors done\n")
		close(epOut)
		wg.Done()
	}()

	go func() {
		cr.handleExtractedLinks(ctx, epOut, dlExtractedInput, output)
		fmt.Printf("handle extracted links done\n")
		wg.Done()
	}()

	fmt.Printf("crawl at %v ... waiting\n", depth)
	<-crawlCompleteCh
	fmt.Printf("crawl complete....\n")
	ep.stop()
	wg.Wait()
	if dlExtractedInput != nil {
		fmt.Printf("CLOSING: %v\n", dlExtractedInput)
		close(dlExtractedInput)
	}
	err := <-extractorErrCh
	return err
}

func pipe[T any](ctx context.Context, purpose string, depth int, inputCh <-chan T, outputCh chan<- T) {
	fmt.Printf("pipe(%v@%v): %v -> %v: %T: starting\n", purpose, depth, inputCh, outputCh, inputCh)
	defer fmt.Printf("pipe(%v@%v): %v -> %v: %T: done\n", purpose, depth, inputCh, outputCh, inputCh)
	for {
		var in T
		var ok bool
		select {
		case <-ctx.Done():
			return
		case in, ok = <-inputCh:
			fmt.Printf("pipe(%v@%v): <- : %v: %T\n", purpose, depth, inputCh, in)
			if !ok {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case outputCh <- in:
			fmt.Printf("pipe(%v@%v): -> : %v: %T (%v/%v)\n", purpose, depth, outputCh, in, len(outputCh), cap(outputCh))
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
			fmt.Printf("crawled... %v\n", len(crawled.Downloads))
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
