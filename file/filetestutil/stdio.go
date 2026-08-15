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
func CaptureStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()
	defer w.Close()

	outChan := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		outChan <- buf.String()
	}()

	fnErr := fn()
	_ = w.Close()
	out := <-outChan
	return out, fnErr
}

// CaptureStderr redirects os.Stderr to a pipe, runs fn, and returns any
// output written to os.Stderr along with any error returned by fn.
func CaptureStderr(fn func() error) (string, error) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()
	defer w.Close()

	outChan := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		outChan <- buf.String()
	}()

	fnErr := fn()
	_ = w.Close()
	out := <-outChan
	return out, fnErr
}

// FeedStdin redirects os.Stdin to read from a pipe populated with input,
// runs fn, and restores os.Stdin.
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
