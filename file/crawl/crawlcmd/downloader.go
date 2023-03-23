// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd

import (
	"context"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
)

// NewFactory returns a new instance of a crawl.DownloaderFactory which is
// parametised via its DownloadFactoryConfig receiver.
func (df DownloadFactoryConfig) NewFactory(ch chan<- download.Progress) crawl.DownloaderFactory {
	return &downloaderFactory{
		DownloadFactoryConfig: df,
		ProgressChan:          ch,
	}
}

type downloaderFactory struct {
	DownloadFactoryConfig
	ProgressChan chan<- download.Progress
}

func (df DownloadFactoryConfig) depthOrDefault(depth int, values []int, def int) int {
	if depth < len(values) {
		return values[depth]
	}
	return def
}

// New implements download.DownloaderFactory.
// New creates a new instance of a download.T and the associated input and output
// channels appropriate for the depth of the crawl.
func (df *downloaderFactory) New(_ context.Context, depth int) (
	downloader download.T,
	inputCh chan download.Request,
	outputCh chan download.Downloaded) {
	concurrency := df.depthOrDefault(depth, df.PerDepthConcurrency, df.DefaultConcurrency)
	reqChanSize := df.depthOrDefault(depth, df.PerDepthRequestChanSizes, df.DefaultRequestChanSize)
	dlChanSize := df.depthOrDefault(depth, df.PerDepthCrawledChanSizes, df.DefaultCrawledChanSize)
	inputCh = make(chan download.Request, reqChanSize)
	outputCh = make(chan download.Downloaded, dlChanSize)
	downloader = download.New(download.WithNumDownloaders(concurrency))
	return
}

// Depth0Chans creates the chanels required to start the crawl with their
// capacities set to the values specified in the DownloadFactoryConfig for
// a depth0 crawl, or the default values if none are specified.
func (df DownloadFactoryConfig) Depth0Chans() (chan download.Request, chan crawl.Crawled) {
	reqChanSize := df.depthOrDefault(0, df.PerDepthRequestChanSizes, df.DefaultRequestChanSize)
	dlChanSize := df.depthOrDefault(0, df.PerDepthCrawledChanSizes, df.DefaultCrawledChanSize)
	return make(chan download.Request, reqChanSize),
		make(chan crawl.Crawled, dlChanSize)
}
