// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package lcs

import (
	"fmt"
	"strings"
)

// EditOp represents an edit operation.
type EditOp int

const (
	// EditOp values are as follows:
	Insert EditOp = iota
	Delete
	Identical
)

// Edit represents a single edit.
// For deletions, an edit specifies the index in the original (A) slice to be
// deleted.
// For insertions, an edit specifies the new value and the index in the original
// (A) slice that the new value is to be inserted at, but immediately after the
// existing value if that value was not deleted. Insertions also provide the
// index of the new value in the new (B) slice.
// A third operation is provided, that identifies identical values, ie.
// the members of the LCS and their position in the original and new slices.
// This allows for the LCS to retrieved from the SES.
//
// An EditScript that can be trivially 'replayed' to create the new slice
// from the original one.
//
//  var b []uint8
//   for _, action := range actions {
//     switch action.Op {
//     case Insert:
//       b = append(b, action.Val.(int64))
//     case Identical:
//       b = append(b, a[action.A])
//     }
//   }
type Edit struct {
	Op   EditOp
	A, B int
	Val  interface{}
}

// EditScript represents a series of Edits.
type EditScript []Edit

var opStr = map[EditOp]string{
	Insert:    "+",
	Delete:    "-",
	Identical: "=",
}

// String implements stringer.
func (es EditScript) String() string {
	out := strings.Builder{}
	for i, e := range es {
		out.WriteString(opStr[e.Op])
		if e.Op == Insert || e.Op == Identical {
			out.WriteString(" ")
			switch v := e.Val.(type) {
			case uint8:
				out.WriteByte(v)
			case rune:
				out.WriteRune(v)
			default:
				out.WriteString(fmt.Sprintf("%v", e.Val))
			}
			if e.Op == Identical {
				out.WriteString(fmt.Sprintf("@[%v == %v]", e.A, e.B))
			} else {
				out.WriteString(fmt.Sprintf("@[%v < %v]", e.A, e.B))
			}
		} else {
			out.WriteString(fmt.Sprintf(" @[%v]", e.A))
		}
		if i < len(es)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

func apply64(script EditScript, a []int64) []int64 {
	b := make([]int64, 0, len(script))
	for _, action := range script {
		switch action.Op {
		case Insert:
			b = append(b, action.Val.(int64))
		case Identical:
			b = append(b, a[action.A])
		}
	}
	return b
}

func apply32(script EditScript, a []int32) []int32 {
	b := make([]int32, 0, len(script))
	for _, action := range script {
		switch action.Op {
		case Insert:
			b = append(b, action.Val.(int32))
		case Identical:
			b = append(b, a[action.A])
		}
	}
	return b
}

func apply8(script EditScript, a []uint8) []uint8 {
	b := make([]uint8, 0, len(script))
	for _, action := range script {
		switch action.Op {
		case Insert:
			b = append(b, action.Val.(uint8))
		case Identical:
			b = append(b, a[action.A])
		}
	}
	return b
}

// Apply transforms the original slice to the new slice by
// applying the SES.
func (es EditScript) Apply(a interface{}) interface{} {
	if len(es) == 0 {
		return a
	}
	switch orig := a.(type) {
	case []int64:
		return apply64(es, orig)
	case []int32:
		return apply32(es, orig)
	case []uint8:
		return apply8(es, orig)
	}
	panic(fmt.Sprintf("unsupported type %T\n", a))
}

// Reverse returns a new edit script that is the inverse of the one supplied.
// That is, of the original script would transform A to B, then the results of
// this function will transform B to A.
func Reverse(es EditScript) EditScript {
	rev := make([]Edit, len(es))
	for i, e := range es {
		switch e.Op {
		case Identical:
			rev[i] = Edit{Op: Insert, A: e.B, B: e.A, Val: e.Val}
		case Delete:
			rev[i] = Edit{Op: Insert, A: e.B, B: e.A, Val: e.Val}
		case Insert:
			rev[i] = Edit{Op: Delete, A: e.B, B: e.A, Val: e.Val}
		}
	}
	return rev
}
