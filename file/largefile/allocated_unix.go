// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package largefile

import (
	"fmt"
	"os"
	"syscall"
)

func blockInfo(file *os.File) (int64, int, error) {
	fi, err := file.Stat()
	if err != nil {
		return 0, 0, err
	}
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("invalid fileinfo.Sys() type: %T", fi.Sys())
	}
	return s.Blocks, int(s.Blksize), nil
}

func allocated(file *os.File, size int64) (bool, error) {
	nBlocks, blksize, err := blockInfo(file)
	if err != nil {
		return false, err
	}
	return nBlocks*int64(blksize) >= size, nil
}
