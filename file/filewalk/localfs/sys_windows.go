// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build windows
package localfs

import (
	"syscall"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

type sysinfo struct {
	uid, gid       uint64
	device, fileID uint64
	blocks         int64
	hardlinks      int64
}

func xAttr(pathname string, fi file.Info) (filewalk.XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return getSysInfo(pathname)
	}
	switch s := si.(type) {
	case *sysinfo:
		return filewalk.XAttr{
			UID:       s.uid,
			GID:       s.gid,
			Device:    s.device,
			FileID:    s.fileID,
			Blocks:    s.blocks,
			Hardlinks: s.hardlinks,
		}, nil
	}
	return getSysInfo(pathname)
}

func newXAttr(xattr filewalk.XAttr) any {
	return &sysinfo{
		uid:       xattr.UID,
		gid:       xattr.GID,
		dev:       xattr.Device,
		ino:       xattr.FileID,
		blocks:    xattr.Blocks,
		hardlinks: xattr.Hardlinks,
	}
}

func packFileIndices(hi, low uint32) uint64 {
	return uint64(hi)<<32 | uint64(low)
}

func getSysInfo(pathname string) (filewalk.XAttr, error) {
	// taken from loadFileId in types_windows.go
	pathp, err := syscall.UTF16PtrFromString(pathname)
	if err != nil {
		return
	}
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS | syscall.FILE_FLAG_OPEN_REPARSE_POINT)
	h, err := windows.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, attrs, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	var d windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(h, &d); err != nil {
		return
	}
	size := uint64(d.FileSizeHigh)<<32 | uint64(d.FileSizeLow)
	blocks := (size + 512) / 512
	return filewalk.XAttr{
		UID:       0,
		GID:       0,
		Device:    uint64(d.VolumeSerialNumber),
		FileID:    packFileIndices(d.FileIndexHigh, d.FileIndexLow),
		Blocks:    blocks,
		Hardlinks: d.NumberOfLinks,
	}, nil
}
