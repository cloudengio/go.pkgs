// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package sys

import (
	"os"

	"golang.org/x/sys/unix"
)

func statFS(filename string) (*filesystemInfo, error) {
	fs, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	var buf unix.Statfs_t
	if err := unix.Fstatfs(int(fs.Fd()), &buf); err != nil {
		return nil, err
	}

	return &filesystemInfo{
		BlockSize:   int(buf.Bsize),
		Blocks:      int64(buf.Blocks),
		BlocksFree:  int64(buf.Bfree),
		BlocksAvail: int64(buf.Bavail),
	}, nil
}
