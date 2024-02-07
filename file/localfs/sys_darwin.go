// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build darwin
package localfs

import (
	"fmt"
	"syscall"

	"cloudeng.io/file"
)

func xAttr(pathname string, fi file.Info) (file.XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return file.XAttr{}, fmt.Errorf("no system set for %v", pathname)
	}
	if s, ok := si.(*syscall.Stat_t); ok {
		return file.XAttr{
			UID:       int64(s.Uid),
			GID:       int64(s.Gid),
			Device:    uint64(s.Dev),
			FileID:    s.Ino,
			Blocks:    s.Blocks,
			Hardlinks: uint64(s.Nlink),
		}, nil
	}
	return file.XAttr{}, fmt.Errorf("unrecognised system information %T for %v", si, pathname)
}

func mergeXAttr(existing any, xattr file.XAttr) any {
	n := &syscall.Stat_t{}
	ex, ok := existing.(*syscall.Stat_t)
	if ok {
		*n = *ex
	}
	n.Uid = uint32(xattr.UID & 0xffffffff)
	n.Gid = uint32(xattr.GID & 0xffffffff)
	n.Dev = int32(xattr.Device & 0xffffffff)
	n.Ino = xattr.FileID
	n.Blocks = xattr.Blocks
	n.Nlink = uint16(xattr.Hardlinks & 0xffff)
	return n
}
