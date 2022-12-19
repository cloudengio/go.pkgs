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

/*
for _, dl := range downloaded {
	rdc, err := dl.Container.Open(dl.Name)
	if err != nil {
		cr.extractorErrors.append(dl.Name, err)
		continue
	}
*/

type extractor struct {
	sync.Mutex
	names map[string]bool
}

func (e *extractor) Extract(ctx context.Context, downloaded []crawl.Downloaded) []crawl.Request {
	e.Lock()
	defer e.Unlock()
	outlinks := []crawl.Request{}
	for _, dl := range downloaded {
		if !e.names[dl.Name] {
			outlinks = append(outlinks, crawl.Request{Object: dl.Request})
		}
		e.names[dl.Name] = true
	}
	fmt.Printf("len outlinks: %v\n", len(outlinks))
	return outlinks
}

func TestCrawler(t *testing.T) {
	ctx := context.Background()
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan []crawl.Request, 10)
	output := make(chan []crawl.Downloaded, 10)
	writeFS := &collector{files: map[string][]byte{}}
	progressCh := make(chan crawl.DownloadProgress, 1)

	downloader := crawl.NewDownloader(
		crawl.WithDownloadProgress(time.Millisecond, progressCh),
		crawl.WithNumDownloaders(1),
		crawl.WithNumExtractors(1))
	//		crawl.WithDownloadLogging(1, os.Stdout))

	crawler := crawl.New() //crawl.WithCrawlerLogging(1, os.Stdout))

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
		fmt.Printf("DONE .... crawlItems.....\n")
	}()

	crawled := []crawl.Downloaded{}
	go func() {
		for outs := range output {
			crawled = append(crawled, outs...)
			fmt.Printf("output: %v .. %v\n", len(crawled), len(outs))
		}
		fmt.Printf("output: done...\n")
		wg.Done()
	}()

	for p := range progressCh {
		fmt.Printf("progress: %v\n", p)
	}
	wg.Wait()

}
