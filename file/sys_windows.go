// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build windows
package file

import (
	"fmt"
	"slices"
	"syscall"

	"golang.org/x/sys/windows"
)

type sysinfo struct {
	userinfo       []byte
	groupinfo      []byte
	device, fileID uint64
	blocks         int64
	hardlinks      uint64
}

func xAttr(pathname string, fi Info) (XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return getSysInfo(pathname)
	}
	switch s := si.(type) {
	case *sysinfo:
		return XAttr{
			UID:       -1,
			GID:       -1,
			UserInfo:  nil,
			GroupInfo: nil,
			Device:    s.device,
			FileID:    s.fileID,
			Blocks:    s.blocks,
			Hardlinks: s.hardlinks,
		}, nil
	}
	return getSysInfo(pathname)
}

func mergeXAttr(existing any, xattr XAttr) any {
	n := &sysinfo{}
	ex, ok := existing.(*sysinfo)
	if ok {
		*n = *ex
	}
	if xattr.UserInfo != nil {
		n.userinfo = slices.Clone(xattr.UserInfo)
	}
	if xattr.GroupInfo != nil {
		n.groupinfo = slices.Clone(xattr.GroupInfo)
	}
	n.device = xattr.Device
	n.fileID = xattr.FileID
	n.blocks = xattr.Blocks
	n.hardlinks = xattr.Hardlinks
	return n
}

func packFileIndices(hi, low uint32) uint64 {
	return uint64(hi)<<32 | uint64(low)
}

func getSysInfo(pathname string) (XAttr, error) {
	// taken from loadFileId in types_windows.go
	pathp, err := syscall.UTF16PtrFromString(pathname)
	if err != nil {
		return XAttr{}, fmt.Errorf("failed to convert %v to win32 utf16p: %v", pathname, err)
	}
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS | syscall.FILE_FLAG_OPEN_REPARSE_POINT)
	h, err := windows.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, attrs, 0)
	if err != nil {
		return XAttr{}, fmt.Errorf("CreateFile OPEN_EXISTING: failed to open %v: %v", pathname, err)
	}
	defer windows.CloseHandle(h)
	var d windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(h, &d); err != nil {
		return XAttr{}, fmt.Errorf("GetFileInformationByHandle for %v: %v", pathname, err)
	}
	size := int64(uint64(d.FileSizeHigh)<<32 | uint64(d.FileSizeLow))
	blocks := size / 512
	if blocks == 0 {
		blocks = 1
	}
	return XAttr{
		UID:       -1,
		GID:       -1,
		UserInfo:  nil, // TODO(cnicolaou): get SID etc.
		GroupInfo: nil, // TODO(cnicolaou): get SID etc.
		Device:    uint64(d.VolumeSerialNumber),
		FileID:    packFileIndices(d.FileIndexHigh, d.FileIndexLow),
		Blocks:    blocks,
		Hardlinks: uint64(d.NumberOfLinks),
	}, nil
}
