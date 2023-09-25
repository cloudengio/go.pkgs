// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package filewalk

import (
	"os"
)

// symlinkSize returns the size of the symlinks.
func symlinkSize(_ string, info os.FileInfo) (int64, error) {
	return info.Size(), nil
}
