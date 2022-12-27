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
}

// Outlinks represents the interface to an 'outlink' extractor, that is, an
// entity that determines additional items to be downloaded based on the
// contents of an already downloaded one.
type Outlinks interface {
	Extract(ctx context.Context, download download.Downloaded) []download.Request
}

// T represents the interface to a crawler. The crawler will download
// the requested items and in addition, determine further items to be crawled,
// based on their contents, using the supplied link extractor.
type T interface {
	Run(ctx context.Context,
		extractor Outlinks,
		downloader download.T,
		writeFS file.WriteFS,
		input <-chan download.Request,
		output chan<- Crawled) error
}
