// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build linux && arm64
package localfs

import (
	"fmt"
	"syscall"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

func xAttr(pathname string, fi file.Info) (filewalk.XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return filewalk.XAttr{}, fmt.Errorf("no system set for %v", pathname)
	}
	if s, ok := si.(*syscall.Stat_t); ok {
		return filewalk.XAttr{
			UID:       uint64(s.Uid),
			GID:       uint64(s.Gid),
			Device:    s.Dev,
			FileID:    s.Ino,
			Blocks:    s.Blocks,
			Hardlinks: uint64(s.Nlink),
		}, nil
	}
	return filewalk.XAttr{}, fmt.Errorf("unrecognised system information %T for %v", si, pathname)
}

func newXAttr(xattr filewalk.XAttr) any {
	return &syscall.Stat_t{
		Uid:    uint32(xattr.UID & 0xffffffff),
		Gid:    uint32(xattr.GID & 0xffffffff),
		Dev:    xattr.Device,
		Ino:    xattr.FileID,
		Blocks: xattr.Blocks,
		Nlink:  uint32(xattr.Hardlinks & 0xffffffff),
	}
}
