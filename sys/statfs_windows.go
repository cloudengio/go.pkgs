// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package sys

import (
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

func statFS(filename string) (*filesystemInfo, error) {
	dirname := filepath.Dir(filename)
	lpDirectoryName, _ := syscall.UTF16PtrFromString(dirname)

	var bytesAvailable, bytesTotal, bytesFree uint64
	// GetDiskFreeSpaceEx retrieves information about the amount of space available on a disk volume.
	err := windows.GetDiskFreeSpaceEx(
		lpDirectoryName,
		&bytesAvailable,
		&bytesTotal,
		&bytesFree,
	)
	if err != nil {
		return nil, err
	}

	return &filesystemInfo{
		BlockSize:   1,
		Blocks:      int64(bytesTotal),
		BlocksFree:  int64(bytesFree),
		BlocksAvail: int64(bytesAvailable),
	}, nil
}
