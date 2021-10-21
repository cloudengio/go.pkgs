// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package internal

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32      = windows.MustLoadDLL("advapi32.dll")
	getSecInfo    = advapi32.MustFindProc("GetNamedSecurityInfoW")
	setSecInfo    = advapi32.MustFindProc("SetNamedSecurityInfoW")
	setACLEntries = advapi32.MustFindProc("SetEntriesInAclW")
)

/*
func getACLInfo(path string) (owner, group *windows.SID, dacl, secDesc windows.Handle, err error) {
	err = getSecInfo.Call(
		path,
		_SE_FILE_OBJECT,
		_OWNER_SECURITY_INFORMATION|_GROUP_SECURITY_INFORMATION|_DACL_SECURITY_INFORMATION,
		&owner,
		&group,
		&dacl,
		nil,
		&secDesc,
	)
}*/

func denySID(sid *windows.SID) explicitAccess {
	return explicitAccess{
		AccessPermissions: 0,
		AccessMode:        _DENY_ACCESS,
		Inheritance:       _SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: trustee{
			TrusteeForm: _TRUSTEE_IS_SID,
			Name:        (*uint16)(unsafe.Pointer(sid)),
		},
	}
}

func MakeInaccessibleToOwner(path string) error {
	owner, err := windows.StringToSid("S-1-3-0")
	if err != nil {
		return nil
	}
	/*
		owner, group, dacl,  secDesc, err := getACLInfo(path)
		if err != nil {
			return err
		}
		defer func() {
			windows.LocalFree(dacl)
		 windows.LocalFree(secDesc)
		}()*/

	var acl windows.Handle
	aclEntries := []explicitAccess{denySID(owner)}
	ret, _, err := setACLEntries.Call(
		uintptr(len(aclEntries)),
		uintptr(unsafe.Pointer(&aclEntries[0])),
		uintptr(unsafe.Pointer(nil)), //uintptr(dacl),
		uintptr(unsafe.Pointer(&acl)),
	)
	if ret != 0 {
		return err
	}
	ret, _, err = setSecInfo.Call(
		uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(path))),
		uintptr(_SE_FILE_OBJECT),
		_DACL_SECURITY_INFORMATION|_PROTECTED_DACL_SECURITY_INFORMATION,
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
		uintptr(acl),
		uintptr(unsafe.Pointer(nil)),
	)
	if ret != 0 {
		return err
	}
	return nil
}
