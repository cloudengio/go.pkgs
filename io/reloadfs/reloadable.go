// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package reloadfs provides an implemtation of fs.FS whose contents
// can be selectively reloaded from disk. This allows for default contents
// to be embedded in a binary, typically via go:embed, to be overridden at
// run time if so desired. This can be useful for configuration files
// as well web server assets.
package reloadfs

import (
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"
)

type reloadable struct {
	sync.Mutex
	embedded    fs.FS
	root        string
	prefix      string
	logger      *slog.Logger
	stat        map[string]fs.FileInfo
	reloadAfter time.Time
	loadNew     bool
}

func (r *reloadable) reloadablePath(p string) string {
	return path.Join(r.root, r.prefix, p)
}

func (r *reloadable) embeddedPath(p string) string {
	return path.Join(r.prefix, p)
}

func (r *reloadable) embeddedIsStale(embedded, loaded fs.FileInfo) bool {
	return loaded.ModTime().After(r.reloadAfter) || loaded.Size() != embedded.Size()
}

func (r *reloadable) statEmbedded(name string) (fs.FileInfo, error) {
	r.Lock()
	fi, ok := r.stat[name]
	r.Unlock()
	if ok {
		return fi, nil
	}
	f, err := r.embedded.Open(r.embeddedPath(name))
	if err != nil {
		return nil, err
	}
	fi, err = f.Stat()
	if err != nil {
		return nil, err
	}
	r.Lock()
	r.stat[name] = fi
	r.Unlock()
	return fi, nil
}

func (r *reloadable) reload(name string) (bool, bool, error) {
	r.logger.Debug("reload", "name", name, "embedded", r.embeddedPath(name), "disk", r.reloadablePath(name))
	ondisk, err := os.Stat(r.reloadablePath(name))
	if err == nil {
		inram, err := r.statEmbedded(name)
		if err != nil {
			if !os.IsNotExist(err) {
				return false, true, err
			}
			if r.loadNew {
				return true, true, nil
			}
			return false, true, os.ErrNotExist
		}
		r.logger.Debug("reload check",
			"embedded_size", inram.Size(),
			"embedded_modtime_threshold", r.reloadAfter,
			"ondisk_size", ondisk.Size(),
			"ondisk_modtime", ondisk.ModTime(),
			"stale", r.embeddedIsStale(inram, ondisk))
		return r.embeddedIsStale(inram, ondisk), false, nil
	}
	if os.IsNotExist(err) {
		r.logger.Debug("reload: on disk", "name", name, "path", r.reloadablePath(name), "err", err)
		return false, false, nil
	}
	return false, false, err
}

// Open implements fs.FS.
func (r *reloadable) Open(name string) (fs.File, error) {
	if name != "" && !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	shouldReload, isNew, err := r.reload(name)
	if err != nil {
		rp := r.reloadablePath(name)
		if isNew && !r.loadNew && os.IsNotExist(err) {
			r.logger.Info("new files not allowed", "name", name, "path", rp, "err", err)
		}
		return nil, &fs.PathError{
			Op:   "open",
			Path: rp,
			Err:  err,
		}
	}
	if !shouldReload {
		ep := r.embeddedPath(name)
		f, err := r.embedded.Open(ep)
		r.logger.Info("reused", "name", name, "path", ep, "err", err)
		return f, err
	}
	rl := r.reloadablePath(name)
	f, err := os.Open(rl)
	action := ReloadedExisting
	if isNew {
		action = ReloadedNewFile
	}
	r.logger.Info(action.String(), "name", name, "path", rl, "err", err)
	return f, err
}

// Option represents an option to ReloadableFS.
type Option func(*reloadable)

// WithLogger provides a logger to be used by the underlying implementation.
func WithLogger(logger *slog.Logger) Option {
	return func(r *reloadable) {
		r.logger = logger
	}
}

// WithNewFiles controls whether files that exist only in file system
// and not in the embedded FS are returned. If false, only files that
// exist in the embedded FS may be reloaded from the new FS.
func WithNewFiles(a bool) Option {
	return func(r *reloadable) {
		r.loadNew = a
	}
}

// WithReloadAfter sets the time after which assets are to be reloaded
// rather than reused. Note that the current implementation of go:embed
// does not record
func WithReloadAfter(t time.Time) Option {
	return func(r *reloadable) {
		r.reloadAfter = t
	}
}

// Action represents the action taken by the implementation of fs.FS.
type Action int

// The set of available actions.
const (
	ReloadedExisting Action = iota
	ReloadedNewFile
	Reused
	NewFilesNotAllowed
)

func (a Action) String() string {
	switch a {
	case ReloadedExisting:
		return "reloaded existing"
	case ReloadedNewFile:
		return "reloaded new file"
	case Reused:
		return "reused"
	case NewFilesNotAllowed:
		return "new files not allowed"
	default:
		return "unknown action"
	}
}

// New returns a new fs.FS that will dynamically reload files that
// have either been changed, or optionally only exist, in the filesystem
// as compared to the embedded files. See ReloadAfter and LoadNewFiles. If
// ReloadAfter is not specified the current time is assumed, that is,
// files whose modification time is after that will be reloaded. For a file
// to be reloaded either its modification time or size have to differ. Comparing
// sizes can catch cases where the file system time granularity is coarse. This
// leaves the one corner case of a file being modified without changing either
// its size or modification time.
//
// The prefix is prepended to the argument supplied to Open to obtain
// the full name passed to the supplied FS below. The root and prefix
// are prepended to obtain the name to be used in the newly returned FS,
// typically a local file system. For example, given:
//
//	//go:embed assets/*.html
//	var htmlAssets embed.FS
//
// With the reloadable assets in /tmp/overrides, then New should be called as:
//
//	New("/tmp/overrides", "assets", htmlAssets)
//
// Currently files are reloaded when Open'ed, in the future support may be
// provided to watch for changes and reload (or update metdata) those
// ahead of time. Reloaded files are not cached and will be reloaded on
// every access.
func New(root, prefix string, embedded fs.FS, opts ...Option) fs.FS {
	r := &reloadable{
		embedded:    embedded,
		root:        root,
		prefix:      prefix,
		stat:        make(map[string]fs.FileInfo),
		reloadAfter: time.Now(),
	}
	for _, fn := range opts {
		fn(r)
	}
	if r.logger == nil {
		r.logger = slog.New(slog.DiscardHandler)
	}
	r.logger = r.logger.With("reloadfs root", r.root, "reloadfs prefix", r.prefix)
	return r
}
