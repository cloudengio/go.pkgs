// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package executil provides utilities for working with os/exec.
package executil

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

type linefilter struct {
	*io.PipeWriter
	re      *regexp.Regexp
	forward io.Writer
	ch      chan<- []byte
	prd     *io.PipeReader
	errCh   chan error
}

func discardIfNil(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}

// NewLineFilter returns an io.WriteCloser that scans the contents of the
// supplied io.Writer and sends lines that match the regexp to the supplied
// channel. It can be used to filter the output of a command started
// by the exec package for example for specific output. Call Close on
// the returned io.WriteCloser to ensure that all resources are reclaimed.
func NewLineFilter(forward io.Writer, re *regexp.Regexp, ch chan<- []byte) io.WriteCloser {
	lf := &linefilter{
		forward: discardIfNil(forward),
		re:      re,
		ch:      ch,
		errCh:   make(chan error, 1),
	}
	lf.prd, lf.PipeWriter = io.Pipe()
	go lf.readlines()
	return lf
}

func send(ch chan<- []byte, buf []byte) {
	cpy := make([]byte, len(buf))
	copy(cpy, buf)
	select {
	case ch <- cpy:
	default:
	}
}

func (lf *linefilter) readlines() {
	sc := bufio.NewScanner(lf.prd)
	for sc.Scan() {
		buf := sc.Bytes()
		if lf.re.Match(buf) {
			send(lf.ch, buf)
		}
		lf.forward.Write(buf)
		lf.forward.Write([]byte{'\n'})
	}
	lf.errCh <- sc.Err()
}

// Close implements io.Closer.
func (lf *linefilter) Close() error {
	// TODO(cnicolaou): make sure all goroutines shutdown.
	lf.prd.Close()
	lf.PipeWriter.Close()
	close(lf.ch)
	err := <-lf.errCh
	if !strings.Contains(err.Error(), "read/write on closed pipe") {
		return err
	}
	return nil
}
