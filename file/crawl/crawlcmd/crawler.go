// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/glean/crawlindex/config"
	"cloudeng.io/path"
	"cloudeng.io/path/cloudpath"
)

// Crawler represents a crawler instance that contains global configuration
// information.
type Crawler struct {
	Config
	Extractors      func() map[content.Type]outlinks.Extractor
	FSForCrawl      func(config.Crawl) map[string]file.FSFactory
	crawlCachePath  string
	displayOutlinks bool
	displayProgress bool
}

func (c *Crawler) Run(ctx context.Context, root string, displayOutlinks, displayProgress bool) error {
	crawlCache, _, err := c.Cache.Initialize(root)
	if err != nil {
		return fmt.Errorf("failed to initialize crawl cache: %v: %v", c.Cache, err)
	}
	c.crawlCachePath = crawlCache
	c.displayOutlinks = displayOutlinks
	c.displayProgress = displayProgress
	return c.run(ctx)
}

func displayProgress(ctx context.Context, name string, progress <-chan download.Progress) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-progress:
			fmt.Printf(" 16%v: % 8v: % 8v\n", name, p.Downloaded, p.Outstanding)
		}
	}
}

func (c *Crawler) run(ctx context.Context) error {
	seedsByScheme, rejected := c.SeedsByScheme(cloudpath.DefaultMatchers)
	if len(rejected) > 0 {
		return fmt.Errorf("unable to determine a file system for seeds: %v", rejected)
	}

	requests, err := c.CreateSeedCrawlRequests(ctx, c.FSForCrawl(c.Config), seedsByScheme)
	if err != nil {
		return err
	}

	var progressCh chan download.Progress
	if c.displayProgress {
		progressCh = make(chan download.Progress, 100)
		go displayProgress(ctx, c.Name, progressCh)
	}

	dlFactory := c.Download.NewFactory(progressCh)

	reqCh, crawledCh := c.Download.Depth0Chans()

	extractorErrCh := make(chan outlinks.Errors, 100)

	crawler := crawl.New(crawl.WithNumExtractors(c.NumExtractors),
		crawl.WithCrawlDepth(c.Depth))

	linkProcessor, err := c.NewLinkProcessor()
	if err != nil {
		return fmt.Errorf("failed to compile link processing rules: %v", err)
	}

	extractorRegistry, err := c.ExtractorRegistry(c.Extractors())
	if err != nil {
		return fmt.Errorf("failed to create extractor registry: %v", err)
	}

	extractor := outlinks.NewExtractors(extractorErrCh, linkProcessor, extractorRegistry)

	var errs errors.M
	var wg sync.WaitGroup
	wg.Add(3)

	go func(ch chan crawl.Crawled) {
		errs.Append(c.saveCrawled(ctx, c.Name, ch))
		wg.Done()
	}(crawledCh)

	go func() {
		errs.Append(crawler.Run(ctx, dlFactory, extractor, reqCh, crawledCh))
		wg.Done()
	}()

	go func() {
		defer wg.Done()
		defer close(reqCh)
		for _, req := range requests {
			select {
			case <-ctx.Done():
				errs.Append(ctx.Err())
				return
			case reqCh <- req:
			}
		}
	}()

	go func() {
		for err := range extractorErrCh {
			if len(err.Errors) > 0 {
				fmt.Printf("extractor error: %v\n", err)
			}
		}
	}()

	wg.Wait()
	close(extractorErrCh)
	return errs.Err()
}

func (c Crawler) saveCrawled(ctx context.Context, name string, crawledCh chan crawl.Crawled) error {
	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.Cache.ShardingPrefixLen))

	for crawled := range crawledCh {
		if c.displayOutlinks {
			for _, req := range crawled.Outlinks {
				fmt.Printf("%v\n", strings.Join(crawled.Request.Names(), " "))
				for _, name := range req.Names() {
					fmt.Printf("\t-> %v\n", name)
				}
			}
		}

		objs := crawl.CrawledObjects(crawled)
		for _, obj := range objs {
			dld := obj.Response
			if dld.Err != nil {
				fmt.Printf("download error: %v: %v\n", dld.Name, dld.Err)
				continue
			}
			prefix, suffix := sharder.Assign(name + dld.Name)
			path := filepath.Join(c.crawlCachePath, prefix, suffix)
			if err := obj.WriteObjectBinary(path); err != nil {
				fmt.Printf("failed to write: %v as %v: %v\n", dld.Name, path, err)
				continue
			}
			fmt.Printf("%v -> %v\n", dld.Name, path)
		}
	}
	return nil
}
