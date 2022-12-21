// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download

import (
	"context"
	"io"
	"io/fs"
)

// Request represents a request for a list of objects, stored in the same
// container, to be downloaded.
type Request interface {
	Container() fs.FS
	Names() []string
}

// Result represents the result of the download for a single object.
type Result struct {
	Name    string
	Retries int
	Err     error
}

// Downloaded represents all of the downloads in response
// to a given request.
type Downloaded struct {
	Request   Request
	Container fs.FS
	Downloads []Result
}

// Creator provides a means of creating a new file.
type Creator interface {
	Container() fs.FS
	New(name string) (io.WriteCloser, string, error)
}

// T represents the interface to a downloader that is used
// to download content.
type T interface {
	// Run initiates a fetch run.
	Run(ctx context.Context,
		creator Creator,
		input <-chan Request,
		output chan<- Downloaded) error
}
