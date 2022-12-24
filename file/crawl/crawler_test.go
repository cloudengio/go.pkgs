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
	// test with a fanout ...
	fanOut int
	names  map[string]bool
}

func (e *extractor) Extract(ctx context.Context, downloaded download.Downloaded) []crawl.Request {
	e.Lock()
	defer e.Unlock()
	outlinks := &crawlRequest{container: downloaded.Container}
	for _, dlr := range downloaded.Downloads {
		if e.names[dlr.Name] {
			continue
		}
		outlinks.names = append(outlinks.names, dlr.Name)
		e.names[dlr.Name] = true
	}
	//fmt.Printf("test extractor return # outlinks: %v\n", len(outlinks.Names))
	return []crawl.Request{outlinks}
}

func crawlItems(ctx context.Context, nItems int, input chan<- crawl.Request, reader fs.FS) {
	for i := 0; i < nItems; i++ {
		select {
		case input <- &crawlRequest{container: reader, names: []string{fmt.Sprintf("%v", i)}}:
		case <-ctx.Done():
			break
		}
	}
	close(input)
}

func TestCrawler(t *testing.T) {
	ctx := context.Background()
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan crawl.Request, 10)
	output := make(chan download.Downloaded, 10)
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)
	progressCh := make(chan download.Progress, 1)

	downloader := download.New(
		download.WithProgress(time.Millisecond, progressCh, false),
		download.WithNumDownloaders(1))

	fmt.Printf("object downloader: %p\n", downloader)
	crawler := crawl.New(crawl.WithNumExtractors(1),
		crawl.WithCrawlDepth(1))

	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(3)

	outlinks := &extractor{names: map[string]bool{}}

	go func() {
		errCh <- crawler.Run(ctx, outlinks, downloader, writeFS, input, output)
		wg.Done()
	}()

	go func() {
		crawlItems(ctx, 1000, input, readFS)
		wg.Done()
	}()

	crawled := []download.Downloaded{}
	total := 0
	go func() {
		for outs := range output {
			crawled = append(crawled, outs)
			total += len(outs.Downloads)
			//fmt.Printf("test received: %v/%v\n", len(outs.Downloads), total)
		}
		fmt.Printf("output: done...\n")
		wg.Done()
	}()

	// Need to merge these?
	for {
		select {
		case p := <-progressCh:
			fmt.Printf("object progress: %v\n", p)
		}
	}
	wg.Wait()

}
