// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devserver

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"

	"cloudeng.io/os/executil"
)

// DevServer represents a development server, such as provided
// by webpack or vite that serves UI content with live reload
// capabilities.
type DevServer struct {
	cmd    *exec.Cmd
	closer io.Closer
}

// URLExtractor parses each line of output from the dev server looking
// for a URL to which requests can be proxied.
// If a URL is successfully extracted it is returned with a nil error.
// If the line does not contain a URL, then a nil URL and a nil error
// are returned.
// If the line should contain a URL but it cannot be extracted
// then a nil URL and a non-nil error should be returned.
type URLExtractor func(line []byte) (*url.URL, error)

// NewServer creates a new DevServer instance that will manage
// the lifecycle of the supplied exec.Cmd instance. The stdout of
// the command is scanned line-by-line and passed to the supplied
// URLExtractor function until a URL is successfully extracted.
func NewServer(ctx context.Context, dir, binary string, args ...string) *DevServer {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	ds := &DevServer{
		cmd: cmd,
	}
	return ds
}

// StartAndWaitForURL starts the dev server and waits until a URL is extracted
// from its output using the supplied URLExtractor function. The
// context can be used to cancel the wait operation. If the context
// is cancelled before a URL is extracted an error is returned.
func (ds *DevServer) StartAndWaitForURL(ctx context.Context, writer io.Writer, extractor URLExtractor) (*url.URL, error) {
	ch := make(chan []byte, 1)
	filter := executil.NewLineFilter(writer, ch)
	ds.cmd.Stdout = filter
	ds.closer = filter
	if err := ds.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dev server %q in %q: %w", ds.cmd.String(), ds.cmd.Dir, err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case line := <-ch:
			u, err := extractor(line)
			if u == nil && err == nil {
				continue

			}
			// expect a URL, it's either valid or invalid, either way
			// we return.
			return u, err
		}
	}
}

// CloseStdout closes the stdout from the dev server process and
// will prevent any further output from being processed or forwarded
// to the writer supplied to StartAndWaitForURL.
func (ds *DevServer) Close() {
	if ds.closer != nil {
		ds.closer.Close()
	}
}
