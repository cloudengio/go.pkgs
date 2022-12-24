// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

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

func (cr *crawler) logf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("crawler(%p): ", cr)+strings.TrimSuffix(format, "\n")+"\n", args...)

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
	output chan<- download.Downloaded) error {

	errCh := make(chan error, 1)
	dlIn := make(chan download.Request, cap(input))
	dlOut := make(chan download.Downloaded, cap(output))

	go func() {
		// Returns when dlIn is closed, the downloader will close
		// dlOut when all in flight downloads are complete.
		errCh <- downloader.Run(ctx, writeFS, dlIn, dlOut)
	}()

	// Returns when input is closed and any buffered requests have been
	// forwarded to the downloader.
	go cr.forwardRequests(ctx, input, dlIn)

	extractorIn := make(chan download.Downloaded, cap(output))
	extractorStop := make(chan struct{})

	// Returns when dlOut is closed.
	go cr.downloadsTee(ctx, dlOut, output, extractorIn)

	var grp errgroup.T
	for i := 0; i < cr.concurrency; i++ {
		i := i
		grp.Go(func() error {
			cr.runExtractor(ctx, i, extractorStop, extractor, extractorIn, dlIn)
			return nil
		})
	}

	if err := grp.Wait(); err != nil {
		close(output)
		return err
	}

	return nil
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
func (cr *crawler) downloadsTee(ctx context.Context, input <-chan download.Downloaded, outputA, outputB chan<- download.Downloaded) {
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
		var a, b bool
		for !a && !b {
			select {
			case <-ctx.Done():
				return
			case outputA <- downloaded:
				a = true
			case outputB <- downloaded:
				b = true
			}
		}
	}
}

func (cr *crawler) runExtractor(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	dlOut <-chan download.Downloaded,
	dlIn chan<- download.Request) {
	for {
		nlinks, done := cr.handleOutlinks(ctx, id, doneCh, outlinks, dlOut, dlIn)
		cr.logf("runExtractor: links extracted: %v, done: %v", nlinks, done)
		if done {
			return
		}
	}
}

