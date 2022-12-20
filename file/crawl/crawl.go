// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"io"
	"io/fs"
)

// Request represents a request for a list of objects stored in the
// same container to be downloaded/crawled.
type Request struct {
	Container fs.FS
	Names     []string
	Depth     int
}

// DownloadStatus represents the result of the download for a single
// object.
type DownloadStatus struct {
	Name    string
	Retries int
	Err     error
}

// Downloadeded represents all of the downloads in response
// to a given request.
type Downloaded struct {
	Request   Request
	Container fs.FS
	Downloads []DownloadStatus
}

// Creator provides a means of creating a new file.
type Creator interface {
	Container() fs.FS
	New(name string) (io.WriteCloser, string, error)
}

// Downloader represents the interface to a downloader that is used
// to download content.
type Downloader interface {
	// Run initiates a fetch run.
	Run(ctx context.Context,
		creator Creator,
		input <-chan Request,
		output chan<- Downloaded) error
}

// Outlinks represents the interface to an 'outlink' extractor, that is, an
// entity that determines additional items to be downloaded based on the
// contents of an already downloaded one. Generally these will references
// to external documents/files.
type Outlinks interface {
	Extract(ctx context.Context, item Downloaded) []Request
}

// T represents the interface to a crawler. The crawler will download
// the requested items and in addition, determine further items to be crawled,
// based on their contents, using the supplied link extractor.
type T interface {
	Run(ctx context.Context,
		extractor Outlinks,
		downloader, outlinkDownloader Downloader,
		creator Creator,
		input <-chan Request,
		output chan<- Downloaded) error
}
