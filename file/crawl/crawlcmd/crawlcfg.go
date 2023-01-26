// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package crawlcmd provides support for building command line tools for
// crawling. In particular it provides support for managing the configuration
// of a crawl via yaml.
package crawlcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/path/cloudpath"
)

// DownloadFactoryConfig is the configuration for a crawl.DownloaderFactory.
type DownloadFactoryConfig struct {
	DefaultConcurrency       int   `yaml:"default_concurrency"`
	DefaultRequestChanSize   int   `yaml:"default_request_chan_size"`
	DefaultCrawledChanSize   int   `yaml:"default_crawled_chan_size"`
	PerDepthConcurrency      []int `yaml:"per_depth_concurrency"`
	PerDepthRequestChanSizes []int `yaml:"per_depth_request_chan_sizes"`
	PerDepthCrawledChanSizes []int `yaml:"per_depth_crawled_chan_sizes"`
}

// Each crawl may specify its own cache directory and configuration. This
// will be used to store the results of the crawl. The cache is intended
// to be relative to the
type CrawlCacheConfig struct {
	Prefix           string `yaml:"cache_prefix"`
	ClearBeforeCrawl bool   `yaml:"cache_clear_before_crawl"`
}

// Confiug represents the configuration for a single crawl.
type Config struct {
	Name          string                `yaml:"name"`
	Depth         int                   `yaml:"depth"`
	Seeds         []string              `yaml:"seeds"`
	NoFollowRules []string              `yaml:"nofollow"`
	FollowRules   []string              `yaml:"follow"`
	RewriteRules  []string              `yaml:"rewrite"`
	Download      DownloadFactoryConfig `yaml:"download"`
	NumExtractors int                   `yaml:"num_extractors"`
	Extractors    []content.Type        `yaml:"extractors"`
	Cache         CrawlCacheConfig      `yaml:"cache"`
}

// NewLinkProcessor creates a outlinks.RegexpProcessor using the
// nofollow, follow and reqwrite specifications in the configuration.
func (c Config) NewLinkProcessor() (*outlinks.RegexpProcessor, error) {
	linkProcessor := &outlinks.RegexpProcessor{
		NoFollow: c.NoFollowRules,
		Follow:   c.FollowRules,
		Rewrite:  c.RewriteRules,
	}
	if err := linkProcessor.Compile(); err != nil {
		return nil, fmt.Errorf("failed to compile link processing rules: %v", err)
	}
	return linkProcessor, nil
}

// CreateAndCleanCache creates the cache directory for the crawl, relative
// to the specified root, and optionally clears it before the crawl (if
// Cache.ClearBeforeCrawl is true). Any environment variables in the
// root or Cache.Prefix will be expanded.
func (c Config) CreateAndCleanCache(root string) error {
	if len(c.Cache.Prefix) == 0 {
		return nil
	}
	root = os.ExpandEnv(root)
	crawlCache := filepath.Join(root, os.ExpandEnv(c.Cache.Prefix))
	if c.Cache.ClearBeforeCrawl {
		if err := os.RemoveAll(crawlCache); err != nil {
			return fmt.Errorf("failed to remove %v: %v", crawlCache, err)
		}
	}
	return nil
}

// SeedsByScheme returns the crawl seeds grouped by their scheme and any seeds
// that are not recognised by the supplied cloudpath.MatcherSpec.
func (c Config) SeedsByScheme(matchers cloudpath.MatcherSpec) (map[string][]cloudpath.Match, []string) {
	matches := map[string][]cloudpath.Match{}
	rejected := []string{}
	for _, seed := range c.Seeds {
		match := matchers.Match(seed)
		if len(match.Matched) == 0 {
			rejected = append(rejected, seed)
			continue
		}
		scheme := match.Scheme
		matches[scheme] = append(matches[scheme], match)
	}
	return matches, rejected
}

func (c Config) CreateSeedCrawlRequests(ctx context.Context, factories map[string]file.FSFactory, seeds map[string][]cloudpath.Match) ([]download.Request, error) {
	requests := []download.Request{}
	for scheme, matched := range seeds {
		factory, ok := factories[scheme]
		if !ok {
			return nil, fmt.Errorf("no file.FSFactory for scheme: %v", scheme)
		}
		container, err := factory.New(ctx, scheme)
		if err != nil {
			return nil, err
		}
		var req crawl.SimpleRequest
		req.FS = container
		req.Mode = 0600
		req.Depth = 0
		for _, match := range matched {
			req.Filenames = append(req.Filenames, match.Matched)
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// ExtractorRegistry returns a content.Registry containing for outlinks.Extractor
// that can be used with outlinks.Extract.
func (c Config) ExtractorRegistry(avail map[content.Type]outlinks.Extractor) (*content.Registry[outlinks.Extractor], error) {
	reg := content.NewRegistry[outlinks.Extractor]()
	for _, ctype := range c.Extractors {
		_, _, _, err := content.ParseTypeFull(ctype)
		if err != nil {
			return nil, err
		}
		if extractor, ok := avail[ctype]; ok {
			if err := reg.RegisterHandlers(ctype, extractor); err != nil {
				return nil, err
			}
		}
	}
	return reg, nil
}
