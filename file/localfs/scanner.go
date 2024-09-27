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
	"time"

	"cloudeng.io/file/filewalk"
)

type scanner struct {
	path     string
	openWait time.Duration
	err      error
	file     *os.File
	entries  []fs.DirEntry
}

func (s *scanner) Contents() []filewalk.Entry {
	return newContents(s.entries)
}

func (s *scanner) Scan(ctx context.Context, n int) bool {
	if n == 0 {
		return false
	}
	// Check for ctx.Done() before performing any IO since
	// readdir operations may be very slow.
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		return false
	default:
	}
	if s.file == nil {
		if !s.open(ctx, s.path, s.openWait) {
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

func (s *scanner) open(ctx context.Context, path string, waitDuration time.Duration) bool {
	if waitDuration == 0 {
		s.file, s.err = os.Open(path)
		return s.err == nil
	}
	return s.openTimed(ctx, path, waitDuration)
}

func (s *scanner) openTimed(ctx context.Context, path string, waitDuration time.Duration) bool {
	ch := make(chan openState, 1)
	start := time.Now()
	go func() {
		// This will leak a goroutine if os.Open hangs.
		f, err := os.Open(path)
		ch <- openState{file: f, err: err}
	}()
	after := time.NewTimer(waitDuration)
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		if !after.Stop() {
			<-after.C
		}
	case state := <-ch:
		s.file, s.err = state.file, state.err
		if !after.Stop() {
			<-after.C
		}
	case <-after.C:
		s.err = fmt.Errorf("os.Open took too long for: %v: %v", time.Since(start), path)
	}

	return s.err == nil
}

func NewLevelScanner(path string, openwait time.Duration) filewalk.LevelScanner {
	return &scanner{path: path, openWait: openwait}
}

func (f *T) LevelScanner(prefix string) filewalk.LevelScanner {
	return NewLevelScanner(prefix, f.opts.scannerOpenWait)
}

func newContents(des []fs.DirEntry) []filewalk.Entry {
	c := make([]filewalk.Entry, len(des))
	for i, de := range des {
		c[i] = filewalk.Entry{Name: de.Name(), Type: de.Type()}
	}
	return c
}
