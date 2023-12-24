// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package file

import (
	"os"
)

// symlinkInfo returns a file.Info appropriate for a symlink.
func symlinkInfo(_ string, info os.FileInfo) (Info, error) {
	return NewInfoFromFileInfo(info), nil
}
