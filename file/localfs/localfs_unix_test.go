// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package localfs_test

import (
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/localfs"
)

func TestMergeXAttr(t *testing.T) {
	ctx := context.Background()
	x := file.XAttr{
		UID:       1,
		GID:       2,
		Device:    3,
		FileID:    4,
		Blocks:    5,
		Hardlinks: 6,
	}
	fs := localfs.New()
	xattr := fs.SysXAttr(nil, x)

	stat, ok := xattr.(*syscall.Stat_t)
	if !ok {
		t.Fatalf("got %T, want *syscall.Stat_t", xattr)
	}
	if got, want := stat.Blocks, int64(5); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	tmpdir := t.TempDir()
	info, err := fs.Stat(ctx, tmpdir)
	if err != nil {
		t.Fatal(err)
	}
	osStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("got %T, want *syscall.Stat_t", info.Sys())
	}

	xattr = fs.SysXAttr(osStat, x)
	stat, ok = xattr.(*syscall.Stat_t)
	if !ok {
		t.Fatalf("got %T, want *syscall.Stat_t", xattr)
	}
	if got, want := stat.Blocks, int64(5); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := stat.Blksize, osStat.Blksize; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWait(t *testing.T) {
	for {
		tmpdir := t.TempDir()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		// Impossibly short timeout.
		fs := localfs.New(localfs.WithScannerOpenWait(time.Nanosecond))
		sc := fs.LevelScanner(tmpdir)
		if sc.Scan(ctx, 1) {
			t.Errorf("expected scan to fail")
		}
		err := sc.Err()
		if err == nil || errors.Is(err, context.DeadlineExceeded) {
			continue
		}
		if strings.Contains(err.Error(), "took too long") {
			break
		}
		t.Errorf("missing or wrong error: %v", err)
	}
}
