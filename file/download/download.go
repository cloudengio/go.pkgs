// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download

import (
	"context"
	"io/fs"

	"cloudeng.io/file"
	"cloudeng.io/file/content"
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
	// Contents of the download, nil on error.
	Contents []byte
	// FileInfo for the downloaded file.
	FileInfo fs.FileInfo
	// Name of the downloaded file.
	Name string
	// Number of retries that were required to download the file.
	Retries int
	// Error encountered during the download.
	Err error
}

// Downloaded represents all of the downloads in response
// to a given request.
type Downloaded struct {
	Request   Request
	Downloads []Result
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

// AsObjects returns the specified downloaded results as a slice
// of content.Object.
func AsObjects(downloaded []Result) (objs []content.Object[[]byte, Result]) {
	for _, dl := range downloaded {
		var obj content.Object[[]byte, Result]
		obj.Value = dl.Contents
		obj.Response = dl
		obj.Response.Contents = nil
		obj.Response.Err = content.Error(dl.Err)
		obj.Type = content.TypeForPath(dl.Name)
		objs = append(objs, obj)
	}
	return objs
}
