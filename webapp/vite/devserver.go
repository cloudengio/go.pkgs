// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"regexp"

	"cloudeng.io/os/executil"
)

// DevServer represents a vite dev server.
type DevServer struct {
	ctx    context.Context
	cancel context.CancelFunc
	filter io.WriteCloser
	addrRE *regexp.Regexp
	ch     chan []byte
	cmd    *exec.Cmd
}

// Flags represents the flags commonly used when using
// vite dev servers.
type Flags struct {
	ViteDir    string `subcmd:"vite-dir,,'set to a directory containing a vite configuration with the vite dev server configured. This dev server will then be started and requests proxied to it.'"`
	ViteServer string `subcmd:"vite-server,,set to the url of an already running vite dev server to which requests will be proxied."`
}

// NewDevServer creates a new instance of DevServer. Note, that the
// server is not started at this point. The dir argument specifies the directory
// containing the vite configuration. Context, command and args are passed to
// exec.CommandContext. A typical usage would be:
//
//	NewDevServer(ctx, "./frontend", "vite", "dev")
//
// Additional, optional configuration is possible via the Configure method.
func NewDevServer(ctx context.Context, dir string, command string, args ...string) *DevServer {
	ds := &DevServer{}
	ds.ctx, ds.cancel = context.WithCancel(ctx)
	ds.cmd = exec.CommandContext(ctx, command, args...)
	ds.cmd.Dir = dir
	return ds
}

// SetSdoutStderr sets the stdout and stderr io.Writers to be used
// by the dev server.
func SetSdoutStderr(stdout, stderr io.Writer) DevServerOption {
	return func(ds *DevServer) {
		ds.cmd.Stdout = stdout
		ds.cmd.Stderr = stderr
	}
}

// AddrRegularExpression specifies the regular expression to use for
// determining the running server's address. The default RE is:
// '> Local:'. However, some configurations may use a different
// output.
func AddrRegularExpression(re *regexp.Regexp) DevServerOption {
	return func(ds *DevServer) {
		ds.addrRE = re
	}
}

// DevServerOption represents an option to Configure.
type DevServerOption func(ds *DevServer)

var hostRE = regexp.MustCompile("> Local:")

// Configure applies options and mus be called before Start.
func (ds *DevServer) Configure(opts ...DevServerOption) {
	for _, fn := range opts {
		fn(ds)
	}
	if ds.addrRE == nil {
		ds.addrRE = hostRE
	}
}

func extractURL(line []byte) (*url.URL, error) {
	sp := bytes.LastIndex(line, []byte{' '})
	if sp < 0 || (sp+1 >= len(line)) {
		return nil, fmt.Errorf("malformed line: %s", line)
	}
	return url.Parse(string(line[sp+1:]))
}

// WaitForURL parses the output of the development server looking for a
// line that specifies the URL it is listening on.
func (ds *DevServer) WaitForURL(ctx context.Context) (*url.URL, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case line := <-ds.ch:
			u, err := extractURL(line)
			ds.filter.Close()
			return u, err
		}
	}
}

// Start starts the dev server.
func (ds *DevServer) Start() error {
	ds.ch = make(chan []byte, 1)
	ds.filter = executil.NewLineFilter(ds.cmd.Stdout, ds.addrRE, ds.ch)
	ds.cmd.Stdout = ds.filter
	return ds.cmd.Start()
}

// Shutdown asks the dev server to shut itself down.
func (ds *DevServer) Shutdown() {
	ds.cancel()
}