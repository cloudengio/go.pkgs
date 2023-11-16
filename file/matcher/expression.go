// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package matcher provides support for matching file names, types and modification
// times using regular expressions and boolean operators.
package matcher

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"time"
)

type itemType int

const (
	regex itemType = iota
	fileType
	newerThan
	andOp
	orOp
	leftBracket
	rightBracket
	subExpression
)

// Item represents an operator or operand in an expression. It is exposed
// to allow clients packages to create their own parsers.
type Item struct {
	typ       itemType
	text      string
	re        *regexp.Regexp
	filemode  os.FileMode
	newerThan time.Time
	sub       subItems
}

type subItems []Item

func (it itemType) String() string {
	switch it {
	case regex:
		return "regex"
	case andOp:
		return "&&"
	case orOp:
		return "||"
	case leftBracket:
		return "("
	case rightBracket:
		return ")"
	case fileType:
		return "filetype"
	case newerThan:
		return "newerthan"
	case subExpression:
		return "(...)"
	default:
		return fmt.Sprintf("unknown item type: %d", it)
	}
}

func (it Item) String() string {
	switch it.typ {
	case regex:
		return it.text
	case andOp:
		return "&&"
	case orOp:
		return "||"
	case leftBracket:
		return "("
	case rightBracket:
		return ")"
	case fileType:
		return `filetype("` + it.text + `")`
	case newerThan:
		return `newerthan("` + it.text + `")`
	case subExpression:
		return "(" + it.sub.String() + ")"
	default:
		return fmt.Sprintf("unknown item type: %v", it.typ)
	}
}

func (si subItems) String() string {
	res := ""
	for _, i := range si {
		res += i.String()
	}
	return res
}

func (it Item) isOperator() bool {
	return it.typ == andOp || it.typ == orOp
}

func (it Item) isOperand() bool {
	return it.typ == regex || it.typ == newerThan || it.typ == fileType || it.typ == subExpression
}

// OR returns an OR item.
func OR() Item {
	return Item{typ: orOp}
}

// And returns an AND item.
func AND() Item {
	return Item{typ: andOp}
}

// LeftBracket returns a left bracket item.
func LeftBracket() Item {
	return Item{typ: leftBracket}
}

// RightBracket returns a right bracket item.
func RightBracket() Item {
	return Item{typ: rightBracket}
}

// Regexp returns a regular expression item. It is not compiled until
// a matcher.T is created using New.
func Regexp(re string) Item {
	return Item{typ: regex, text: re}
}

func (it Item) evalFileType(v fs.FileMode) bool {
	if it.text == "f" {
		return v.IsRegular()
	}
	return v&it.filemode == it.filemode
}

// FileType returns a 'file type' item. It is not validated until a
// matcher.T is created using New. Supported file types are
// (as per the unix find command):
//   - f for regular files
//   - d for directories
//   - l for symbolic links
func FileType(typ string) Item {
	return Item{typ: fileType, text: typ}
}

func (it Item) prepare() (Item, error) {
	switch it.typ {
	case regex:
		re, err := regexp.Compile(it.text)
		if err != nil {
			return Item{}, err
		}
		it.re = re
		return it, nil
	case fileType:
		switch it.text {
		case "d":
			it.filemode = fs.ModeDir
		case "f":
			it.filemode = 0
		case "l":
			it.filemode = fs.ModeSymlink
		default:
			return Item{}, fmt.Errorf("invalid file type: %v, use one of d, f or l", it.text)
		}
		return it, nil
	case newerThan:
		for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
			if t, err := time.Parse(format, it.text); err == nil {
				it.newerThan = t
				return it, nil
			}
		}
		return Item{}, fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", it.text)
	}
	return it, nil
}

// NewerThan returns a 'newer than' item. It is not validated until a
// matcher.T is created using New.
func NewerThan(typ string) Item {
	return Item{typ: newerThan, text: typ}
}

// T represents a boolean expression of regular expressions,
// file type and mod time comparisons. It is evaluated against a single
// input value.
type T struct {
	items        []Item
	hasFileType  bool
	hasNewerThan bool
}

// NeedsFileMode returns true if the expression contains a file type
// comparison.
func (m T) NeedsFileMode() bool {
	return m.hasFileType
}

// NeedsModTime returns true if the expression contains a mod time
// comparison.
func (m T) NeedsModTime() bool {
	return m.hasNewerThan
}

