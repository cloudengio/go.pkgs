// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"fmt"
	"runtime"
	"sync"

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
	if cr.depth == 0 {
		cr.depth = 1
	}
	return cr
}

func (cr *crawler) Run(ctx context.Context,
	factory DownloaderFactory,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	output chan<- Crawled) error {
	for depth := 0; depth < cr.depth; depth++ {
		if err := cr.crawlAtDepth(ctx, depth, factory, extractor, writeFS, input, output); err != nil {
			return err
		}
	}
	return nil
}

func (cr *crawler) crawlAtDepth(ctx context.Context,
	depth int,
	factory DownloaderFactory,
	extractor Outlinks,
	writeFS file.WriteFS,
	input <-chan download.Request,
	output chan<- Crawled) error {
	dl, dlIn, dlOut := factory(ctx)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		cr.pipeInput(ctx, input, dlIn)
		wg.Done()
	}()

}

func (cr *crawler) pipeInput(ctx context.Context, input <-chan download.Request, dlIn chan<- download.Request) {
	for {
		var req download.Request
		var ok bool
		select {
		case <-ctx.Done():
			return
		case req, ok = <-input:
			if !ok {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case dlIn <- req:
		}
	}
}

func (cr *crawler) download(ctx context.Context,
	extractor Outlinks,
	downloader download.T,
	writeFS file.WriteFS,
	input <-chan download.Request,
	output chan<- Crawled) error {

	errCh := make(chan error, 1)
	dlOut := make(chan download.Downloaded, cap(output))

	extractorStop := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		errCh <- downloader.Run(ctx, writeFS, input, dlOut)
		wg.Done()
	}()

	// Extractors return when extractorStop is closed.
	var grp errgroup.T
	for i := 0; i < cr.concurrency; i++ {
		i := i
		grp.Go(func() error {
			cr.runExtractor(ctx, i, extractorStop, extractor, dlOut, output)
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		// sort out closing conditions...
		close(output)
		return err
	}
	fmt.Printf("extractors done\n")
	//	close(dlIn)
	//	close(teeStop)
	wg.Wait()
	close(output)
	return <-errCh
}

func (cr *crawler) runExtractor(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	dlOut <-chan download.Downloaded,
	output chan<- Crawled) {
	for {
		// Wait for newly downloaded items.
		var crawled Crawled
		var ok bool
		select {
		case <-ctx.Done():
			return
		case <-doneCh:
			return
		case crawled.Downloaded, ok = <-dlOut:
			if !ok {
				return
			}
		}
		// Extract outlinks and add them to the downloader's queue.
		extracted := outlinks.Extract(ctx, crawled.Downloaded)
		for _, outlinks := range extracted {
			crawled.Outlinks = append(crawled.Outlinks, outlinks)
		}
		// Check to see if the extractor should stop.
		select {
		case output <- crawled:
		case <-doneCh:
			return
		default:
		}
	}
}
