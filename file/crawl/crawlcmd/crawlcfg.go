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
	"time"

	"cloudeng.io/file/checkpoint"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/net/ratecontrol"
	"cloudeng.io/path/cloudpath"
	"gopkg.in/yaml.v3"
)

// ExponentialBackoffConfig is the configuration for an exponential backoff
// retry strategy for downloads.
type ExponentialBackoff struct {
	InitialDelay time.Duration `yaml:"initial_delay" cmd:"the initial delay between retries for exponential backoff"`
	Steps        int           `yaml:"steps" cmd:"the number of steps of exponential backoff before giving up"`
	StatusCodes  []int         `yaml:"status_codes,flow" cmd:"the status codes that trigger a retry"`
}

// Rate specifies a rate in one of several forms, only one should
// be used.
type Rate struct {
	Tick            time.Duration `yaml:"tick" cmd:"the duration of a tick"`
	RequestsPerTick int           `yaml:"requests_per_tick" cmd:"the number of requests per tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick" cmd:"the number of bytes per tick"`
}

// RateControl is the configuration for rate based control of download
// requests.
type RateControl struct {
	Rate               Rate               `yaml:"rate_control" cmd:"the rate control parameters"`
	ExponentialBackoff ExponentialBackoff `yaml:"exponential_backoff" cmd:"the exponential backoff parameters"`
}

// DownloadFactoryConfig is the configuration for a crawl.DownloaderFactory.
type DownloadFactoryConfig struct {
	DefaultConcurrency       int   `yaml:"default_concurrency" cmd:"the number of concurrent downloads (defaults to GOMAXPROCS(0)), used when a per crawl depth value is not specified via per_depth_concurrency."`
	DefaultRequestChanSize   int   `yaml:"default_request_chan_size" cmd:"the size of the channel used to queue download requests, used when a per crawl depth value is not specified via per_depth_request_chan_sizes. Increased values allow for more concurrency between discovering new items to crawl and crawling them."`
	DefaultCrawledChanSize   int   `yaml:"default_crawled_chan_size" cmd:"the size of the channel used to queue downloaded items, used when a per crawl depth value is not specified via per_depth_crawled_chan_sizes. Increased values allow for more concurrency between downloading documents and processing them."`
	PerDepthConcurrency      []int `yaml:"per_depth_concurrency" cmd:"per crawl depth values for the number of concurrent downloads"`
	PerDepthRequestChanSizes []int `yaml:"per_depth_request_chan_sizes" cmd:"per crawl depth values for the size of the channel used to queue download requests"`
	PerDepthCrawledChanSizes []int `yaml:"per_depth_crawled_chan_sizes" cmd:"per crawl depth values for the size of the channel used to queue downloaded items"`
}

type DownloadConfig struct {
	DownloadFactoryConfig `yaml:",inline"`
	RateControlConfig     RateControl `yaml:",inline"`
}

// Each crawl may specify its own cache directory and configuration. This
// will be used to store the results of the crawl. The ServiceSpecific
// field is intended to be parametized to some service specific configuration
// for cache services that require it, such as AWS S3. This is deliberately
// left to client packages to avoid depenedency bloat in core packages
// such as this.
type CrawlCacheConfig[T any] struct {
	Downloads         string `yaml:"downloads" cmd:"the prefix/directory to use for the cache of downloaded documents. This is an absolute path the root directory of the crawl."`
	ClearBeforeCrawl  bool   `yaml:"clear_before_crawl" cmd:"if true, the cache and checkpoint will be cleared before the crawl starts."`
	Checkpoint        string `yaml:"checkpoint" cmd:"the location of any checkpoint data used to resume a crawl, this is an absolute path."`
	ShardingPrefixLen int    `yaml:"sharding_prefix_len" cmd:"the number of characters of the filename to use for sharding the cache. This is intended to avoid filesystem limits on the number of files in a directory."`
	Concurrency       int    `yaml:"concurrency" cmd:"the number of concurrent operations to use when reading/writing to the cache."`
	ServiceConfig     T      `yaml:"service_config,omitempty" cmd:"cache service specific configuration, eg. AWS specific configuration"`
}

