// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"fmt"
	"io/fs"
	"math/rand"
	"sync"
	"time"
)

// Contents returns the contents stored in the mock fs.FS.
func Contents(fs fs.FS) map[string][]byte {
	switch mfs := fs.(type) {
	case *randFS:
		return mfs.contents
	case *randAfteRetryFS:
		return mfs.contents
	}
	panic(fmt.Sprintf("%T is not a mock fs.FS", fs))
}

type FSOption func(o *fsOptions)

type fsOptions struct {
	rnd        *rand.Rand
	maxSize    int
	random     bool
	numRetries int
	retryErr   error
}

func FSWithRandomContents(src rand.Source, maxSize int) FSOption {
	return func(o *fsOptions) {
		o.random = true
		o.rnd = rand.New(src)
		o.maxSize = maxSize
	}
}

func FSWithRandomContentsAfterRetry(src rand.Source, maxSize, numRetries int, err error) FSOption {
	return func(o *fsOptions) {
		o.numRetries = numRetries
		o.rnd = rand.New(src)
		o.maxSize = maxSize
		o.retryErr = err
	}
}

func NewMockFS(opts ...FSOption) fs.FS {
	var options fsOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.random {
		return &randFS{fsOptions: options, contents: map[string][]byte{}}
	}
	if options.numRetries > 0 {
		return &randAfteRetryFS{
			randFS:  randFS{fsOptions: options, contents: map[string][]byte{}},
			retries: map[string]int{},
		}
	}
	return nil
}

type randFS struct {
	sync.Mutex
	fsOptions
	contents map[string][]byte
}

func newRandomFileCreator(name string, rnd *rand.Rand, maxSize int) ([]byte, fs.File, error) {
	size := rnd.Intn(maxSize)
	contents := make([]byte, size)
	size, err := rnd.Read(contents)
	if err != nil {
		return nil, nil, err
	}
	return contents, NewFile(&BufferCloser{bytes.NewBuffer(contents)},
		NewInfo(name, size, 0666, time.Now(), false, nil)), nil
}

// Open implements fs.FS.
func (mfs *randFS) Open(name string) (fs.File, error) {
	mfs.Lock()
	defer mfs.Unlock()
	contents, f, err := newRandomFileCreator(name, mfs.rnd, mfs.maxSize)
	if err != nil {
		return nil, err
	}
	mfs.contents[name] = contents
	return f, nil
}

type randAfteRetryFS struct {
	randFS
	retries map[string]int
}

// Open implements fs.FS.
func (mfs *randAfteRetryFS) Open(name string) (fs.File, error) {
	mfs.Lock()
	mfs.retries[name]++
	if mfs.retries[name] <= mfs.numRetries {
		mfs.Unlock()
		return nil, mfs.retryErr
	}
	mfs.Unlock()
	return mfs.randFS.Open(name)
}
