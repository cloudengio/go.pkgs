// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package lcs

import (
	"fmt"
	"io"
	"strings"
)

// EditOp represents an edit operation.
type EditOp int

// Values for EditOP.
const (
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
//	var b []uint8
//	 for _, action := range actions {
//	   switch action.Op {
//	   case Insert:
//	     b = append(b, action.Val.(int64))
//	   case Identical:
//	     b = append(b, a[action.A])
//	   }
//	 }
type Edit[T comparable] struct {
	Op   EditOp
	A, B int
	Val  T
}

// EditScript represents a series of Edits.
type EditScript[T comparable] []Edit[T]

var opStr = map[EditOp]string{
	Insert:    "+",
	Delete:    "-",
	Identical: "=",
}

// String implements stringer.
func (es *EditScript[T]) String() string {
	out := strings.Builder{}
	for i, e := range *es {
		out.WriteString(opStr[e.Op])
		if e.Op == Insert || e.Op == Identical {
			out.WriteString(" ")
			fmt.Fprintf(&out, "%v", e.Val)
			if e.Op == Identical {
				fmt.Fprintf(&out, "@[%v == %v]", e.A, e.B)
			} else {
				fmt.Fprintf(&out, "@[%v < %v]", e.A, e.B)
			}
		} else {
			fmt.Fprintf(&out, " @[%v]", e.A)
		}
		if i < len(*es)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

// Apply transforms the original slice to the new slice by
// applying the SES.
func (es *EditScript[T]) Apply(a []T) []T {
	if len(*es) == 0 {
		return a
	}
	b := make([]T, 0, len(*es))
	for _, action := range *es {
		switch action.Op {
		case Insert:
			b = append(b, action.Val)
		case Identical:
			b = append(b, a[action.A])
		}
	}
	return b
}

// Reverse returns a new edit script that is the inverse of the one supplied.
// That is, of the original script would transform A to B, then the results of
// this function will transform B to A.
func (es *EditScript[T]) Reverse() *EditScript[T] {
	var rev EditScript[T] = make([]Edit[T], len(*es))
	for i, e := range *es {
		switch e.Op {
		case Identical:
			rev[i] = Edit[T]{Op: Insert, A: e.B, B: e.A, Val: e.Val}
		case Delete:
			rev[i] = Edit[T]{Op: Insert, A: e.B, B: e.A, Val: e.Val}
		case Insert:
			rev[i] = Edit[T]{Op: Delete, A: e.B, B: e.A, Val: e.Val}
		}
	}
	return &rev
}

func verticalFormatFor(a any) string {
	switch a.(type) {
	case []int8, []uint8, []rune:
		return "%3c"
	case []int16, []uint16, []uint32, []int64, []uint64:
		return "% 20d"
	case []float32, []float64:
		return "% 20.3e"
	case []complex64, complex128:
		return "% 30i"
	case []string:
		return "%s"
	default:
		return "%v"
	}
}

// FormatVertical prints a representation of the edit script with one
// item per line, eg:
//   - 6864772235558415538
//     -8997218578518345818
//   - -6615550055289275125
//   - -7192184552745107772
//     5717881983045765875
func (es *EditScript[T]) FormatVertical(out io.Writer, a []T) {
	format := verticalFormatFor(a)
	for _, op := range *es {
		switch op.Op {
		case Identical:
			f := fmt.Sprintf(format, a[op.A])
			fmt.Fprintf(out, "  %s\n", f)
		case Delete:
			f := fmt.Sprintf(format, a[op.A])
			fmt.Fprintf(out, "- %s\n", f)
		case Insert:
			f := fmt.Sprintf(format, op.Val)
			fmt.Fprintf(out, "+ %s\n", f)
		}
	}
}

func horizontalFormatFor(a any) string {
	switch a.(type) {
	case []int8, []uint8, []rune:
		return "%c"
	default:
		return "%v"
	}
}

// FormatVertical prints a representation of the edit script across
// three lines, with the top line showing the result of applying the
// edit, the middle line the operations applied and the bottom line
// any items deleted, eg:
//
//	 CB AB AC
//	-+|-||-|+
//	A  C  B
func (es *EditScript[T]) FormatHorizontal(out io.Writer, a []T) {
	format := horizontalFormatFor(a)
	displaySizes := []int{}
	for _, op := range *es {
		var f string
		switch op.Op {
		case Identical:
			f = fmt.Sprintf(format, a[op.A])
			_, _ = out.Write([]byte(f))
		case Delete:
			f = fmt.Sprintf(format, a[op.A])
			_, _ = out.Write([]byte(strings.Repeat(" ", len(f))))
		case Insert:
			f = fmt.Sprintf(format, op.Val)
			_, _ = out.Write([]byte(f))
		}
		displaySizes = append(displaySizes, len(f))
	}
	_, _ = out.Write([]byte{'\n'})

	pad := func(o string, i int) {
		totalPadding := displaySizes[i] - len(o)
		prePad := totalPadding / 2
		postPad := totalPadding - prePad
		_, _ = out.Write([]byte(strings.Repeat(" ", prePad)))
		_, _ = out.Write([]byte(o))
		_, _ = out.Write([]byte(strings.Repeat(" ", postPad)))
	}

	for i, op := range *es {
		switch op.Op {
		case Identical:
			pad("|", i)
		case Delete:
			pad("-", i)
		case Insert:
			pad("+", i)
		}
	}
	_, _ = out.Write([]byte{'\n'})
	for i, op := range *es {
		switch op.Op {
		case Delete:
			f := fmt.Sprintf(format, a[op.A])
			_, _ = out.Write([]byte(f))
		default:
			pad("", i)
		}
	}
	_, _ = out.Write([]byte{'\n'})
}
