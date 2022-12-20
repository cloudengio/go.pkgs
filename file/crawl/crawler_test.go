// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/filetestutil"
)

type extractor struct {
	sync.Mutex
	names map[string]bool
}

func (e *extractor) Extract(ctx context.Context, downloaded crawl.Downloaded) []crawl.Request {
	e.Lock()
	defer e.Unlock()
	outlinks := crawl.Request{Container: downloaded.Container}
	for _, dl := range downloaded.Downloads {
		if !e.names[dl.Name] {
			outlinks.Names = append(outlinks.Names, dl.Name)
		}
		e.names[dl.Name] = true
	}
	//fmt.Printf("test extractor return # outlinks: %v\n", len(outlinks.Names))
	return []crawl.Request{outlinks}
}

func TestCrawler(t *testing.T) {
	ctx := context.Background()
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan crawl.Request, 10)
	output := make(chan crawl.Downloaded, 10)
	writeFS := &collector{files: map[string][]byte{}}
	progressCh := make(chan crawl.DownloadProgress, 1)

	downloader := crawl.NewDownloader(
		crawl.WithDownloadProgress(time.Millisecond, progressCh, false),
		//crawl.WithDownloadLogging(1, os.Stdout),
		crawl.WithNumDownloaders(1))

	outlinkDownloader := crawl.NewDownloader(
		crawl.WithDownloadProgress(time.Millisecond, progressCh, false),
		//crawl.WithDownloadLogging(1, os.Stdout),
		crawl.WithNumDownloaders(1))

	fmt.Printf("object downloader: %p, outlinks %p\n", downloader, outlinkDownloader)
	crawler := crawl.New(crawl.WithNumExtractors(1),
		//crawl.WithCrawlerLogging(1, os.Stdout),
		crawl.WithCrawlDepth(1))

	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(3)

	outlinks := &extractor{names: map[string]bool{}}

	go func() {
		errCh <- crawler.Run(ctx, outlinks, downloader, outlinkDownloader, writeFS, input, output)
		wg.Done()
	}()

	go func() {
		crawlItems(ctx, 1000, input, readFS)
		wg.Done()
	}()

	crawled := []crawl.Downloaded{}
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
