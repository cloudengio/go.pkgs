// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/sync/synctestutil"
)

func copyDownloadsToFS(ctx context.Context, t *testing.T, crawled []crawl.Crawled) *filetestutil.WriteFS {
	writeFS := filetestutil.NewWriteFS()
	for _, c := range crawled {
		for _, d := range c.Downloads {
			if d.Err != nil {
				continue
			}
			f, err := writeFS.Create(ctx, d.Name, fs.FileMode(0600))
			if err != nil {
				t.Fatal(err)
			}
			rd := bytes.NewBuffer(d.Contents)
			if _, err := io.Copy(f, rd); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
	return writeFS
}

type extractor struct {
	sync.Mutex
	fanOut int
}

func (e *extractor) Extract(ctx context.Context, depth int, downloaded download.Downloaded) []download.Request {
	e.Lock()
	defer e.Unlock()
	outlinks := (*downloaded.Request.(*crawl.SimpleRequest))
	outlinks.Filenames = nil
	outlinks.Depth = depth
	for _, dlr := range downloaded.Downloads {
		for nout := 0; nout < e.fanOut; nout++ {
			outlinks.Filenames = append(outlinks.Filenames, dlr.Name+fmt.Sprintf("-%02v", nout))
		}
	}
	return []download.Request{&outlinks}
}

func issuseCrawlRequests(ctx context.Context, nItems int, input chan<- download.Request, reader file.FS) {
	for i := 0; i < nItems; i++ {
		req := crawl.SimpleRequest{}
		req.FS = reader
		req.Filenames = []string{fmt.Sprintf("%08v", i)}
		select {
		case input <- &req:
		case <-ctx.Done():
			break
		}
	}
	close(input)
}

type dlFactory struct {
	numDownloaders int
}

func (df dlFactory) create(ctx context.Context, depth int) (
	downloader download.T,
	inputCh chan download.Request,
	outputCh chan download.Downloaded) {
	concurrency := 0
	chanCap := 0
	// Use conservative channel capacities to ensure that blocking
	// is encountered within the crawler.
	switch {
	case depth == 0:
		concurrency = 2
		chanCap = 1
	case depth < 6:
		concurrency = depth * 2
		chanCap = depth
	default:
		chanCap = 100
	}
	inputCh = make(chan download.Request, chanCap)
	outputCh = make(chan download.Downloaded, chanCap)
	downloader = download.New(
		download.WithNumDownloaders(concurrency),
		download.WithNumDownloaders(df.numDownloaders))
	return
}

func expectedFilenames(nItems, depth, fanOut int) []string {
	names := []string{}
	for i := 0; i < nItems; i++ {
		names = append(names, fmt.Sprintf("%08v", i))
	}
	prev := names
	for d := 1; d <= depth; d++ {
		extracted := []string{}
		for _, p := range prev {
			for f := 0; f < fanOut; f++ {
				extracted = append(extracted, p+fmt.Sprintf("-%02v", f))
			}
		}
		prev = extracted
		names = append(names, extracted...)
	}
	return names
}

func TestCrawler(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()

	ctx := context.Background()
	src := rand.NewSource(time.Now().UnixMicro())

	nItems := 100
	fanOut := 2 // number of outlinks per download.

	for _, depth := range []int{0, 1, 4} {
		readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 1024))

		inputCh := make(chan download.Request, 10)
		outputCh := make(chan crawl.Crawled, 10)

		crawler := crawl.New(
			crawl.WithNumExtractors(2),
			crawl.WithCrawlDepth(depth))

		errCh := make(chan error, 1)
		wg := &sync.WaitGroup{}
		wg.Add(3)

		outlinks := &extractor{fanOut: fanOut}

		df := &dlFactory{numDownloaders: 1}
		go func() {
			errCh <- crawler.Run(ctx, df.create, outlinks, inputCh, outputCh)
			wg.Done()
		}()

		go func() {
			issuseCrawlRequests(ctx, nItems, inputCh, readFS)
			wg.Done()
		}()

		crawled := []crawl.Crawled{}
		nDownloads := 0
		nOutlinks := 0
		go func() {
			for outs := range outputCh {
				crawled = append(crawled, outs)
				nDownloads += len(outs.Downloads)
				for _, ol := range outs.Outlinks {
					nOutlinks += len(ol.Names())
				}
			}
			wg.Done()
		}()

		wg.Wait()

		if err := <-errCh; err != nil {
			t.Fatal(err)
		}

		expectedDownloads := nItems
		prev := nItems
		for d := 0; d < depth; d++ {
			prev *= fanOut
			expectedDownloads += prev
		}
		expectedOutlinks := expectedDownloads * fanOut

		t.Logf("depth %v, expected downloads: %v, expected outlinks %v", depth, expectedDownloads, expectedOutlinks)

		if got, want := nDownloads, expectedDownloads; got != want {
			t.Errorf("depth %v: got %v, want %v", depth, got, want)
		}

		if got, want := nOutlinks, expectedOutlinks; got != want {
			t.Errorf("depth %v: got %v, want %v", depth, got, want)
		}

		writeFS := copyDownloadsToFS(ctx, t, crawled)

		crawledContents := filetestutil.Contents(writeFS)
		if got, want := len(crawledContents), expectedDownloads; got != want {
			t.Errorf("depth %v: got %v, want %v", depth, got, want)
		}

		expectedDownloadNames := expectedFilenames(nItems, depth, fanOut)
		if got, want := len(expectedDownloadNames), expectedDownloads; got != want {
			t.Errorf("depth %v: got %v, want %v", depth, got, want)
		}
		for _, k := range expectedDownloadNames {
			if _, ok := crawledContents[k]; !ok {
				t.Errorf("%v was not crawled", k)
			}
		}
		if err := filetestutil.CompareFS(readFS, writeFS); err != nil {
			t.Fatal(err)
		}
	}
}
