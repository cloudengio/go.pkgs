// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package filewalk

import (
	"os"
	"path/filepath"
)

// need to read the symlink to determine its size on windows.
func symlinkSize(path string, info os.FileInfo) int64 {
	s, err := os.Readlink(filepath.Join(path, info.Name()))
	if err != nil {
		return info.Size()
	}
	return int64(len(s))

}
