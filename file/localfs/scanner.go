// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"cloudeng.io/algo/container/stream"
	"cloudeng.io/file/filewalk"
)

type scanner struct {
	path     string
	openWait time.Duration
	prefetch int // see WithPrefetch.
	read     int // entries read so far.
	err      error
	file     *os.File
	done     bool
	entries  []fs.DirEntry
	pf       *stream.Scanner[fs.DirEntry] // non-nil once prefetching.
}

func (s *scanner) Contents() []filewalk.Entry {
	return newContents(s.entries)
}

// Close releases the file and stops any prefetching. It is called
// automatically once the scan completes or fails.
func (s *scanner) Close() error {
	s.done = true
	if s.pf != nil {
		// Wait for the prefetcher to stop using the file.
		_ = s.pf.Close()
		s.pf = nil
	}
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
	return nil
}

func (s *scanner) Scan(ctx context.Context, n int) bool {
	if n == 0 || s.done {
		return false
	}
	// Check for ctx.Done() before performing any IO since
	// readdir operations may be very slow.
	select {
	case <-ctx.Done():
		s.err = ctx.Err()
		s.Close()
		return false
	default:
	}
	if s.file == nil {
		if !s.open(ctx, s.path, s.openWait) {
			s.done = true
			return false
		}
	}
	if s.pf != nil {
		return s.scanPrefetched(n)
	}
	dirEntries, err := s.file.ReadDir(n)
	if err != nil {
		s.entries = nil
		s.Close()
		if !errors.Is(err, io.EOF) {
			s.err = err
		}
		return false
	}
	s.entries = dirEntries
	s.read += len(dirEntries)
	if s.prefetch > 0 && s.read >= s.prefetch && len(dirEntries) == n {
		s.startPrefetch(ctx, n)
	}
	return true
}

// startPrefetch reads the remainder of the directory in the background.
func (s *scanner) startPrefetch(ctx context.Context, n int) {
	f := s.file
	popn := func(_ context.Context, items []fs.DirEntry) (int, bool, error) {
		entries, err := f.ReadDir(len(items))
		c := copy(items, entries)
		if errors.Is(err, io.EOF) {
			return c, true, nil
		}
		return c, false, err
	}
	s.pf = stream.NewScanner(ctx, popn, stream.WithBatchSize(n))
}

func (s *scanner) scanPrefetched(n int) bool {
	s.entries = s.entries[:0]
	for len(s.entries) < n && s.pf.Scan() {
		s.entries = append(s.entries, s.pf.Data())
	}
	if len(s.entries) > 0 {
		return true
	}
	s.err = s.pf.Err()
	s.Close()
	return false
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
	return &scanner{path: prefix, openWait: f.opts.scannerOpenWait, prefetch: f.opts.scannerPrefetch}
}

func (f *R) LevelScanner(prefix string) filewalk.LevelScanner {
	return f.T.LevelScanner(filepath.Join(f.root, prefix))
}

func newContents(des []fs.DirEntry) []filewalk.Entry {
	c := make([]filewalk.Entry, len(des))
	for i, de := range des {
		c[i] = filewalk.Entry{Name: de.Name(), Type: de.Type()}
	}
	return c
}
