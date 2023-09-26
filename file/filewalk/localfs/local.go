// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

// T represents an instance of filewalk.FS for a local filesystem.
type T struct{}

func New() filewalk.FS {
	return &T{}
}

type scanner struct {
	err     error
	file    *os.File
	entries []fs.DirEntry
}

func (s *scanner) Contents() filewalk.Contents {
	return newContents(s.entries)
}

func (s *scanner) Scan(_ context.Context, n int) bool {
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

func NewLevelScanner(path string) filewalk.LevelScanner {
	f, err := os.Open(path)
	if err != nil {
		return &scanner{err: err}
	}
	return &scanner{file: f}
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

func (l *T) LStat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return file.Info{}, err
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

func newContents(des []fs.DirEntry) filewalk.Contents {
	c := filewalk.Contents{
		Entries: make([]filewalk.Entry, len(des)),
	}
	for i, de := range des {
		c.Entries[i] = filewalk.Entry{Name: de.Name(), Type: de.Type()}
	}
	return c
}
