// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalktestutil

import (
	"context"
	"sync"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

type Walker struct {
	sync.Mutex
	FS       filewalk.FS
	Prefixes []string
	Names    []string
}

func (w *Walker) Prefix(_ context.Context, _ *struct{}, prefix string, _ file.Info, _ error) (bool, file.InfoList, error) {
	w.Lock()
	w.Prefixes = append(w.Prefixes, prefix)
	w.Unlock()
	return false, nil, nil
}

func (w *Walker) Contents(ctx context.Context, _ *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		key := w.FS.Join(prefix, c.Name)
		if !c.IsDir() {
			w.Lock()
			w.Names = append(w.Names, key)
			w.Unlock()
			continue
		}
		info, err := w.FS.Stat(ctx, key)
		if err != nil {
			return nil, err
		}
		children = append(children, info)
	}
	return children, nil
}

func (w *Walker) Done(_ context.Context, _ *struct{}, _ string, err error) error {
	return err
}

func WalkContents(ctx context.Context, fs filewalk.FS, roots ...string) (prefixes, names []string, err error) {
	cw := &Walker{FS: fs}
	if err := filewalk.New(fs, cw).Walk(ctx, roots...); err != nil {
		return nil, nil, err
	}
	return cw.Prefixes, cw.Names, nil
}

func Scan(ctx context.Context, fs filewalk.FS, prefix string) ([]filewalk.Entry, error) {
	sc := fs.LevelScanner(prefix)
	found := []filewalk.Entry{}
	for sc.Scan(ctx, 1) {
		for _, c := range sc.Contents() {
			found = append(found, c)
		}
	}
	return found, sc.Err()
}

func ScanNames(ctx context.Context, fs filewalk.FS, prefix string) ([]string, error) {
	entries, err := Scan(ctx, fs, prefix)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = fs.Join(prefix, e.Name)
	}
	return names, nil
}
