// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"
	"io"
	"io/fs"
)

// Item represents a item to be crawled.
type Item struct {
	Container fs.FS
	Name      string
}

// Crawled represents a item that has been crawled.
type Crawled struct {
	Item
	Retries int
	Err     error
}

// Progress is used to communicate the progress of a crawl run.
type Progress struct {
	Crawled     int64
	Outstanding int64
}

// Creator provides a means of creating a new file.
type Creator interface {
	New(name string) (io.WriteCloser, Item, error)
}

// T represents the interface to a generic crawler.
type T interface {
	// Run initiates a crawl run.
	Run(ctx context.Context,
		creator Creator,
		progress chan<- Progress,
		input <-chan []Item,
		output chan<- []Crawled) error
}
