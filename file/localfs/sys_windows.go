// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package localfs

import (
	"fmt"
	"syscall"

	"cloudeng.io/file"
	"golang.org/x/sys/windows"
)

type sysinfo struct {
	user, group    string
	device, fileID uint64
	blocks         int64
	hardlinks      uint64
}

func xAttr(pathname string, fi file.Info) (file.XAttr, error) {
	si := fi.Sys()
	if si == nil {
		return getSysInfo(pathname)
	}
	switch s := si.(type) {
	case *sysinfo:
		return file.XAttr{
			UID:       -1,
			GID:       -1,
			User:      "",
			Group:     "",
			Device:    s.device,
			FileID:    s.fileID,
			Blocks:    s.blocks,
			Hardlinks: s.hardlinks,
		}, nil
	}
	return getSysInfo(pathname)
}

func mergeXAttr(existing any, xattr file.XAttr) any {
	n := &sysinfo{}
	ex, ok := existing.(*sysinfo)
	if ok {
		*n = *ex
	}
	n.user = xattr.User
	n.group = xattr.Group
	n.device = xattr.Device
	n.fileID = xattr.FileID
	n.blocks = xattr.Blocks
	n.hardlinks = xattr.Hardlinks
	return n
}

func packFileIndices(hi, low uint32) uint64 {
	return uint64(hi)<<32 | uint64(low)
}

func getSysInfo(pathname string) (file.XAttr, error) {
	// taken from loadFileId in types_windows.go
	pathp, err := syscall.UTF16PtrFromString(pathname)
	if err != nil {
		return file.XAttr{}, fmt.Errorf("failed to convert %v to win32 utf16p: %v", pathname, err)
	}
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS | syscall.FILE_FLAG_OPEN_REPARSE_POINT)
	h, err := windows.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, attrs, 0)
	if err != nil {
		return file.XAttr{}, fmt.Errorf("CreateFile OPEN_EXISTING: failed to open %v: %v", pathname, err)
	}
	defer windows.CloseHandle(h)
	var d windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(h, &d); err != nil {
		return file.XAttr{}, fmt.Errorf("GetFileInformationByHandle for %v: %v", pathname, err)
	}
	size := int64(uint64(d.FileSizeHigh)<<32 | uint64(d.FileSizeLow))
	blocks := size / 512
	if blocks == 0 {
		blocks = 1
	}
	secinfo, err := windows.GetNamedSecurityInfo(pathname,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION)
	var usid, gsid string
	if err == nil {
		if sid, _, err := secinfo.Owner(); err == nil {
			usid = sid.String()
		}
		if sid, _, err := secinfo.Group(); err == nil {
			gsid = sid.String()
		}
	}
	return file.XAttr{
		UID:       -1,
		GID:       -1,
		User:      usid,
		Group:     gsid,
		Device:    uint64(d.VolumeSerialNumber),
		FileID:    packFileIndices(d.FileIndexHigh, d.FileIndexLow),
		Blocks:    blocks,
		Hardlinks: uint64(d.NumberOfLinks),
	}, nil
}
