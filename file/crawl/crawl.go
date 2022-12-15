// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"io"
	"io/fs"
)

// Request represents a request for an item to be fetched.
type Request struct {
	Container fs.FS
	Name      string
}

// Downloaded represents a item that has been Downloaded.
type Downloaded struct {
	Request
	Retries int
	Err     error
}

// Progress is used to communicate the progress of a crawl run.
type Progress struct {
	Downloaded  int64
	Outstanding int64
}

// Creator provides a means of creating a new file.
type Creator interface {
	New(name string) (io.WriteCloser, Request, error)
}

// Downloader represents the interface to a downloader that is used
// to download content.
type Downloader interface {
	// Run initiates a fetch run.
	Run(ctx context.Context,
		creator Creator,
		progress chan<- Progress,
		input <-chan []Request,
		output chan<- []Downloaded) error
}

// Extractor represents the interface to a link extractor, that is,
// an entity which determines additional items to download based on
// already downloaded ones.
type Extractor interface {
	Run(ctx context.Context,
		input <-chan []Downloaded,
		output chan<- []Request) error
}

/*
type T struct {
}

func NewCrawler(fetcher Fetcher, extractor Links) (*T, error) {
	return &T{}, nil
}

func (c *T) Run(ctx context.Context,
	creator Creator,
	progress chan<- Progress,
	input <-chan []Request,
	output chan<- []Downloaded) {
}
*/
