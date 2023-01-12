// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download

import (
	"context"
	"io"
	"io/fs"

	"cloudeng.io/file"
)

// Request represents a request for a list of objects, stored in the same
// container, to be downloaded.
type Request interface {
	Container() file.FS
	FileMode() fs.FileMode // FileMode to use for the downloaded contents.
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
	Container file.FS
	Downloads []Result
}

// WriteFS extends file.FS to add a Create method.
type WriteFS interface {
	file.FS
	Create(ctx context.Context, name string, mode fs.FileMode) (io.WriteCloser, error)
}

// T represents the interface to a downloader that is used
// to download content.
type T interface {
	// Run initiates a download run. It reads Requests from the specified
	// input channel and writes the results of those downloads to the output
	// channel. Closing the input channel indicates to Run that it should
	// complete all outstanding download requests. Run will close the output
	// channel when all requests have been processed.
	Run(ctx context.Context,
		input <-chan Request,
		output chan<- Downloaded) error
}
