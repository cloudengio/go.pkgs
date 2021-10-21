// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package internal

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379593.aspx
const (
	_SE_UNKNOWN_OBJECT_TYPE = iota
	_SE_FILE_OBJECT
	_SE_SERVICE
	_SE_PRINTER
	_SE_REGISTRY_KEY
	_SE_LMSHARE
	_SE_KERNEL_OBJECT
	_SE_WINDOW_OBJECT
	_SE_DS_OBJECT
	_SE_DS_OBJECT_ALL
	_SE_PROVIDER_DEFINED_OBJECT
	_SE_WMIGUID_OBJECT
	_SE_REGISTRY_WOW64_32KEY
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379573.aspx
const (
	_OWNER_SECURITY_INFORMATION               = 0x00001
	_GROUP_SECURITY_INFORMATION               = 0x00002
	_DACL_SECURITY_INFORMATION                = 0x00004
	_SACL_SECURITY_INFORMATION                = 0x00008
	_LABEL_SECURITY_INFORMATION               = 0x00010
	_ATTRIBUTE_SECURITY_INFORMATION           = 0x00020
	_SCOPE_SECURITY_INFORMATION               = 0x00040
	_PROCESS_TRUST_LABEL_SECURITY_INFORMATION = 0x00080
	_BACKUP_SECURITY_INFORMATION              = 0x10000

	_PROTECTED_DACL_SECURITY_INFORMATION   = 0x80000000
	_PROTECTED_SACL_SECURITY_INFORMATION   = 0x40000000
	_UNPROTECTED_DACL_SECURITY_INFORMATION = 0x20000000
	_UNPROTECTED_SACL_SECURITY_INFORMATION = 0x10000000
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379284.aspx
const (
	_NO_MULTIPLE_TRUSTEE = iota
	_TRUSTEE_IS_IMPERSONATE
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379638.aspx
const (
	_TRUSTEE_IS_SID = iota
	_TRUSTEE_IS_NAME
	_TRUSTEE_BAD_FORM
	_TRUSTEE_IS_OBJECTS_AND_SID
	_TRUSTEE_IS_OBJECTS_AND_NAME
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379639.aspx
const (
	_TRUSTEE_IS_UNKNOWN = iota
	_TRUSTEE_IS_USER
	_TRUSTEE_IS_GROUP
	_TRUSTEE_IS_DOMAIN
	_TRUSTEE_IS_ALIAS
	_TRUSTEE_IS_WELL_KNOWN_GROUP
	_TRUSTEE_IS_DELETED
	_TRUSTEE_IS_INVALID
	_TRUSTEE_IS_COMPUTER
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa374899.aspx
const (
	_NOT_USED_ACCESS = iota
	_GRANT_ACCESS
	_SET_ACCESS
	_DENY_ACCESS
	_REVOKE_ACCESS
	_SET_AUDIT_SUCCESS
	_SET_AUDIT_FAILURE
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa446627.aspx
const (
	_NO_INHERITANCE                     = 0x0
	_SUB_OBJECTS_ONLY_INHERIT           = 0x1
	_SUB_CONTAINERS_ONLY_INHERIT        = 0x2
	_SUB_CONTAINERS_AND_OBJECTS_INHERIT = 0x3
	_INHERIT_NO_PROPAGATE               = 0x4
	_INHERIT_ONLY                       = 0x8

	_OBJECT_INHERIT_ACE       = 0x1
	_CONTAINER_INHERIT_ACE    = 0x2
	_NO_PROPAGATE_INHERIT_ACE = 0x4
	_INHERIT_ONLY_ACE         = 0x8
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa379636.aspx
type trustee struct {
	MultipleTrustee          *trustee
	MultipleTrusteeOperation int32
	TrusteeForm              int32
	TrusteeType              int32
	Name                     *uint16
}

// https://msdn.microsoft.com/en-us/library/windows/desktop/aa446627.aspx
type explicitAccess struct {
	AccessPermissions uint32
	AccessMode        int32
	Inheritance       uint32
	Trustee           trustee
}
