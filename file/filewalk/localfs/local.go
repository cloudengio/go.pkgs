// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

// T represents an instance of filewalk.FS for a local filesystem.
type T struct{}

func New() filewalk.FS {
	return &T{}
}

type scanner struct {
	path    string
	err     error
	file    *os.File
	entries []fs.DirEntry
}

func (s *scanner) Contents() []filewalk.Entry {
	return newContents(s.entries)
}

func (s *scanner) Scan(ctx context.Context, n int) bool {
	// Check for ctx.Done() before performing any IO since
	// readdir operations may be very slow.
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		return false
	default:
	}
	if s.file == nil {
		if !s.open(ctx, s.path) {
			return false
		}
	}
	dirEntries, err := s.file.ReadDir(n)
	if err != nil {
		s.entries = nil
		s.file.Close()
		if err = io.EOF; err != nil {
			return false
		}
		s.err = err
		return false
	}
	s.entries = dirEntries
	return true
}

func (s *scanner) Err() error {
	return s.err
}

type openState struct {
	file *os.File
	err  error
}

func (s *scanner) open(ctx context.Context, path string) bool {
	ch := make(chan openState, 1)
	start := time.Now()
	go func() {
		// This will leak a gorooutine if os.Open hangs.
		f, err := os.Open(path)
		ch <- openState{file: f, err: err}
	}()
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		return false
	case state := <-ch:
		s.file, s.err = state.file, state.err
	case <-time.After(time.Minute):
		s.err = fmt.Errorf("os.Open took too %v long for: %v", time.Since(start), path)
	}
	return s.err == nil
}

func NewLevelScanner(path string) filewalk.LevelScanner {
	return &scanner{path: path}
}

func (l *T) Open(path string) (fs.File, error) {
	return os.Open(path)
}

func (l *T) OpenCtx(_ context.Context, path string) (fs.File, error) {
	return os.Open(path)
}

func (l *T) Scheme() string {
	return "file"
}

func (l *T) LevelScanner(prefix string) filewalk.LevelScanner {
	return NewLevelScanner(prefix)
}

func (l *T) Readlink(_ context.Context, path string) (string, error) {
	return os.Readlink(path)
}

func (l *T) Stat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfoFromFileInfo(info), nil
}

func (l *T) Lstat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return file.Info{}, err
	}
	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		return symlinkInfo(path, info)
	}
	return file.NewInfoFromFileInfo(info), nil
}

func (l *T) Join(components ...string) string {
	return filepath.Join(components...)
}

func (l *T) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (l *T) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func (l *T) XAttr(ctx context.Context, path string, fi file.Info) (filewalk.XAttr, error) {
	return xAttr(path, fi)
}

func (l *T) NewXattr(x filewalk.XAttr) any {
	return newXAttr(x)
}

func newContents(des []fs.DirEntry) []filewalk.Entry {
	c := make([]filewalk.Entry, len(des))
	for i, de := range des {
		c[i] = filewalk.Entry{Name: de.Name(), Type: de.Type()}
	}
	return c
}
