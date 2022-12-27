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
	return cr
}

func (cr *crawler) Run(ctx context.Context,
	extractor Outlinks,
	downloader download.T,
	writeFS file.WriteFS,
	input <-chan Request,
	output chan<- Crawled) error {

	errCh := make(chan error, 1)
	dlIn := make(chan download.Request, cap(input))
	dlOut := make(chan download.Downloaded, cap(output))

	teeStop := make(chan struct{})
	extractorStop := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		// The downloader runs until dlIn is closed and any requests
		// buffered in dlIn are downloaded, at which point dlOut is
		// closed. When crawling, dlIn must not be closed until
		// all extracted links have processed to the requested depth.
		errCh <- downloader.Run(ctx, writeFS, dlIn, dlOut)
		fmt.Printf("downloader done\n")
		wg.Done()
	}()

	// Returns when input is closed and any buffered requests have been
	// forwarded to the downloader.
	go func() {
		cr.forwardRequests(ctx, input, dlIn)
		close(extractorStop)
		fmt.Printf("forwarder done\n")
		wg.Done()
	}()

	extractorIn := make(chan Crawled, cap(output))

	// Returns when dlOut is closed.
	go func() {
		cr.downloadsTee(ctx, teeStop, dlOut, output, extractorIn)
		wg.Done()
	}()

	// Extractors return when extractorStop is closed.
	var grp errgroup.T
	for i := 0; i < cr.concurrency; i++ {
		i := i
		grp.Go(func() error {
			cr.runExtractor(ctx, i, extractorStop, extractor, extractorIn, dlIn)
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		// sort out closing conditions...
		close(output)
		return err
	}
	fmt.Printf("extractors done\n")
	close(dlIn)
	close(teeStop)
	wg.Wait()
	close(output)
	return <-errCh
}

func (cr *crawler) forwardRequests(ctx context.Context, input <-chan Request, output chan<- download.Request) {
	for {
		var req Request
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
		case output <- req:
		}
	}
}

// forward downloads to both the user output and the extractor.
func (cr *crawler) downloadsTee(ctx context.Context, doneCh chan struct{}, input <-chan download.Downloaded, userOutput, extractorInput chan<- Crawled) {
	//	var aout, bout int
	for {
		var downloaded download.Downloaded
		var ok bool
		select {
		case <-ctx.Done():
			return
		case downloaded, ok = <-input:
			if !ok {
				return
			}
		}
		crawled := Crawled{
			Request:   downloaded.Request.(Request),
			Container: downloaded.Container,
			Downloads: downloaded.Downloads,
		}
		select {
		case <-ctx.Done():
			return
		case userOutput <- crawled:
		}
		select {
		case <-ctx.Done():
			return
		case extractorInput <- crawled:
			//		case <-doneCh:
		default:
		}
		/*
			var a, b bool
			//		fmt.Printf("\n\nLoop start: a/b: %v %v: %v\n", a, b, aout+bout)
			for !a || !b {
				//			fmt.Printf("Loop before select: a/b: %v %v: %v %v: %v\n", a, b, aout, bout, aout+bout)
				select {
				case <-ctx.Done():
					return
				case userOutput <- downloaded:
					a = true
					aout++
				case extractorInput <- downloaded:
					b = true
					bout++
				}
				fmt.Printf("Loop after select: a/b: %v %v: %v/%v\n", a, b, aout, bout)
			}
			//		fmt.Printf("Loop done: a/b: %v %v: %v\n", aout, bout, aout+bout)
		*/
	}
}

func (cr *crawler) runExtractor(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	dlOut <-chan Crawled,
	dlIn chan<- download.Request) {
	for {
		_, done := cr.handleOutlinks(ctx, id, doneCh, outlinks, dlOut, dlIn)
		if done {
			return
		}
	}
}

func (cr *crawler) handleOutlinks(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	dlOut <-chan Crawled,
	dlIn chan<- download.Request) (int, bool) {

	// Wait for newly downloaded items.
	var crawled Crawled
	var ok bool
	select {
	case <-ctx.Done():
		return 0, true
	case <-doneCh:
		return 0, true
	case crawled, ok = <-dlOut:
		if !ok {
			return 0, true
		}
	}
	// Extract outlinks and add them to the downloader's queue.
	extracted := outlinks.Extract(ctx, crawled)
	nlinks := 0
	for _, outlinks := range extracted {
		if len(outlinks.Names()) == 0 {
			continue
		}
		outlinks.IncDepth()
		if outlinks.Depth() > cr.depth {
			continue
		}
		nlinks += len(outlinks.Names())
		select {
		case <-ctx.Done():
			return 0, true
		case dlIn <- outlinks:
		}
	}
	// Check to see if the extractor should stop.
	select {
	case <-doneCh:
		return nlinks, true
	default:
	}
	return nlinks, false
}
