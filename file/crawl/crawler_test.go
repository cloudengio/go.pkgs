// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl_test

import (
	"context"
	"fmt"
	"io/fs"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
	"cloudeng.io/file/filetestutil"
)

type crawlRequest struct {
	container fs.FS
	names     []string
	depth     int
}

func (cr *crawlRequest) Container() fs.FS {
	return cr.container
}

func (cr *crawlRequest) Names() []string {
	return cr.names
}

func (cr crawlRequest) FileMode() fs.FileMode {
	return fs.FileMode(0600)
}

func (cr *crawlRequest) Depth() int {
	return cr.depth
}

func (cr *crawlRequest) IncDepth() {
	cr.depth++
}

type extractor struct {
	sync.Mutex
	fanOut int
}

func (e *extractor) Extract(ctx context.Context, depth int, downloaded download.Downloaded) []download.Request {
	e.Lock()
	defer e.Unlock()
	outlinks := (*downloaded.Request.(*crawlRequest))
	outlinks.names = nil
	for _, dlr := range downloaded.Downloads {
		for nout := 0; nout < e.fanOut; nout++ {
			outlinks.names = append(outlinks.names, dlr.Name+fmt.Sprintf("%v", nout))
		}
		fmt.Printf("outlinks: %v\n", outlinks.names)
	}
	return []download.Request{&outlinks}
}

func issuseCrawlRequests(ctx context.Context, nItems int, input chan<- download.Request, reader fs.FS) {
	for i := 0; i < nItems; i++ {
		select {
		case input <- &crawlRequest{container: reader, depth: 0, names: []string{fmt.Sprintf("%v", i)}}:
		case <-ctx.Done():
			break
		}
	}
	fmt.Printf("sent %v items\n", nItems)
	close(input)
}

type dlFactory struct {
	progressCh     chan download.Progress
	numDownloaders int
}

func (df dlFactory) create(ctx context.Context, depth int) (
	downloader download.T,
	inputCh chan download.Request,
	outputCh chan download.Downloaded) {
	inputCh = make(chan download.Request, 10)
	outputCh = make(chan download.Downloaded, 10)
	downloader = download.New(
		download.WithProgress(time.Millisecond, df.progressCh, false),
		download.WithNumDownloaders(df.numDownloaders))
	return
}

func TestCrawler(t *testing.T) {
	ctx := context.Background()
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))

	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)
	progressCh := make(chan download.Progress, 1)

	inputCh := make(chan download.Request, 10)
	outputCh := make(chan crawl.Crawled, 10)

	crawler := crawl.New(crawl.WithNumExtractors(2), crawl.WithCrawlDepth(2))

	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(3)

	outlinks := &extractor{fanOut: 2}

	df := &dlFactory{
		progressCh:     progressCh,
		numDownloaders: 1,
	}
	go func() {
		errCh <- crawler.Run(ctx, df.create, outlinks, writeFS, inputCh, outputCh)
		wg.Done()
	}()

	nItems := 10
	go func() {
		issuseCrawlRequests(ctx, nItems, inputCh, readFS)
		wg.Done()
	}()

	crawled := []crawl.Crawled{}
	total := 0
	go func() {
		for outs := range outputCh {
			crawled = append(crawled, outs)
			total += len(outs.Downloads)
			fmt.Printf("total/crawled: %v %v\n", total, len(outs.Downloads))
		}
		fmt.Printf("total/crawled: %v %v\n", total, len(crawled))
		wg.Done()
	}()

	// Make sure the progress chan gets closed.
	for range progressCh {
	}
	wg.Wait()

	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if err := filetestutil.CompareFS(readFS, writeFS); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("test done: ... %v: %v\n", total, len(crawled))
	for _, c := range crawled {
		for _, ol := range c.Outlinks {
			fmt.Printf("%v/ %v\n", ol, len(ol.Names()))
		}
	}
	t.Fail()
}
