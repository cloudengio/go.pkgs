// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/content/stores"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/path"
	"cloudeng.io/path/cloudpath"
)

// Crawler represents a crawler instance and contains global configuration
// information.
type Crawler struct {
	Config
	Extractors      map[content.Type]outlinks.Extractor
	displayOutlinks bool
	displayProgress bool
	fsMap           map[string]FSFactory
	cache           stores.T
	cacheRoot       string
}

// FSFactory is a function that returns a file.FS used to crawl
// a given FS.
type FSFactory func(context.Context) (file.FS, error)

// Run runs the crawler.
func (c *Crawler) Run(ctx context.Context,
	fsMap map[string]FSFactory,
	fs content.FS,
	displayOutlinks, displayProgress bool) error {
	if err := c.Cache.PrepareDownloads(ctx, fs); err != nil {
		return fmt.Errorf("failed to initialize crawl cache: %v: %v", c.Cache, err)
	}
	c.displayOutlinks = displayOutlinks
	c.displayProgress = displayProgress
	c.fsMap = fsMap
	c.cache = stores.New(fs, c.Cache.Concurrency)
	downloads, _ := c.Cache.Paths()
	return c.run(ctx, downloads)
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

func (c *Crawler) run(ctx context.Context, downloads string) error {
	seedsByScheme, rejected := c.SeedsByScheme(cloudpath.DefaultMatchers)
	if len(rejected) > 0 {
		return fmt.Errorf("unable to determine a file system for seeds: %v", rejected)
	}

	requests, err := c.CreateSeedCrawlRequests(ctx, c.fsMap, seedsByScheme)
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

	extractorRegistry, err := c.ExtractorRegistry(c.Extractors)
	if err != nil {
		return fmt.Errorf("failed to create extractor registry: %v", err)
	}

	extractor := outlinks.NewExtractors(extractorErrCh, linkProcessor, extractorRegistry)

	var errs errors.M
	var wg sync.WaitGroup
	wg.Add(3)

	go func(ch chan crawl.Crawled) {
		errs.Append(c.saveCrawled(ctx, downloads, c.Name, ch))
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
				log.Printf("extractor error: %v\n", err)
			}
		}
	}()

	wg.Wait()
	close(extractorErrCh)
	return errs.Err()
}

func (c Crawler) saveCrawled(ctx context.Context, downloads, name string, crawledCh chan crawl.Crawled) error {
	sharder := path.NewSharder(path.WithSHA1PrefixLength(c.Cache.ShardingPrefixLen))
	join := c.cache.FS().Join

	written := 0
	defer func() {
		log.Printf("total written: %v", written)
	}()

	for crawled := range crawledCh {
		if c.displayOutlinks {
			for _, req := range crawled.Outlinks {
				log.Printf("%v\n", strings.Join(crawled.Request.Names(), " "))
				for _, name := range req.Names() {
					log.Printf("\t-> %v\n", name)
				}
			}
		}
		objs := crawl.CrawledObjects(crawled)
		for _, obj := range objs {
			dld := obj.Response
			if dld.Err != nil {
				log.Printf("download error: %v: %v\n", dld.Name, dld.Err)
				continue
			}
			prefix, suffix := sharder.Assign(name + dld.Name)
			prefix = join(downloads, prefix)
			if err := obj.Store(ctx, c.cache, prefix, suffix, content.GOBObjectEncoding, content.GOBObjectEncoding); err != nil {
				log.Printf("failed to write: %v as prefix: %v, suffix: %v: %v\n", dld.Name, prefix, suffix, err)
				continue
			}
			written++
			if written%100 == 0 {
				log.Printf("written: %v", written)
			}
		}
	}
	return nil
}
