// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webassets

import (
	"io/fs"
	"path"

	"cloudeng.io/io/reloadfs"
)

type relative struct {
	prefix string
	logger func(action reloadfs.Action, name, path string, err error)
	fs     fs.FS
}

// Open implements fs.FS.
func (r *relative) Open(name string) (fs.File, error) {
	full := path.Join(r.prefix, name)
	fs, err := r.fs.Open(full)
	if r.logger != nil {
		r.logger(reloadfs.Reused, name, full, err)
	}
	return fs, err
}

// RelativeFS wraps the supplied FS so that prefix is prepended
// to all of the paths fetched from it. This is generally useful
// when working with webservers where the FS containing files
// is created from 'assets/...' but the URL path to access them
// is at the root. So /index.html can be mapped to assets/index.html.
func RelativeFS(prefix string, fs fs.FS) fs.FS {
	return relativeFS(prefix, fs)
}

func relativeFS(prefix string, fs fs.FS) *relative {
	return &relative{prefix: prefix, logger: nil, fs: fs}
}
