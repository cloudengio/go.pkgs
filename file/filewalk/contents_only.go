// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"context"
	"io/fs"
	"sync"

	"cloudeng.io/errors"
	"cloudeng.io/file"
)

var (
	SkipAll = fs.SkipDir
	SkipDir = fs.SkipDir
)

// ContentsHandler can return an error of fs.SkipAll or fs.SkipDir to
// skip all subsequent content or the current directory only respectively.
// All other errors are treated as fatal. Note that SkipDir, depending
// on the order that entires are encountered may result in subdirectories
// being skipped also.
type ContentsHandler func(ctx context.Context, prefix string, contents []Entry, err error) error

type contentsOnly struct {
	fs   FS
	h    ContentsHandler
	mu   sync.Mutex
	stop bool
}

func (l *contentsOnly) isStopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stop
}

func (l *contentsOnly) setStop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stop = true
}

func (l *contentsOnly) Prefix(_ context.Context, _ *struct{}, _ string, _ file.Info, err error) (bool, file.InfoList, error) {
	return l.isStopped(), nil, err
}

func (l *contentsOnly) Contents(ctx context.Context, _ *struct{}, prefix string, contents []Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents)/10)
	files := make([]Entry, 0, len(contents))
	for _, c := range contents {
		if c.IsDir() {
			info, err := l.fs.Lstat(ctx, l.fs.Join(prefix, c.Name))
			if err != nil {
				return nil, err
			}
			children = append(children, info)
			continue
		}
		files = append(files, c)
	}
	if err := l.h(ctx, prefix, files, nil); err != nil {
		return nil, err
	}
	return children, nil
}

func (l *contentsOnly) Done(ctx context.Context, _ *struct{}, prefix string, err error) error {
	if err != nil {
		if errors.Is(err, fs.SkipDir) {
			return nil
		}
		if errors.Is(err, fs.SkipAll) {
			l.setStop()
			return nil
		}
		return l.h(ctx, prefix, nil, err)
	}
	return nil
}

// ContentsOnly provides a simplified API for walking the contents (files)
// of a directory hierarchy. Inovations of the ContentsHandler may be concurrent.
func ContentsOnly(ctx context.Context, fs FS, start string, h ContentsHandler, opts ...Option) error {
	wk := New[struct{}](fs, &contentsOnly{fs: fs, h: h}, opts...)
	return wk.Walk(ctx, start)
}
