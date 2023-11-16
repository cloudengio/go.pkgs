// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package localfs

import (
	"os"

	"cloudeng.io/file"
)

// symlinkInfo returns a file.Info appropriate for a symlink.
func symlinkInfo(filename string, info os.FileInfo) (file.Info, error) {
	// on windows the only way to get the size of a symlink is to read it!
	s, err := os.Readlink(filename)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfo(
		info.Name(),
		int64(len(s)),
		info.Mode(),
		info.ModTime(),
		info.Sys()), nil
}
