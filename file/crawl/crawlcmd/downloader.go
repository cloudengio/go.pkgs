// Copyright 2022 cloudeng llc. All rights reserved.
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

func (df *downloaderFactory) New(ctx context.Context, depth int) (
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
