// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher

import (
	"fmt"
	"reflect"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
)

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

func XAttr(opname, doc string,
	prepare func(opVal string) (file.XAttr, error),
	eval func(opVal, val file.XAttr) bool) boolexpr.Operand {
	return xAttrOp{
		prep: prepare,
		eval: eval,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + doc,
			requires: reflect.TypeOf((*XAttrIfc)(nil)).Elem(),
		}}
}

type IDLookup func(string) (uint64, error)

func NewUser(name string, idl IDLookup) boolexpr.Operand {
	return XAttr(name, "matches the supplied user id or name",
		func(text string) (file.XAttr, error) {
			id, err := idl(text)
			if err != nil {
				return file.XAttr{}, err
			}
			return file.XAttr{UID: id}, nil
		},
		func(opVal, val file.XAttr) bool {
			fmt.Printf("%#v %#v\n", opVal, val)
			return opVal.UID == val.UID
		})
}

func NewGroup(name string, idl IDLookup) boolexpr.Operand {
	return XAttr(name, "matches the supplied group id or name",
		func(text string) (file.XAttr, error) {
			id, err := idl(text)
			if err != nil {
				return file.XAttr{}, err
			}
			return file.XAttr{GID: id}, nil
		},
		func(opVal, val file.XAttr) bool {
			return opVal.GID == val.GID
		})
}