func newExpression(input <-chan Item) ([]Item, error) {
	expr := []Item{}
	for cur := range input {
		switch cur.typ {
		case regex, fileType, newerThan:
			cur, err := cur.prepare()
			if err != nil {
				return nil, err
			}
			expr = append(expr, cur)
		case andOp, orOp:
			if len(expr) == 0 {
				return nil, fmt.Errorf("missing left operand for %v", cur.typ)
			}
			if !expr[len(expr)-1].isOperand() {
				return nil, fmt.Errorf("missing operand for %v", cur.typ)
			}
			expr = append(expr, cur)
		case leftBracket:
			if len(expr) > 0 && !expr[len(expr)-1].isOperator() {
				return nil, fmt.Errorf("missing operator for %v", cur.typ)
			}
			sub, err := newExpression(input)
			if err != nil {
				return nil, err
			}
			expr = append(expr, Item{typ: subExpression, sub: sub})
		case rightBracket:
			if len(expr) == 0 {
				return nil, fmt.Errorf("missing left operand for %v", cur.typ)
			}
			if !expr[len(expr)-1].isOperand() {
				return nil, fmt.Errorf("missing operand for %v", cur.typ)
			}
			return expr, nil
		}
	}
	return expr, nil
}

func itemChan(items []Item) <-chan Item {
	itemCh := make(chan Item, len(items))
	for _, it := range items {
		itemCh <- it
	}
	close(itemCh)
	return itemCh
}

// New returns a new matcher.T built from the supplied items.
func New(items ...Item) (T, error) {
	if len(items) == 0 {
		return T{}, fmt.Errorf("empty expression")
	}
	// check for balanced ('s here since it's easy to do so.
	balanced := 0
	for _, it := range items {
		if it.typ == leftBracket {
			balanced++
		}
		if it.typ == rightBracket {
			if balanced == 0 {
				return T{}, fmt.Errorf("unbalanced brackets")
			}
			balanced--
		}
	}
	if balanced != 0 {
		return T{}, fmt.Errorf("unbalanced brackets")
	}
	tree, err := newExpression(itemChan(items))
	if err != nil {
		return T{}, err
	}
	if len(tree)%2 == 0 {
		return T{}, fmt.Errorf("incomplete expression: %v", items)
	}
	m := T{items: tree}
	needs(&m, m.items)
	return m, nil
}

func needs(m *T, items []Item) {
	for _, it := range items {
		switch it.typ {
		case fileType:
			m.hasFileType = true
		case newerThan:
			m.hasNewerThan = true
		case subExpression:
			needs(m, it.sub)
		}
	}
}

func (m T) String() string {
	var out strings.Builder
	for _, it := range m.items {
		if it.typ == subExpression {
			se := T{items: it.sub}
			out.WriteString("(" + se.String() + ") ")
			continue
		}
		out.WriteString(it.String() + " ")
	}
	return strings.TrimSpace(out.String())
}

// Value represents a value to be evaluated against an expression.
type Value interface {
	Name() string
	Mode() fs.FileMode
	ModTime() time.Time
}

// Eval evaluates the expression against the supplied value return
// the value of the expression or an error. If expressions is empty
// it will always return false.
func (m T) Eval(v Value) bool {
	if len(m.items) == 0 {
		return false
	}
	return eval(itemChan(m.items), v)
}

func eval(exprs <-chan Item, v Value) bool {
	values := []bool{}
	operators := []itemType{}
	for cur := range exprs {
		switch cur.typ {
		case regex:
			values = append(values, cur.re.MatchString(v.Name()))
		case fileType:
			values = append(values, cur.evalFileType(v.Mode()))
		case newerThan:
			values = append(values, v.ModTime().After(cur.newerThan))
		case orOp:
			if values[len(values)-1] {
				// early return on true || ....
				return true
			}
			operators = append(operators, cur.typ)
		case andOp:
			operators = append(operators, cur.typ)
		case subExpression:
			values = append(values, eval(itemChan(cur.sub), v))
		}
		// left to right evaluation
		if len(values) == 2 && len(operators) == 1 {
			switch operators[0] {
			case andOp:
				values = []bool{values[0] && values[1]}
			case orOp:
				values = []bool{values[0] || values[1]}
			}
			operators = []itemType{}
		}
	}
	return values[0]
}