// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webassets

import (
	"io/fs"
	"log"
	"time"

	"cloudeng.io/io/reloadfs"
)

type assets struct {
	fs.FS
	logger      func(action reloadfs.Action, name, path string, err error)
	reloadAfter time.Time
	reloadFrom  string
	loadNew     bool
}

// AssetsOption represents an option to NewAssets.
type AssetsOption func(a *assets)

// EnableLogging enables logging using a built in logging function.
func EnableLogging() AssetsOption {
	return func(a *assets) {
		a.logger = fsLogger
	}
}

// UseLogger enables logging using the supplied logging function.
func UseLogger(logger func(action reloadfs.Action, name, path string, err error)) AssetsOption {
	return func(a *assets) {
		a.logger = logger
	}
}

// EnableReloading enables reloading of assets from the specified
// location if they have changed since 'after'; loadNew controls whether
// new files, ie. those that exist only in location, are loaded as opposed.
// See cloudeng.io/io/reloadfs.
func EnableReloading(location string, after time.Time, loadNew bool) AssetsOption {
	return func(a *assets) {
		a.reloadFrom = location
		a.reloadAfter = after
		a.loadNew = loadNew
	}
}

func fsLogger(action reloadfs.Action, name, path string, err error) {
	if err != nil {
		log.Printf("%v -> %v: %v: %v", name, path, action, err)
	} else {
		log.Printf("%v -> %v: %v", name, path, action)
	}
}

// NewAssets returns an fs.FS that is configured to be optional reloaded
// from the local filesystem or to be served directly from the supplied
// fs.FS. The EnableReloading option is used to enable reloading.
// Prefix is prepended to all names passed to the supplied fs.FS, which
// is typically obtained via go:embed. See RelativeFS for more details.
func NewAssets(prefix string, fsys fs.FS, opts ...AssetsOption) fs.FS {
	a := &assets{}
	for _, fn := range opts {
		fn(a)
	}
	if len(a.reloadFrom) == 0 {
		rfs := relativeFS(prefix, fsys)
		rfs.logger = a.logger
		a.FS = rfs
		return a
	}
	a.FS = reloadfs.New(a.reloadFrom,
		prefix,
		fsys,
		reloadfs.UseLogger(a.logger),
		reloadfs.ReloadAfter(a.reloadAfter),
		reloadfs.LoadNewFiles(a.loadNew),
	)
	return a
}
