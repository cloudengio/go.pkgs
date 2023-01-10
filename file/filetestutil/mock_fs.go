// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"sync"
	"time"

	"cloudeng.io/file"
)

// Contents returns the contents stored in the mock fs.FS.
func Contents(fs file.FS) map[string][]byte {
	switch mfs := fs.(type) {
	case *randFS:
		return mfs.contents
	case *constantFS:
		return mfs.contents
	case *randAfteRetryFS:
		return mfs.contents
	case *writeFS:
		return mfs.contents
	}
	panic(fmt.Sprintf("%T is not a mock fs.FS", fs))
}

// FSOption represents an option to configure a new mock instance of fs.FS.
type FSOption func(o *fsOptions)

type fsOptions struct {
	rnd        *rand.Rand
	val        []byte
	maxSize    int
	random     bool
	constant   bool
	numRetries int
	retryErr   error
	returnErr  error
	writeFS    bool
}

// FSWithRandomContents requests a mock FS that will return files of a random
// size (up to maxSize) with random contents.
func FSWithRandomContents(src rand.Source, maxSize int) FSOption {
	return func(o *fsOptions) {
		o.random = true
		o.rnd = rand.New(src)
		o.maxSize = maxSize
	}
}

// FSWithConstantContents requests a mock FS that will return files of a random
// size (up to maxSize) with random contents.
func FSWithConstantContents(val []byte, repeat int) FSOption {
	return func(o *fsOptions) {
		o.constant = true
		o.val = val
		o.maxSize = repeat
	}
}

// FSWithRandomContentsAfterRetry is like FSWithRandomContents but will
// return err, numRetries times before succeeding.
func FSWithRandomContentsAfterRetry(src rand.Source, maxSize, numRetries int, err error) FSOption {
	return func(o *fsOptions) {
		o.numRetries = numRetries
		o.rnd = rand.New(src)
		o.maxSize = maxSize
		o.retryErr = err
	}
}

// FSErrorOnly requests a mock FS that always returns err.
func FSErrorOnly(err error) FSOption {
	return func(o *fsOptions) {
		o.returnErr = err
	}
}

// FSWriteFS requests a mock that implements file.WriteFS.
func FSWriteFS() FSOption {
	return func(o *fsOptions) {
		o.writeFS = true
	}
}

// NewMockFS returns an new mock instance of file.FS as per the specified options.
func NewMockFS(opts ...FSOption) file.FS {
	var options fsOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.random {
		return &randFS{fsOptions: options, contents: map[string][]byte{}}
	}
	if options.constant {
		return &constantFS{fsOptions: options, contents: map[string][]byte{}}
	}
	if options.numRetries > 0 {
		return &randAfteRetryFS{
			randFS:  randFS{fsOptions: options, contents: map[string][]byte{}},
			retries: map[string]int{},
		}
	}
	if err := options.returnErr; err != nil {
		return &errorFs{err: err}
	}
	if options.writeFS {
		return newWriteFS()
	}
	return nil
}

type randFS struct {
	sync.Mutex
	fsOptions
	contents map[string][]byte
}

func newRandomFileCreator(ctx context.Context, name string, rnd *rand.Rand, maxSize int) ([]byte, fs.File, error) {
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
func (mfs *randFS) Open(ctx context.Context, name string) (fs.File, error) {
	mfs.Lock()
	defer mfs.Unlock()
	contents, f, err := newRandomFileCreator(ctx, name, mfs.rnd, mfs.maxSize)
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

// Open implements file.FS.
func (mfs *randAfteRetryFS) Open(ctx context.Context, name string) (fs.File, error) {
	mfs.Lock()
	mfs.retries[name]++
	if mfs.retries[name] <= mfs.numRetries {
		mfs.Unlock()
		return nil, mfs.retryErr
	}
	mfs.Unlock()
	return mfs.randFS.Open(ctx, name)
}

type errorFs struct {
	err error
}

func (mfs *errorFs) Open(ctx context.Context, name string) (fs.File, error) {
	return nil, mfs.err
}

type constantFS struct {
	sync.Mutex
	fsOptions
	val      []byte
	contents map[string][]byte
}

// Open implements fs.FS.
func (cfs *constantFS) Open(ctx context.Context, name string) (fs.File, error) {
	cfs.Lock()
	defer cfs.Unlock()
	contents := bytes.Repeat(cfs.val, cfs.maxSize)
	cfs.contents[name] = contents
	return NewFile(&BufferCloser{bytes.NewBuffer(contents)},
		NewInfo(name, len(contents), 0666, time.Now(), false, nil)), nil
}

type writeFSEntry struct {
	contents []byte
	mode     fs.FileMode
	update   time.Time
}

type writeFS struct {
	sync.Mutex
	entries  map[string]writeFSEntry
	contents map[string][]byte
}

func newWriteFS() file.WriteFS {
	return &writeFS{
		entries:  map[string]writeFSEntry{},
		contents: map[string][]byte{},
	}
}

func (wfs *writeFS) Create(ctx context.Context, name string, filemode fs.FileMode) (io.WriteCloser, string, error) {
	wfs.Lock()
	defer wfs.Unlock()
	if _, ok := wfs.entries[name]; ok {
		return nil, "", os.ErrExist
	}
	entry := writeFSEntry{mode: filemode, update: time.Now()}
	wfs.entries[name] = entry
	wfs.contents[name] = nil
	return &writeCloser{wfs: wfs, name: name}, name, nil
}

func (wfs *writeFS) Open(ctx context.Context, name string) (fs.File, error) {
	wfs.Lock()
	defer wfs.Unlock()
	entry, ok := wfs.entries[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	contents := wfs.contents[name]
	cpy := make([]byte, len(contents))
	copy(cpy, contents)
	info := NewInfo(name, len(cpy), entry.mode, entry.update, false, nil)
	return NewFile(&BufferCloser{bytes.NewBuffer(cpy)}, info), nil
}

func (wfs *writeFS) append(file string, buf []byte) {
	wfs.Lock()
	defer wfs.Unlock()
	entry := wfs.entries[file]
	entry.update = time.Now()
	wfs.entries[file] = entry
	wfs.contents[file] = append(wfs.contents[file], buf...)
}

type writeCloser struct {
	wfs  *writeFS
	name string
}

func (wc *writeCloser) Write(buf []byte) (int, error) {
	wc.wfs.append(wc.name, buf)
	return len(buf), nil
}

func (wc *writeCloser) Close() error {
	return nil
}
