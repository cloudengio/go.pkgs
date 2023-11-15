// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

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
		return "filetype(" + it.text + ")"
	case newerThan:
		return "newerthan(" + it.text + ")"
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

func OR() Item {
	return Item{typ: orOp}
}

func AND() Item {
	return Item{typ: andOp}
}

func LeftBracket() Item {
	return Item{typ: leftBracket}
}

func RightBracket() Item {
	return Item{typ: rightBracket}
}

func Regexp(re string) Item {
	return Item{typ: regex, text: re}
}

func (it Item) compileRE() (Item, error) {
	re, err := regexp.Compile(it.text)
	if err != nil {
		return Item{}, err
	}
	it.re = re
	return it, nil
}

func (it Item) parseFileType() (Item, error) {
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
}

func FileType(typ string) Item {
	return Item{typ: fileType, text: typ}
}

func (it Item) parseNewerThan() (Item, error) {
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		if t, err := time.Parse(format, it.text); err == nil {
			it.newerThan = t
			return it, nil
		}
	}
	return Item{}, fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", it.text)
}

func NewerThan(typ string) Item {
	return Item{typ: newerThan, text: typ}
}

type Expression struct {
	items []Item
}

func newExpression(input <-chan Item) ([]Item, error) {
	expr := []Item{}
	for cur := range input {
		switch cur.typ {
		case regex:
			cur, err := cur.compileRE()
			if err != nil {
				return nil, err
			}
			expr = append(expr, cur)
		case fileType:
			cur, err := cur.parseFileType()
			if err != nil {
				return nil, err
			}
			expr = append(expr, cur)
		case newerThan:
			cur, err := cur.parseNewerThan()
			if err != nil {
				return nil, err
			}
			expr = append(expr, cur)
		case andOp, orOp:
			if len(expr) == 0 {
				return nil, fmt.Errorf("missing operand for %v", cur.typ)
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
				return nil, fmt.Errorf("missing operand for %v", cur.typ)
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

func NewExpression(items ...Item) (Expression, error) {
	tree, err := newExpression(itemChan(items))
	if err != nil {
		return Expression{}, err
	}
	e := Expression{items: tree}
	if err := e.validate(tree); err != nil {
		return Expression{}, err
	}
	return e, nil
}

func (e Expression) isWellFormed(items []Item) error {
	if len(items) == 1 {
		if !items[0].isOperand() {
			return fmt.Errorf("single item expression must be an operand: %v", items[0])
		}
		return nil
	}
	if len(items)%2 == 0 {
		return fmt.Errorf("incomplete expression: %v", items)
	}
	// should be alternating operands and operators.
	for i, it := range items {
		if i%2 == 0 && !it.isOperand() {
			return fmt.Errorf("expected operand at position %v: %v", i, items)
		}
		if i%2 == 1 && !it.isOperator() {
			return fmt.Errorf("expected operator at position %v: %v", i, items)
		}
	}
	return nil
}

func (e Expression) validate(items []Item) error {
	if len(items) == 0 {
		return nil
	}
	for i, it := range items {
		// There should be no bracket's left.
		if it.typ == leftBracket {
			return fmt.Errorf("extra left bracket at position %v", i)
		}
		if it.typ == rightBracket {
			return fmt.Errorf("extra right bracket at position %v", i)
		}
	}
	if err := e.isWellFormed(items); err != nil {
		return err
	}
	for _, it := range items {
		if it.typ == subExpression {
			if err := e.validate(it.sub); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ex Expression) String() string {
	var out strings.Builder
	for _, it := range ex.items {
		if it.typ == subExpression {
			se := Expression{items: it.sub}
			out.WriteString("(" + se.String() + ") ")
			continue
		}
		out.WriteString(it.String() + " ")
	}
	return strings.TrimSpace(out.String())
}

type Evaluable interface {
	Name() string
	Mode() fs.FileMode
	ModTime() time.Time
}

func (ex Expression) Eval(e Evaluable) (bool, error) {
	return eval(itemChan(ex.items), e)
}

type evalState struct {
	value    bool
	operator itemType // and, or.
}

func eval(exprs <-chan Item, e Evaluable) (bool, error) {
	values := []bool{}
	operators := []itemType{}
	for cur := range exprs {
		switch cur.typ {
		case regex:
			values = append(values, cur.re.MatchString(e.Name()))
		case fileType:
			fmt.Printf("%b %b %b\n", e.Mode(), cur.filemode, e.Mode()&cur.filemode)
			val := e.Mode()&cur.filemode == cur.filemode
			if cur.text == "f" {
				val = e.Mode().IsRegular()
			}
			values = append(values, val)
		case newerThan:
			values = append(values, e.ModTime().After(cur.newerThan))
		case orOp:
			if len(values) == 0 {
				return false, fmt.Errorf("missing operand for %v", cur.typ)
			}
			if values[len(values)-1] == true {
				// early return on true || ....
				return true, nil
			}
			operators = append(operators, cur.typ)
		case andOp:
			operators = append(operators, cur.typ)
		case leftBracket:
			subValue, err := eval(exprs, e)
			if err != nil {
				return false, err
			}
			values = append(values, subValue)
		case rightBracket:
			break
		}
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
	if len(values) != 1 {
		return false, fmt.Errorf("invalid evaluation state, too many values: %v", values)
	}
	return values[0], nil
}
