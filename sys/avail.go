// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package sys provides system-level utilities that are supported across
// different operating systems.
package sys

type filesystemInfo struct {
	BlockSize   int
	Blocks      int64
	BlocksFree  int64
	BlocksAvail int64
}

// AvailableBytes returns the number of available bytes on the filesystem
// where the file is located.
func AvailableBytes(filename string) (int64, error) {
	fi, err := statFS(filename)
	if err != nil {
		return 0, err
	}
	return fi.BlocksAvail * int64(fi.BlockSize), nil
}
