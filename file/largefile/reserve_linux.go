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

func reserveSpace(_ context.Context, fs *os.File, size int64, _, concurrency int, progressCh chan<- int64) error {
	err := unix.Fallocate(int(fs.Fd()), 0, 0, size)
	if progessCh != nil {
		select {
		case progressCh <- size:
		default:
		}
		close(progressCh)
	}
	return err
}
