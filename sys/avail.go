// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package sys

type filesystemInfo struct {
	BlockSize   int64
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
	return fi.BlocksAvail * fi.BlockSize, nil
}
