package lcs

import "strings"

type EditOp int

const (
	Same EditOp = iota
	Add
	Delete
)

type Edit struct {
	Op  EditOp
	Pos int
}

type EditScript []Edit

func (es EditScript) String() string {
	out := strings.Builder{}
	for i, e := range es {
		switch e.Op {
		case Same:
			out.WriteRune('=')
		case Add:
			out.WriteRune('+')
		case Delete:
			out.WriteRune('-')
		}
		out.WriteRune('x') //e.Token)
		if i < len(es)-1 {
			out.WriteRune(' ')
		}
	}
	return out.String()
}
