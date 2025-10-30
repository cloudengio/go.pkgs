// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package win32testutil

import (
	"golang.org/x/sys/windows"
)

func explicitAccess(mode windows.ACCESS_MODE, perms windows.ACCESS_MASK, sid *windows.SID) windows.EXPLICIT_ACCESS {
	return windows.EXPLICIT_ACCESS{
		AccessPermissions: perms,
		AccessMode:        mode,
		Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(sid),
		},
	}
}

func sidAllowDeny(mode windows.ACCESS_MODE, perms windows.ACCESS_MASK, path string) error {
	owner, err := windows.StringToSid("S-1-3-0")
	if err != nil {
		return nil
	}
	dacl, err := windows.ACLFromEntries(
		[]windows.EXPLICIT_ACCESS{explicitAccess(mode, perms, owner)},
		nil)
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

// MakeInaccessibleToOwner makes path inaccessible to its owner.
func MakeInaccessibleToOwner(path string) error {
	return sidAllowDeny(windows.DENY_ACCESS, 0, path)
}

// MakeAcessibleToOwner makes path accessible to its owner.
func MakeAccessibleToOwner(path string) error {
	return sidAllowDeny(windows.GRANT_ACCESS, windows.GENERIC_ALL, path)
}
