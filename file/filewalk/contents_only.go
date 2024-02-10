// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"context"
	"sync"

	"cloudeng.io/file"
)

type ContentsHandler func(ctx context.Context, prefix string, contents []Entry, err error) error

type contentsOnly struct {
	mu sync.Mutex
	fs FS
	h  ContentsHandler
}

func (l *contentsOnly) Prefix(_ context.Context, _ *struct{}, _ string, _ file.Info, err error) (bool, file.InfoList, error) {
	return false, nil, err
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
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.h(ctx, prefix, files, nil); err != nil {
		return nil, err
	}
	return children, nil
}

func (l *contentsOnly) Done(ctx context.Context, _ *struct{}, prefix string, err error) error {
	if err != nil {
		return l.h(ctx, prefix, nil, err)
	}
	return err
}

// ContentsOnly provides a simplified API for walking the contents (files)
// of a directory hierarchy. Inovations of the ContentsHandler are serialized
// using a mutex.
func ContentsOnly(ctx context.Context, fs FS, start string, h ContentsHandler, opts ...Option) error {
	wk := New[struct{}](fs, &contentsOnly{fs: fs, h: h}, opts...)
	return wk.Walk(ctx, start)
}
