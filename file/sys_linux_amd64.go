// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build linux && amd64
package file

import (
	"fmt"
	"syscall"
)

func xAttr(pathname string, fi Info) (XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return XAttr{}, fmt.Errorf("no system set for %v", pathname)
	}
	if s, ok := si.(*syscall.Stat_t); ok {
		return XAttr{
			UID:       int64(s.Uid),
			GID:       int64(s.Gid),
			Device:    s.Dev,
			FileID:    s.Ino,
			Blocks:    s.Blocks,
			Hardlinks: s.Nlink,
		}, nil
	}
	return XAttr{}, fmt.Errorf("unrecognised system information %T for %v", si, pathname)
}

func mergeXAttr(existing any, xattr XAttr) any {
	n := &syscall.Stat_t{}
	ex, ok := existing.(*syscall.Stat_t)
	if ok {
		*n = *ex
	}
	n.Uid = uint32(xattr.UID & 0xffffffff)
	n.Gid = uint32(xattr.GID & 0xffffffff)
	n.Dev = xattr.Device
	n.Ino = xattr.FileID
	n.Blocks = xattr.Blocks
	n.Nlink = xattr.Hardlinks
	return n
}
