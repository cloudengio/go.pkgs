package lcs

import (
	"fmt"
	"strings"
)

// EditOp represents an edit operation, either insertion or deletion.
type EditOp int

// Values for EditOp are Insert or Delete.
const (
	Insert EditOp = iota
	Delete
	Identical
)

// Edit represents a single edit.
// For deletions, an edit specifies the index in the original (A)
// slice to be deleted.
// For insertions, an edit specifies the new value and the index in
// the original (A) slice that the new value is to be inserted at,
// but immediately after the existing value.
// Insertions also provide the index of the new value in the new
// (B) slice.
// A third operation is provided, that identifies identical values, ie.
// the members of the LCS and their position in the original and new slices.
// This allows for the LCS to retrieved from the SES.
//
// An EditScript that can be trivially 'replayed' to create the new slice
// from the original one.
//
//   var b []uint8
//    for _, action := range actions {
//		switch action.Op {
//		case Insert:
//			b = append(b, action.Val.(int64))
//		case Identical:
//			b = append(b, a[action.A])
//		}
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

func perPosition(pos int, script EditScript) ([]Edit, EditScript) {
	if len(script) == 0 {
		return nil, nil // []Edit{{Op: Identical, A: pos}}, script
	}
	ops := []Edit{}
	used := 0
	replacing, first := false, true

	// need to handle the following cases:
	// insert... : copy over value and then insert multiple items after it.
	// delete, insert: ie. replace.
	// delete, insert...: i.e. delete the original and insert multiple items.
	for _, op := range script {
		if op.A != pos {
			break
		}
		used++
		if op.Op == Delete {
			ops = append(ops, op)
			replacing = true
			continue
		}
		if !replacing && first {
			//ops = append(ops, Edit{Op: Identical, A: pos})
		}
		ops = append(ops, op)
		replacing, first = false, false
	}
	//	if len(ops) == 0 {
	//		ops = []Edit{{Op: Identical, A: pos}}
	//	}
	return ops, script[used:]
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

// Apply transforms the original slice to the new value by
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
