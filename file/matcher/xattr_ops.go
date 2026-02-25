// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher

import (
	"os/user"
	"reflect"
	"strconv"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/os/userid"
)

// XAttrIfc must be implemented by any values that are used with the
// XAttr operand.
type XAttrIfc interface {
	XAttr() file.XAttr
}

type xAttrOp struct {
	commonOperand
	prep  func(text string) (file.XAttr, error)
	eval  func(opVal, val file.XAttr) bool
	opVal file.XAttr
	text  string
}

func (op xAttrOp) Prepare() (boolexpr.Operand, error) {
	xattr, err := op.prep(op.text)
	if err != nil {
		return op, err
	}
	op.opVal = xattr
	return op, nil
}

func (op xAttrOp) Eval(v any) bool {
	if nt, ok := v.(XAttrIfc); ok {
		return op.eval(op.opVal, nt.XAttr())
	}
	return false
}

func (op xAttrOp) String() string {
	return op.name + "=" + op.text
}

type XAttrParser func(text string) (file.XAttr, error)

// XAttr returns an operand that compares an xattr value with
// the xattr value of the value being evaluated.
func XAttr(opname, value, doc string,
	prepare XAttrParser,
	eval func(opVal, val file.XAttr) bool) boolexpr.Operand {
	return xAttrOp{
		text: value,
		prep: prepare,
		eval: eval,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + doc,
			requires: reflect.TypeFor[XAttrIfc](),
		}}
}

// NewUser returns an operand that compares the user id of the value
// being evaluated with the supplied user id or name. The supplied
// IDLookup is used to convert the supplied text into a user id.
// The value being evaluated must implement the XAttrIfc interface.
func NewUser(name, value string, parser XAttrParser) boolexpr.Operand {
	return XAttr(name, value, "=<uid|username> matches the supplied user id or name",
		parser,
		func(opVal, val file.XAttr) bool {
			return opVal.CompareUser(val)
		})
}

// NewGroup returns an operand that compares the group id of the value
// being evaluated with the supplied group id or name. The supplied
// IDLookup is used to convert the supplied text into a group id.
// The value being evaluated must implement the XAttrIfc interface.
func NewGroup(name, value string, parser XAttrParser) boolexpr.Operand {
	return XAttr(name, value, "=<gid/groupname> matches the supplied group id or name",
		parser,
		func(opVal, val file.XAttr) bool {
			return opVal.CompareGroup(val)
		})
}

// ParseUsernameOrID returns a file.XAttr that represents the supplied
// name or ID.
func ParseUsernameOrID(nameOrID string, lookup func(name string) (userid.IDInfo, error)) (file.XAttr, error) {
	info, err := lookup(nameOrID)
	if err != nil {
		// On Windows, the owner of a file may be a group.
		if grp, err := user.LookupGroupId(nameOrID); err == nil {
			return file.XAttr{UID: -1, User: grp.Gid}, nil
		}
		return file.XAttr{UID: -1, User: nameOrID}, err
	}
	if id, err := strconv.ParseInt(info.UID, 10, 32); err == nil {
		return file.XAttr{UID: id, User: info.Username}, nil
	}
	return file.XAttr{UID: -1, User: info.Username}, nil
}

// ParseGroupnameOrID returns a file.XAttr that represents the supplied
// name or ID.
func ParseGroupnameOrID(nameOrID string, lookup func(name string) (user.Group, error)) (file.XAttr, error) {
	info, err := lookup(nameOrID)
	if err != nil {
		return file.XAttr{GID: -1, Group: nameOrID}, err
	}
	if id, err := strconv.ParseInt(info.Gid, 10, 32); err == nil {
		return file.XAttr{GID: id, Group: info.Name}, nil
	}
	// On Windows, use SID as the name.
	return file.XAttr{GID: -1, Group: info.Gid}, nil
}
