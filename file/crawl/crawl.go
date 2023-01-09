// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package crawl provides a framework for multilevel/recursive crawling files.
// As files are downloaded, they may be processed by an outlinks extractor which
// yields more files to crawled. Typically such a multilevel crawl is limited
// to a set number of iterations referred to as the depth of the crawl.
// The interface to a crawler is channel based to allow for concurrency.
// The outlink extractor is called for all downloaded files and should
// implement duplicate detection and removal.
package crawl

import (
	"context"

	"cloudeng.io/file"
	"cloudeng.io/file/download"
)

// Crawled represents all of the downloaded content in response to a given crawl
// request.
type Crawled struct {
	download.Downloaded
	Outlinks []download.Request
	Depth    int // The depth at which the document was crawled.
}

// Outlinks is the interface to an 'outlink' extractor, that is, an
// entity that determines additional items to be downloaded based on the
// contents of an already downloaded one.
type Outlinks interface {
	// Note that the implementation of Extract is responsible for removing
	// duplicates from the set of extracted links returned.
	Extract(ctx context.Context, depth int, download download.Downloaded) []download.Request
}

// DownloaderFactory is used to create a new downloader for each 'depth'
// in a multilevel crawl.
type DownloaderFactory func(ctx context.Context, depth int) (
	downloader download.T,
	input chan download.Request,
	output chan download.Downloaded)

// T represents the interface to a crawler.
type T interface {
	Run(ctx context.Context,
		factory DownloaderFactory,
		extractor Outlinks,
		writeFS file.WriteFS,
		input <-chan download.Request,
		output chan<- Crawled) error
}
