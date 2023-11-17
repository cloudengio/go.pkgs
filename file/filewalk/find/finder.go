// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package find provides a filewalk.Handler that can be used to find files.
package find

import (
	"context"
	"io/fs"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/matcher"
)

type Found struct {
	Prefix string
	Name   string
}

// New returns a filewalk.Handler that can match on prefix/directory names as well
// as filenames using file.Matcher expressions. The prefixMatcher is applied to
// the prefix/directory and if prune is true no futher processing of that directory
// will take place. The fileMatcher is applied to the filename (without its parent).
func New(fs filewalk.FS, ch chan<- Found, prefixMatcher, fileMatcher matcher.T, prune bool) filewalk.Handler[struct{}] {
	return &handler{
		found:    ch,
		fs:       fs,
		pm:       prefixMatcher,
		fm:       fileMatcher,
		prune:    prune,
		needstat: fileMatcher.NeedsFileMode() || fileMatcher.NeedsModTime(),
	}
}

type handler struct {
	found    chan<- Found
	fs       filewalk.FS
	pm, fm   matcher.T
	needstat bool
	prune    bool
}

type nameValue struct {
	name string
}

func (nv nameValue) Name() string {
	return nv.name
}

func (nv nameValue) Mode() fs.FileMode {
	return 0
}

func (nv nameValue) ModTime() time.Time {
	return time.Time{}
}

func (h *handler) Prefix(_ context.Context, _ *struct{}, prefix string, fi file.Info, err error) (bool, file.InfoList, error) {
	if err != nil {
		return false, nil, err
	}
	if h.pm.Eval(fi) {
		h.found <- Found{Prefix: prefix}
		return h.prune, nil, nil
	}
	return false, nil, nil
}

func (h *handler) Contents(ctx context.Context, _ *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		filename := h.fs.Join(prefix, c.Name)
		var fi file.Info
		var val matcher.Value
		if h.needstat || c.IsDir() {
			var err error
			fi, err = h.fs.Stat(ctx, filename)
			if err != nil {
				return nil, err
			}
			val = fi
		} else {
			val = nameValue{name: c.Name}
		}
		if h.fm.Eval(val) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case h.found <- Found{Prefix: prefix, Name: c.Name}:
			}
		}
		if c.IsDir() {
			children = append(children, fi)
		}
	}
	return children, nil
}

func (h *handler) Done(_ context.Context, _ *struct{}, _ string, _ error) error {
	return nil
}
