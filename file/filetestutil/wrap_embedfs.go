// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"embed"
	"io/fs"

	"cloudeng.io/file"
	"cloudeng.io/file/localfs"
)

// WrapEmbedFS wraps an embed.FS to implement file.FS.
func WrapEmbedFS(fs embed.FS) file.FS {
	return &fsFromFS{fs: fs, FS: localfs.New()}
}

type fsFromFS struct {
	fs fs.FS
	file.FS
}

func (f *fsFromFS) Open(name string) (fs.File, error) {
	return f.fs.Open(name)
}
