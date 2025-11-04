// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package largefile

import (
	"os"

	"golang.org/x/sys/windows"
)

func allocated(file *os.File, size int64) (bool, error) {
	var fi windows.ByHandleFileInformation
	err := windows.GetFileInformationByHandle(file.Fd(), &fi)
	if err != nil {
		return false, err
	}
	var fiSize int64 = int64(fi.FileSizeHigh)<<32 | int64(fi.FileSizeLow)
	return fiSize >= size, nil
}