func (cr *crawler) handleOutlinks(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	dlOut <-chan download.Downloaded,
	dlIn chan<- download.Request) (int, bool) {

	// Wait for newly downloaded items.
	var downloaded download.Downloaded
	var ok bool
	select {
	case <-ctx.Done():
		return 0, true
	case <-doneCh:
		return 0, true
	case downloaded, ok = <-dlOut:
		if !ok {
			return 0, true
		}
	}
	// Extract outlinks and add them to the downloader's queue.
	extracted := outlinks.Extract(ctx, downloaded)
	nlinks := 0
	for _, outlinks := range extracted {
		if len(outlinks.Names()) == 0 {
			continue
		}
		crawlRequest, ok := downloaded.Request.(Request)
		if !ok {
			continue
		}
		crawlRequest.IncDepth()
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

/*
	cr.log(0, "Run starting")

	downloaderErrCh := make(chan error, 1)
	outlinkdownloaderErrCh := make(chan error, 1)
	downloadOutput := make(chan Downloaded, cap(output))
	extractorInput := make(chan Downloaded, cap(output))
	outlinkDownloadInput := make(chan Request, cap(input))
	outlinkDownloadOutput := make(chan Downloaded, cap(output))
	extractorStop := make(chan struct{})

	// primary downloader.
	go func() {
		// returns when input is closed.
		err := downloader.Run(ctx, creator, input, downloadOutput)
		cr.log(1, "downloader.Run: error: %v", err)
		downloaderErrCh <- err
	}()

	fmt.Printf("download output ch: %v\n", downloadOutput)
	// outlink downloader.
	go func() {
		// returns when input is closed.
		err := outlinkDownloader.Run(ctx, creator, outlinkDownloadInput, outlinkDownloadOutput)
		cr.log(1, "outlink downloader.Run: error: %v", err)
		outlinkdownloaderErrCh <- err
	}()

	go func() {
		cr.downloadsTee(ctx, downloadOutput, output, extractorInput)
		cr.log(1, "output forwarding goroutine done")
	}()

	var grp errgroup.T
	for i := 0; i < cr.concurrency; i++ {
		i := i
		grp.Go(func() error {
			cr.runExtractor(ctx, i, extractorStop, extractor,
				extractorInput, outlinkDownloadInput)
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		return err
	}
	return nil

	/*
		inputClosed := make(chan struct{})
		downloadInput := make(chan Request, cap(input))
		downloadOutput := make(chan Downloaded, cap(output))
		extractorStop := make(chan struct{})
		downloaderErrCh := make(chan error, 1)

		/*
		   	go func() {
		   		// forwardRequests will return when input is closed.
		   		cr.forwardRequests(ctx, input, downloadInput)
		   		// Signal that there is no more user input.
		   		close(inputClosed)
		   		cr.log(1, "input forwarding goroutine done")
		   	}()

		   	go func() {
		   		// downloadsTee will return when downloadOutput is closed.
		   		cr.downloadsTee(ctx, downloadOutput, output, extractorInput)
		   		cr.log(1, "output forwarding goroutine done")
		   	}()

		   	go func() {
		   		// Run will return when downloadInput is closed and when it has
		   		// processed all outstanding downloads it will close downloadOutput.
		   		downloaderErrCh <- downloader.Run(ctx, creator, downloadInput, downloadOutput)
		   	}()

		   var grp errgroup.T

		   	for i := 0; i < cr.concurrency; i++ {
		   		i := i
		   		grp.Go(func() error {
		   			cr.runExtractor(ctx, i, extractorStop, extractor, extractorInput, downloadInput)
		   			return nil
		   		})
		   	}

		   select {
		   case <-ctx.Done():

		   	// Need to clean up all goroutines.
		   	// It's left to the user to close the input channel.
		   	close(extractorStop)
		   	close(output)
		   	return ctx.Err()

		   case <-inputClosed:

		   		cr.log(1, "input closed, drain remaining requests/outlinks")
		   	}

		   // There are no more new external/user crawl requests to processed.
		   // However, there are still crawl requests in the process of being
		   // downloaded, having links extracted and those downloaded etc.

		   // First, stop all of the currently running extractors.
		   close(extractorStop)

		   	if err := grp.Wait(); err != nil {
		   		close(output)
		   		return err
		   	}

		   cr.drain(ctx, extractor, downloadOutput, output, downloadInput)

		   close(output)
		   err := <-downloaderErrCh
		   cr.log(0, "Run done: err: %v", err)
		   return err
*/
/*}

// Drain any remaining downloaded documents, that is, continuing downloading
// and extracting until there are no outlinks left. Determining when there
// are no outlinks is awkward.
func (cr *crawler) drain(ctx context.Context, outlinks Outlinks,
	downloadOutput <-chan Downloaded,
	output chan<- Downloaded,
	downloadInput chan<- Request) {

	doneCh := make(chan struct{})
	closed := false
	for {
		nlinks, done := cr.handleOutlinks(ctx, -1, doneCh, outlinks, downloadOutput, downloadInput)
		if done {
			break
		}
		if nlinks == 0 && !closed {
			close(downloadInput)
			closed = true
		}
	}
	return
}

func (cr *crawler) forwardRequests(ctx context.Context, input <-chan Request, output chan<- Request) {
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
func (cr *crawler) downloadsTee(ctx context.Context, input <-chan Downloaded, outputA, outputB chan<- Downloaded) {
	for {
		var downloaded Downloaded
		var ok bool
		select {
		case <-ctx.Done():
			return
		case downloaded, ok = <-input:
			if !ok {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		case outputA <- downloaded:
		}
		select {
		case <-ctx.Done():
			return
		case outputB <- downloaded:
		}
	}
}

func (cr *crawler) runExtractor(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	downloaderOutput <-chan Downloaded,
	downloaderInput chan<- Request) {
	for {
		nlinks, done := cr.handleOutlinks(ctx, id, doneCh, outlinks, downloaderOutput, downloaderInput)
		cr.log(0, "runExtractor: links extracted: %v, done: %v", nlinks, done)
		if done {
			return
		}
	}
}

func (cr *crawler) handleOutlinks(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	downloaderOutput <-chan Downloaded,
	downloaderInput chan<- Request) (int, bool) {

	// Wait for newly downloaded items.
	var downloaded Downloaded
	var ok bool
	select {
	case <-ctx.Done():
		return 0, true
	case <-doneCh:
		return 0, true
	case downloaded, ok = <-downloaderOutput:
		if !ok {
			return 0, true
		}
	}
	// Extract outlinks and add them to the downloader's queue.
	extracted := outlinks.Extract(ctx, downloaded)
	nlinks := 0
	for _, outlinks := range extracted {
		if len(outlinks.Names) == 0 {
			continue
		}
		outlinks.Depth = downloaded.Request.Depth + 1
		if outlinks.Depth > cr.depth {
			continue
		}
		nlinks += len(outlinks.Names)
		select {
		case <-ctx.Done():
			return 0, true
		case downloaderInput <- outlinks:
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
*/
