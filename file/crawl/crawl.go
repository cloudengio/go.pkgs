// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"io"
	"io/fs"
)

// Change Object -> Objects
// Container + slice of names.
type Object struct {
	Container fs.FS
	Name      string
}

// Request represents a request for an item to be fetched.
type Request struct {
	Object
}

// Downloaded represents a item that has been Downloaded.
type Downloaded struct {
	Object
	Request Object
	Retries int
	Err     error
}

// DownloadProgress is used to communicate the progress of a download run.
type DownloadProgress struct {
	// Downloaded is the total number of items downloaded so far.
	Downloaded int64
	// Outstanding is the current size of the input channel for items to
	// be downloaded.
	Outstanding int64
}

// Creator provides a means of creating a new file.
type Creator interface {
	New(name string) (io.WriteCloser, Object, error)
}

// Downloader represents the interface to a downloader that is used
// to download content.
type Downloader interface {
	// Run initiates a fetch run.
	Run(ctx context.Context,
		creator Creator,
		input <-chan []Request,
		output chan<- []Downloaded) error
}

// Outlinks represents the interface to an 'outlink' extractor, that is, an
// entity that determines additional items to be downloaded based on the
// contents of an already downloaded one. Generally these will references
// to external documents/files.
type Outlinks interface {
	Extract(ctx context.Context, item []Downloaded) []Request
}

// T represents the interface to a crawler. The crawler will download
// the requested items and in addition, determine further items to be crawled,
// based on their contents, using the supplied link extractor.
type T interface {
	Run(ctx context.Context,
		extractor Outlinks,
		downloader Downloader,
		creator Creator,
		input <-chan []Request,
		output chan<- []Downloaded) error
}
