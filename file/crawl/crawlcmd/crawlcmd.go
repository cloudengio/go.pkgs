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
	config          Config
	resources       Resources
	displayOutlinks bool
	displayProgress bool
	cache           stores.T
}

// FSFactory is a function that returns a file.FS used to crawl
// a given FS.
type FSFactory func(context.Context) (file.FS, error)

// Resources contains the resources required by the crawler.
type Resources struct {
	// Extractors are used to extract outlinks from crawled documents
	// based on their content type.
	Extractors map[content.Type]outlinks.Extractor
	// CrawlStoreFactories are used to create file.FS instances for
	// the files being crawled based on their scheme.
	CrawlStoreFactories map[string]FSFactory
	// ContentStoreFactory is a function that returns a content.FS used to store
	// the downloaded content.
	NewContentFS func(context.Context, CrawlCacheConfig) (content.FS, error)
}

// NewCrawler creates a new crawler instance using the supplied configuration
// and resources.
func NewCrawler(cfg Config, resources Resources) *Crawler {
	return &Crawler{config: cfg, resources: resources}
}

// Run runs the crawler.
func (c *Crawler) Run(ctx context.Context,
	displayOutlinks, displayProgress bool) error {
	cfs, err := c.resources.NewContentFS(ctx, c.config.Cache)
	if err != nil {
		return fmt.Errorf("failed to create content store: %v: %v", c.config.Cache, err)
	}
	if err := c.config.Cache.PrepareDownloads(ctx, cfs); err != nil {
		return fmt.Errorf("failed to initialize crawl cache: %v: %v", c.config.Cache, err)
	}
	c.displayOutlinks = displayOutlinks
	c.displayProgress = displayProgress
	c.cache = stores.New(cfs, c.config.Cache.Concurrency)
	downloads, _ := c.config.Cache.Paths()
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
	seedsByScheme, rejected := c.config.SeedsByScheme(cloudpath.DefaultMatchers)
	if len(rejected) > 0 {
		return fmt.Errorf("unable to determine a file system for seeds: %v", rejected)
	}

	requests, err := c.config.CreateSeedCrawlRequests(ctx, c.resources.CrawlStoreFactories, seedsByScheme)
	if err != nil {
		return err
	}

	var progressCh chan download.Progress
	if c.displayProgress {
		progressCh = make(chan download.Progress, 100)
		go displayProgress(ctx, c.config.Name, progressCh)
	}

	dlFactory := c.config.Download.NewFactory(progressCh)

	reqCh, crawledCh := c.config.Download.Depth0Chans()

	extractorErrCh := make(chan outlinks.Errors, 100)

	crawler := crawl.New(crawl.WithNumExtractors(c.config.NumExtractors),
		crawl.WithCrawlDepth(c.config.Depth))

	linkProcessor, err := c.config.NewLinkProcessor()
	if err != nil {
		return fmt.Errorf("failed to compile link processing rules: %v", err)
	}

	extractorRegistry, err := c.config.ExtractorRegistry(c.resources.Extractors)
	if err != nil {
		return fmt.Errorf("failed to create extractor registry: %v", err)
	}

	extractor := outlinks.NewExtractors(extractorErrCh, linkProcessor, extractorRegistry)

	var errs errors.M
	var wg sync.WaitGroup
	wg.Add(3)

	go func(ch chan crawl.Crawled) {
		errs.Append(c.saveCrawled(ctx, downloads, c.config.Name, ch))
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
	sharder := path.NewSharder(
		path.WithSHA1PrefixLength(c.config.Cache.ShardingPrefixLen))
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
