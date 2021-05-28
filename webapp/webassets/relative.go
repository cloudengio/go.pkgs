package webassets

import (
	"io/fs"
	"path"
)

type relative struct {
	prefix string
	fs     fs.FS
}

// Open implements fs.FS.
func (r *relative) Open(name string) (fs.File, error) {
	name = path.Join(r.prefix, name)
	return r.fs.Open(name)
}

// RelativeFS wraps the supplied FS so that prefix is prepended
// to all of the paths fetched from it. This is generally useful
// when working with webservers where the FS containing files
// is created from 'assets/...' but the URL path to access them
// is at the root. So /index.html can be mapped to assets/index.html.
func RelativeFS(prefix string, fs fs.FS) fs.FS {
	return &relative{prefix, fs}
}
