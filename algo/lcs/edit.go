package lcs

import (
	"fmt"
	"sort"
	"strings"
)

// EditOp represents an edit operation, either insertion or deletion.
type EditOp int

// Values for EditOp are Insert or Delete.
const (
	Insert EditOp = iota
	Delete
)

// Edit represents a single edit, the position in the original string
// that it should take place at and the value for an insertion.
type Edit struct {
	Op   EditOp
	A, B int
	Val  interface{}
}

// EditScript represents a series of Edits.
type EditScript []Edit

var opStr = map[EditOp]string{
	Insert: "+",
	Delete: "-",
}

// String implements stringer.
func (es EditScript) String() string {
	out := strings.Builder{}
	for i, e := range es {
		out.WriteString(opStr[e.Op])
		if e.Op == Insert {
			out.WriteString(" ")
			switch v := e.Val.(type) {
			case uint8:
				out.WriteByte(v)
			case rune:
				out.WriteRune(v)
			default:
				out.WriteString(fmt.Sprintf("%v", e.Val))
			}
		}
		out.WriteString(fmt.Sprintf(" @[%v]", e.A))
		if i < len(es)-1 {
			out.WriteString(", ")
		}
	}
	return out.String()
}

type editOperation int

const (
	//	doneAction
	copyAction editOperation = iota
	skipAction
	insertAction
)

type action struct {
	op     editOperation
	pos    int
	insert interface{}
}

func perPosition(pos int, script EditScript) ([]action, EditScript) {
	if len(script) == 0 {
		return []action{{op: copyAction, pos: pos}}, script
	}
	ops := []action{}
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
			ops = append(ops, action{op: skipAction, pos: pos})
			replacing = true
			continue
		}
		if !replacing && first {
			ops = append(ops, action{op: copyAction, pos: pos})
		}
		ops = append(ops, action{op: insertAction, pos: pos, insert: op.Val})
		replacing, first = false, false
	}
	if len(ops) == 0 {
		ops = []action{{op: copyAction, pos: pos}}
	}
	return ops, script[used:]
}

func interpret(n int, script EditScript) []action {
	var actions []action
	if n == 0 {
		for _, edit := range script {
			actions = append(actions, action{op: insertAction, pos: 0, insert: edit.Val})
		}
		return actions
	}
	for i := 0; i < n; i++ {
		var edits []action
		edits, script = perPosition(i, script)
		actions = append(actions, edits...)
	}
	return actions
}

func apply64(script EditScript, a []int64) []int64 {
	actions := interpret(len(a), script)
	b := make([]int64, 0, len(actions))
	for _, action := range actions {
		switch action.op {
		case skipAction:
		case insertAction:
			b = append(b, action.insert.(int64))
		case copyAction:
			b = append(b, a[action.pos])
		}
	}
	return b
}

func apply32(script EditScript, a []int32) []int32 {
	actions := interpret(len(a), script)
	b := make([]int32, 0, len(actions))
	for _, action := range actions {
		switch action.op {
		case skipAction:
		case insertAction:
			b = append(b, action.insert.(int32))
		case copyAction:
			b = append(b, a[action.pos])
		}
	}
	return b
}

func apply8(script EditScript, a []uint8) []uint8 {
	actions := interpret(len(a), script)
	b := make([]uint8, 0, len(actions))
	for _, action := range actions {
		switch action.op {
		case skipAction:
		case insertAction:
			b = append(b, action.insert.(uint8))
		case copyAction:
			b = append(b, a[action.pos])
		}
	}
	return b
}

func (es EditScript) Apply(a interface{}) interface{} {
	if len(es) == 0 {
		return a
	}
	// sort by position and then then deletes first.
	sort.Slice(es, func(i, j int) bool {
		if es[i].A == es[j].A {
			// delete's first
			if es[i].Op == Delete {
				return true
			}
			return false
		}
		return es[i].A < es[j].A
	})
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
