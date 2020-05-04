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
	Op  EditOp
	A   int
	Val interface{}
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
	doneAction editOperation = iota
	copyAction
	skipAction
	insertAction
)

func (ea editAction) String() string {
	switch ea {
	case doneAction:
		return "::"
	case copyAction:
		return "="
	case skipAction:
		return "-"
	case insertAction:
		return "+"
	}
	panic("unreachable")
}

// editState implements a per edit-position state machine. A state machine is needed
// to handle the fact that edit strings contain deletion+insertion pairs
// for replacement, and runs of insertions. The runs of insertions require
// copying over the original value followed by one or more insertions.
type editState struct {
	inserting, replacing bool
}

func (es *editState) nextOp(pos int, script EditScript) (editAction, interface{}, int, bool) {
	if len(script) == 0 {
		return copyAction, nil, 0, true
		//		return doneAction, nil, 0, true
	}
	if script[0].A != pos {
		es.inserting, es.replacing = false, false
		return copyAction, nil, 0, true
	}
	final := len(script) == 1 || (len(script) > 1 && script[1].A != pos)
	if script[0].Op == Delete {
		if !final {
			es.inserting = true
		}
		return skipAction, nil, 1, final
	}
	if es.inserting || es.replacing {
		if final {
			es.inserting, es.replacing = false, false
		}
		return insertAction, script[0].Val, 1, final
	}
	es.inserting = true
	return copyAction, nil, 0, false
}

func perPosition(pos int, script EditScript)

type action struct {
	skip   bool
	pos    int
	insert interface{}
}

func interpret(n int, script EditScript) []action {
	var actions []action
	if n == 0 {
		for _, edit := range script {
			actions = append(actions, action{pos: 0, insert: edit.Val})
		}
		return actions
	}
	for i := 0; i < n; i++ {
		es := &editState{}
		for {
			editAction, val, consumed, final := es.nextOp(i, script)
			switch editAction {
			case copyAction:
				actions = append(actions, action{pos: i})
			case insertAction:
				actions = append(actions, action{pos: i, insert: val})
			case skipAction:
				actions = append(actions, action{pos: i, skip: true})
			}
			script = script[consumed:]
			if final {
				break
			}
		}
	}
	return actions
}

func apply32(script EditScript, a []int32) []int32 {
	actions := interpret(len(a), script)
	b := make([]int32, 0, len(actions))
	for _, action := range actions {
		if action.skip {
			continue
		}
		if v := action.insert; v != nil {
			b = append(b, v.(int32))
			continue
		}
		b = append(b, a[action.pos])
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
	case []int32:
		return apply32(es, orig)
	case []uint8:
		//return apply8(es, orig)
	}
	panic(fmt.Sprintf("unsupported type %T\n", a))
}
