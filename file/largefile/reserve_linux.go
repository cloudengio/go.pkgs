// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build linux

package largefile

import (
	"context"
	"os"

	"golang.org/x/sys/unix"
)

func reserveSpace(ctx context.Context, fs *os.File, size int64, blockSize, concurrency int) error {
	return unix.Fallocate(fs.Fd(), 0, 0, size)
}
