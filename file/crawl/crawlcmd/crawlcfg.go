// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd

import "cloudeng.io/file/content"

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
	CachePrefix           string `yaml:"cache_prefix"`
	CacheClearBeforeCrawl bool   `yaml:"cache_clear_before_crawl"`
}

// Confiug represents the configuration for a single crawl.
type Config struct {
	Name          string                  `yaml:"name"`
	Depth         int                     `yaml:"depth"`
	Seeds         []string                `yaml:"seeds"`
	NoFollowRules []string                `yaml:"nofollow"`
	FollowRules   []string                `yaml:"follow"`
	RewriteRules  []string                `yaml:"rewrite"`
	Download      DownloadFactoryConfig   `yaml:"download"`
	NumExtractors int                     `yaml:"num_extractors"`
	Extractors    map[content.Type]string `yaml:"extractors"`
	Cache         CrawlCacheConfig        `yaml:"cache"`
}
