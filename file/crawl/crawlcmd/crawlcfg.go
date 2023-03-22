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
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/checkpoint"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/net/ratecontrol"
	"cloudeng.io/path/cloudpath"
)

// ExponentialBackoffConfig is the configuration for an exponential backoff
// retry strategy for downloads.
type ExponentialBackoff struct {
	InitialDelay time.Duration `yaml:"initial_delay"`
	Steps        int           `yaml:"steps"`
	StatusCodes  []int         `yaml:"status_codes,flow"`
}

// Rate specifies a rate in one of several forms, only one should
// be used.
type Rate struct {
	Tick            time.Duration `yaml:"tick"`
	RequestsPerTick int           `yaml:"requests_per_tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick"`
}

// RateControl is the configuration for rate based control of download
// requests.
type RateControl struct {
	Rate               Rate               `yaml:"rate_control"`
	ExponentialBackoff ExponentialBackoff `yaml:"exponential_backoff"`
}

// DownloadFactoryConfig is the configuration for a crawl.DownloaderFactory.
type DownloadFactoryConfig struct {
	DefaultConcurrency       int   `yaml:"default_concurrency"`
	DefaultRequestChanSize   int   `yaml:"default_request_chan_size"`
	DefaultCrawledChanSize   int   `yaml:"default_crawled_chan_size"`
	PerDepthConcurrency      []int `yaml:"per_depth_concurrency"`
	PerDepthRequestChanSizes []int `yaml:"per_depth_request_chan_sizes"`
	PerDepthCrawledChanSizes []int `yaml:"per_depth_crawled_chan_sizes"`
}

type DownloadConfig struct {
	DownloadFactoryConfig `yaml:",inline"`
	RateControlConfig     RateControl `yaml:",inline"`
}

// Each crawl may specify its own cache directory and configuration. This
// will be used to store the results of the crawl. The cache is intended
// to be relative to the
type CrawlCacheConfig struct {
	Prefix            string `yaml:"cache_prefix"`
	ClearBeforeCrawl  bool   `yaml:"cache_clear_before_crawl"`
	Checkpoint        string `yaml:"cache_checkpoint"`
	ShardingPrefixLen int    `yaml:"cache_sharding_prefix_len"`
}

// Initialize creates the cache and checkpoint directories relative to the
// specified root, and optionally clears them before the crawl (if
// Cache.ClearBeforeCrawl is true). Any environment variables in the
// root or Cache.Prefix will be expanded.
func (c CrawlCacheConfig) Initialize(root string) (string, checkpoint.Operation, error) {
	root = os.ExpandEnv(root)
	cachePath, checkpointPath := os.ExpandEnv(c.Prefix), os.ExpandEnv(c.Checkpoint)
	cachePath = filepath.Join(root, cachePath)
	checkpointPath = filepath.Join(root, checkpointPath)

	if c.ClearBeforeCrawl {
		if err := os.RemoveAll(cachePath); err != nil {
			return "", nil, err
		}
		if len(c.Checkpoint) > 0 {
			if err := os.RemoveAll(checkpointPath); err != nil {
				return "", nil, err
			}
		}
	}
	var cp checkpoint.Operation
	var err error
	if len(c.Checkpoint) > 0 {
		cp, err = checkpoint.NewDirectoryOperation(checkpointPath)
		if err != nil {
			return "", nil, err
		}
	}
	return cachePath, cp, os.MkdirAll(cachePath, 0700)
}

// Config represents the configuration for a single crawl.
type Config struct {
	Name          string           `yaml:"name"`
	Depth         int              `yaml:"depth"`
	Seeds         []string         `yaml:"seeds"`
	NoFollowRules []string         `yaml:"nofollow"`
	FollowRules   []string         `yaml:"follow"`
	RewriteRules  []string         `yaml:"rewrite"`
	Download      DownloadConfig   `yaml:"download"`
	NumExtractors int              `yaml:"num_extractors"`
	Extractors    []content.Type   `yaml:"extractors"`
	Cache         CrawlCacheConfig `yaml:"cache"`
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

// CreateSeedCrawlRequests creates a set of crawl requests for the supplied
// seeds. It use the factories to create a file.FS for the URI scheme of
// each seed.
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

func (c RateControl) NewRateController() (*ratecontrol.Controller, error) {
	opts := []ratecontrol.Option{}
	fmt.Printf("rate: %v\n", c.Rate.Tick)
	if c.Rate.Tick != 0 {
		var clock ratecontrol.Clock
		switch {
		case c.Rate.Tick < time.Minute:
			clock = ratecontrol.SecondClock{}
		case c.Rate.Tick < time.Hour:
			clock = ratecontrol.MinuteClock{}
		case c.Rate.Tick == time.Hour:
			clock = ratecontrol.HourClock{}
		default:
			return nil, fmt.Errorf("unsupported tick duration (only seconds, minutes and hours are supported): %v", c.Rate.Tick)
		}
		opts = append(opts, ratecontrol.WithClock(clock))
	}
	if c.Rate.BytesPerTick > 0 {
		opts = append(opts, ratecontrol.WithBytesPerTick(c.Rate.BytesPerTick))
	}
	if c.Rate.RequestsPerTick > 0 {
		opts = append(opts, ratecontrol.WithRequestsPerTick(c.Rate.RequestsPerTick))
	}
	if c.ExponentialBackoff.InitialDelay > 0 {
		opts = append(opts, ratecontrol.WithExponentialBackoff(c.ExponentialBackoff.InitialDelay, c.ExponentialBackoff.Steps))
	}
	return ratecontrol.New(opts...), nil
}
