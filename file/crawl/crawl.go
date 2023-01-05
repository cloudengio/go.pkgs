// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"

	"cloudeng.io/file"
	"cloudeng.io/file/download"
)

// Request represents a request for a list of objects stored in the
// same container to be downloaded/crawled.
//type Request interface {
//	download.Request
//	Depth() int
//	IncDepth()
//}

// Crawled represents all of the downloads in response to a given crawl request.
type Crawled struct {
	download.Downloaded
	Outlinks []download.Request
	Depth    int
}

// Outlinks represents the interface to an 'outlink' extractor, that is, an
// entity that determines additional items to be downloaded based on the
// contents of an already downloaded one.
type Outlinks interface {
	// Note that the implementation of Extract is responsible for removing
	// duplicates from the set of extracted links returned.
	Extract(ctx context.Context, depth int, download download.Downloaded) []download.Request
}

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