// ParseCrawlCacheConfig parses a CrawlCacheConfig for a specific cache service.
func ParseCrawlCacheConfig[T any](cfg CrawlCacheConfig[yaml.Node], specific *CrawlCacheConfig[T]) error {
	specific.Downloads = cfg.Downloads
	specific.ClearBeforeCrawl = cfg.ClearBeforeCrawl
	specific.Checkpoint = cfg.Checkpoint
	specific.ShardingPrefixLen = cfg.ShardingPrefixLen
	specific.Concurrency = cfg.Concurrency
	if err := cfg.ServiceConfig.Decode(&specific.ServiceConfig); err != nil {
		return err
	}
	return nil
}

// Paths returns the downloads and checkpoint paths expanded using os.ExpandEnv.
func (c CrawlCacheConfig[T]) Paths() (downloads, checkpoint string) {
	return os.ExpandEnv(c.Downloads), os.ExpandEnv(c.Checkpoint)
}

// PrepareDownloads ensures that the cache directory exists and is empty if
// ClearBeforeCrawl is true. It returns an error if the directory cannot be
// created or cleared.
func (c CrawlCacheConfig[T]) PrepareDownloads(ctx context.Context, fs content.FS) error {
	if c.ClearBeforeCrawl && len(c.Downloads) > 0 {
		if err := fs.DeleteAll(ctx, c.Downloads); err != nil {
			return err
		}
	}
	return fs.EnsurePrefix(ctx, c.Downloads, 0700)
}

// PrepareCheckpoint initializes the checkpoint operation (ie.
// calls op.Init(ctx, checkpointPath)) and optionally clears the checkpoint if
// ClearBeforeCrawl is true. It returns an error if the checkpoint cannot be
// initialized or cleared.
func (c CrawlCacheConfig[T]) PrepareCheckpoint(ctx context.Context, op checkpoint.Operation) error {
	if err := op.Init(ctx, c.Checkpoint); err != nil {
		return err
	}
	if c.ClearBeforeCrawl {
		return op.Clear(ctx)
	}
	return nil
}

// Config represents the configuration for a single crawl.
type Config struct {
	Name          string                      `yaml:"name" cmd:"the name of the crawl"`
	Depth         int                         `yaml:"depth" cmd:"the maximum depth to crawl"`
	Seeds         []string                    `yaml:"seeds" cmd:"the initial set of URIs to crawl"`
	NoFollowRules []string                    `yaml:"nofollow" cmd:"a set of regular expressions that will be used to determine which links to not follow. The regular expressions are applied to the full URL."`
	FollowRules   []string                    `yaml:"follow" cmd:"a set of regular expressions that will be used to determine which links to follow. The regular expressions are applied to the full URL."`
	RewriteRules  []string                    `yaml:"rewrite" cmd:"a set of regular expressions that will be used to rewrite links. The regular expressions are applied to the full URL."`
	Download      DownloadConfig              `yaml:"download" cmd:"the configuration for downloading documents"`
	NumExtractors int                         `yaml:"num_extractors" cmd:"the number of concurrent link extractors to use"`
	Extractors    []content.Type              `yaml:"extractors" cmd:"the content types to extract links from"`
	Cache         CrawlCacheConfig[yaml.Node] `yaml:"cache" cmd:"the configuration for the cache of downloaded documents"`
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
func (c Config) CreateSeedCrawlRequests(
	ctx context.Context,
	factories map[string]FSFactory,
	seeds map[string][]cloudpath.Match,
) ([]download.Request, error) {
	requests := []download.Request{}
	for scheme, matched := range seeds {
		factory, ok := factories[scheme]
		if !ok {
			return nil, fmt.Errorf("no file.FSFactory for scheme: %v", scheme)
		}
		container, err := factory(ctx)
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

// NewRateController creates a new rate controller based on the values
// contained in RateControl.
func (c RateControl) NewRateController() (*ratecontrol.Controller, error) {
	opts := []ratecontrol.Option{}
	if c.Rate.BytesPerTick > 0 {
		opts = append(opts, ratecontrol.WithBytesPerTick(c.Rate.Tick, c.Rate.BytesPerTick))
	}
	if c.Rate.RequestsPerTick > 0 {
		opts = append(opts, ratecontrol.WithRequestsPerTick(c.Rate.Tick, c.Rate.RequestsPerTick))
	}
	if c.ExponentialBackoff.InitialDelay > 0 {
		opts = append(opts, ratecontrol.WithExponentialBackoff(c.ExponentialBackoff.InitialDelay, c.ExponentialBackoff.Steps))
	}
	return ratecontrol.New(opts...), nil
}
