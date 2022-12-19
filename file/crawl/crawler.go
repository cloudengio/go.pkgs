// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"cloudeng.io/sync/errgroup"
)

type CrawlerOption func(o *crawlerOptions)

type crawlerOptions struct {
	loggerOpts
	concurrency int
}

func WithNumExtractors(concurrency int) DownloaderOption {
	return func(o *downloaderOptions) {
		o.concurrency = concurrency
	}
}

func WithCrawlerLogging(level uint32, out io.Writer) CrawlerOption {
	return func(o *crawlerOptions) {
		o.logEnabled = true
		o.logLevel = int(level)
		o.logOut = out
	}
}

func New(opts ...CrawlerOption) T {
	cr := &crawler{}
	for _, opt := range opts {
		opt(&cr.crawlerOptions)
	}
	if cr.concurrency == 0 {
		cr.concurrency = runtime.GOMAXPROCS(0)
	}
	cr.logPrefix = fmt.Sprintf("crawler(%p): ", cr)
	return cr
}

type crawler struct {
	crawlerOptions
}

func (cr *crawler) Run(ctx context.Context,
	extractor Outlinks,
	downloader Downloader,
	creator Creator,
	input <-chan []Request,
	output chan<- []Downloaded) error {

	cr.log(0, "Run starting")

	userInputDone := make(chan struct{})
	downloadInput := make(chan []Request, cap(input))
	downloadOutput := make(chan []Downloaded, cap(output))
	extractorStop := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		// Forward passes all user crawl requests to the downloaded, it will
		// return when input is closed.
		cr.forward(ctx, input, downloadInput)
		// Signal that there is no more user input.
		close(userInputDone)
		fmt.Printf("forward DONE.......\n")
	}()

	go func() {
		// Run will return when downloadInput is closed, it will then close downloadOutput.
		errCh <- downloader.Run(ctx, creator, downloadInput, downloadOutput)
	}()

	var grp errgroup.T
	for i := 0; i < cr.concurrency; i++ {
		i := i
		grp.Go(func() error {
			cr.runExtractor(ctx, i, extractorStop, extractor, downloadOutput, output, downloadInput)
			return nil
		})
	}

	/*
		select {
		case <-ctx.Done():
			// Need to clean up all goroutines.
			// It's left to the user to close the input channel.
			close(extractorStop)
			close(output)
			return ctx.Err()
		case <-userInputDone:
		}

		// Close the done channel to signal to the extractor goroutines that
		// they should finish.
		close(extractorStop)
		if err := grp.Wait(); err != nil {
			return err
		}

		// Extractors are no longer running, but there may be some more downloaded
		// documents in the process of being downloaded or being sent over
		// the downloader's output channel.

		if err := cr.drain(ctx, extractor, downloadInput, downloadOutput, output); err != nil {
			return err
		}
		fmt.Printf("closed download input...\n")
		close(downloadInput)*/

	close(output)
	err := <-errCh
	cr.log(0, "Run done: err: %v", err)
	return err
}

// Drain any remaining downloaded documents, that is, continuing downloading
// and extracting until the extractor returns no links.
func (cr *crawler) drain(ctx context.Context, outlinks Outlinks, downloadInput chan<- []Request, downloadOutput <-chan []Downloaded, output chan<- []Downloaded) error {
	var extracted []Request
	for remaining := range downloadOutput {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case output <- remaining:
		case downloadInput <- extracted:
		}
		extracted = outlinks.Extract(ctx, remaining)
		if len(extracted) == 0 {
			return nil
		}
	}
	return nil
}

func (cr *crawler) forward(ctx context.Context, input <-chan []Request, output chan<- []Request) {
	// Forward all client inputs directly to the downloader.
	for req := range input {
		select {
		case <-ctx.Done():
			return
		case output <- req:
		}
	}
}

func (cr *crawler) runExtractor(ctx context.Context,
	id int,
	doneCh chan struct{},
	outlinks Outlinks,
	downloadedCh <-chan []Downloaded,
	outputCh chan<- []Downloaded,
	downloaderInput chan<- []Request) {

	for {
		cr.log(1, "id: %v: extractor: waiting for downloads", id)
		// Wait for newly downloaded items.
		var downloaded []Downloaded
		var ok bool
		select {
		case <-ctx.Done():
			return
		case <-doneCh:
			return
		case downloaded, ok = <-downloadedCh:
			if !ok {
				return
			}
		}
		// Extract outlinks and add them to the downloader's queue.
		extracted := outlinks.Extract(ctx, downloaded)
		cr.log(0, "id: %v: extractor: outlinks found: #%v, from downloads: #%v", id, len(extracted), len(downloaded))
		select {
		case <-ctx.Done():
			return
		case downloaderInput <- extracted:
		}
	}
}
