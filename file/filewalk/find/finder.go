// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package find provides a filewalk.Handler that can be used to locate
// prefixes/directories and files based on file.Matcher expressions.
package find

import (
	"context"
	"io/fs"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/matcher"
)

// Found is used to send matches or errors to the client.
type Found struct {
	Prefix string
	Name   string
	Err    error
}

type needModTime struct{}

func (nt needModTime) ModTime() time.Time { return time.Time{} }

// NeedsStat determines if either of the supplied matcher.T's include
// operands that would require a call to fs.Stat or fs.Lstat.
func NeedsStat(prefixMatcher, fileMatcher matcher.T) bool {
	return prefixMatcher.Needs(needModTime{}) || fileMatcher.Needs(needModTime{})
}

// Option represents an option for New.
type Option func(*options)

type options struct {
	prune           bool
	needStat        bool
	followSoftlinks bool
	prefixMatcher   matcher.T
	fileMatcher     matcher.T
}

// WithStat specifies that the filewalk.Handler should call fs.Stat or fs.Lstat
// for files. Note that stat is always called for directories.
func WithStat(v bool) Option {
	return func(o *options) {
		o.needStat = v
	}
}

// WithPrune specifies that the filewalk.Handler should prune directories
// that match the prefixMatcher. That is, once a directory is matched
// no subdirectories will be examined.
func WithPrune(v bool) Option {
	return func(o *options) {
		o.prune = v
	}
}

// WithPrefixMatcher specifies the matcher.T to use for matching prefixes/directories.
// If none is supplied then no matches will be returned. The matcher.T is applied
// to the full path of the prefix/directory.
func WithPrefixMatcher(m matcher.T) Option {
	return func(o *options) {
		o.prefixMatcher = m
	}
}

// WithFileMatcher specifies the matcher.T to use for matching filenames.
// If none is supplied then no matches will be returned. The matcher.T is applied
// to name of the entry within a prefix/directory.
func WithFileMatcher(m matcher.T) Option {
	return func(o *options) {
		o.fileMatcher = m
	}
}

// WithFollowSoftlinks specifies that the filewalk.Handler should follow
// softlinks by calling fs.Stat rather than the default of calling fs.Lstat.
func WithFollowSoftlinks(v bool) Option {
	return func(o *options) {
		o.followSoftlinks = v
	}
}

// New returns a filewalk.Handler that can match on prefix/directory names as well
// as filenames using file.Matcher expressions. The prefixMatcher is applied to
// the prefix/directory and if prune is true no further processing of that directory
// will take place. The fileMatcher is applied to the filename (without its parent).
func New(fs filewalk.FS, ch chan<- Found, opts ...Option) filewalk.Handler[struct{}] {
	h := &handler{found: ch, fs: fs}
	for _, fn := range opts {
		fn(&h.options)
	}
	return h
}

type handler struct {
	found chan<- Found
	fs    filewalk.FS
	options
}

type nameAndType struct {
	prefix string
	typ    fs.FileMode
}

func (pn nameAndType) Name() string {
	return pn.prefix
}

func (pn nameAndType) Type() fs.FileMode {
	return pn.typ
}

func (h *handler) Prefix(_ context.Context, _ *struct{}, prefix string, fi file.Info, err error) (bool, file.InfoList, error) {
	if err != nil {
		return false, nil, err
	}
	if h.prefixMatcher.Eval(fi) {
		h.found <- Found{Prefix: prefix}
		return h.prune, nil, nil
	}
	return false, nil, nil
}

func (h *handler) handleStat(ctx context.Context, prefix, name string) (fi file.Info, err error) {
	filename := h.fs.Join(prefix, name)
	if h.followSoftlinks {
		fi, err = h.fs.Stat(ctx, filename)
	} else {
		fi, err = h.fs.Lstat(ctx, filename)
	}
	if err == nil {
		return
	}
	select {
	case <-ctx.Done():
		return file.Info{}, ctx.Err()
	case h.found <- Found{Prefix: prefix, Name: name, Err: err}:
	}
	return fi, nil
}

func (h *handler) Contents(ctx context.Context, _ *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		var fi file.Info
		var val any
		if h.needStat || c.IsDir() {
			var err error
			fi, err = h.handleStat(ctx, prefix, c.Name)
			if err != nil {
				return nil, err
			}
			val = fi
		} else {
			val = nameAndType{c.Name, c.Type}
		}
		if c.IsDir() {
			children = append(children, fi)
			continue
		}
		if h.fileMatcher.Eval(val) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case h.found <- Found{Prefix: prefix, Name: c.Name}:
			}
		}
	}
	return children, nil
}

func (h *handler) Done(ctx context.Context, _ *struct{}, prefix string, err error) error {
	if err != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case h.found <- Found{Prefix: prefix, Err: err}:
		}
	}
	return nil
}
