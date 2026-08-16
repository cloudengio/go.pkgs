// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"io"
	"os"
)

// CaptureStdout redirects os.Stdout to a pipe, runs fn, and returns any
// output written to os.Stdout along with any error returned by fn.
// NOTE: CaptureStdout temporarily replaces os.Stdout and is not
// safe for concurrent use.
func CaptureStdout(fn func() error) ([]byte, error) {
	return captureOSOutput(func() *os.File { return os.Stdout }, func(w *os.File) { os.Stdout = w }, fn)
}

// CaptureStderr redirects os.Stderr to a pipe, runs fn, and returns any
// output written to os.Stderr along with any error returned by fn.
// NOTE: CaptureStderr temporarily replaces os.Stderr and is not
// safe for concurrent use.
func CaptureStderr(fn func() error) ([]byte, error) {
	return captureOSOutput(func() *os.File { return os.Stderr }, func(w *os.File) { os.Stderr = w }, fn)
}

func captureOSOutput(get func() *os.File, set func(*os.File), fn func() error) ([]byte, error) {
	old := get()
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	set(w)
	defer func() {
		set(old)
	}()
	defer w.Close()

	outChan := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		outChan <- buf.Bytes()
	}()

	fnErr := fn()
	_ = w.Close()
	out := <-outChan
	return out, fnErr
}

// FeedStdin redirects os.Stdin to read from a pipe populated with input,
// runs fn, and restores os.Stdin.
// NOTE: FeedStdin temporarily replaces os.Stdin and is not
// safe for concurrent use.
func FeedStdin(input string, fn func() error) error {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
	}()
	defer r.Close()

	go func() {
		_, _ = w.Write([]byte(input))
		_ = w.Close()
	}()

	fnErr := fn()
	_ = r.Close()
	return fnErr
}
