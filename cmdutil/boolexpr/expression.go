// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package boolexpr provides a boolean expression evaluator and parser.
// The supported operators are &&, || and ! (negation), and grouping via ().
// The set of operands is defined by clients of the package by
// implementing the Operand interface. Operands represent simple predicates
// against which the value supplied to the expression is evaluated, as such,
// they implicitly contain a value of their own that is assigned when the
// operand is instantiated. For example, a simple string comparison operand
// would be represented as "name='foo' || name='bar'" which evaluated to true
// if the expression is evaluated for "foo" or "bar", but not otherwise.
package boolexpr

import (
	"fmt"
	"reflect"
	"strings"
)

type itemType int

const (
	orOp itemType = iota
	andOp
	notOp
	leftBracket
	rightBracket
	subExpression
	operand
)

func (it itemType) String() string {
	switch it {
	case andOp:
		return "&&"
	case orOp:
		return "||"
	case leftBracket:
		return "("
	case rightBracket:
		return ")"
	case notOp:
		return "!"
	case operand:
		return "operand"
	case subExpression:
		return "(...)"
	default:
		return fmt.Sprintf("unknown item type: %d", it)
	}
}

// Item represents an operator or operand in an expression. It is exposed
// to allow clients packages to create their own parsers.
type Item struct {
	typ itemType
	sub subItems
	op  Operand
}

type subItems []Item

func (si subItems) String() string {
	res := ""
	for _, i := range si {
		res += i.String()
	}
	return res
}

func (it Item) String() string {
	switch it.typ {
	case andOp:
		return "&&"
	case orOp:
		return "||"
	case leftBracket:
		return "("
	case rightBracket:
		return ")"
	case notOp:
		return "!"
	case subExpression:
		return "(" + it.sub.String() + ")"
	case operand:
		return it.op.String()
	default:
		return fmt.Sprintf("unknown item type: %v", it.typ)
	}
}

func (it Item) isOperator() bool {
	return it.typ == andOp || it.typ == orOp
}

func (it Item) isOperand() bool {
	return it.typ == operand || it.typ == subExpression
}

// OR returns an OR item.
func OR() Item {
	return Item{typ: orOp}
}

// And returns an AND item.
func AND() Item {
	return Item{typ: andOp}
}

// NOT returns a NOT item.
func NOT() Item {
	return Item{typ: notOp}
}

// LeftBracket returns a left bracket item.
func LeftBracket() Item {
	return Item{typ: leftBracket}
}

// RightBracket returns a right bracket item.
func RightBracket() Item {
	return Item{typ: rightBracket}
}

// T represents a boolean expression of regular expressions,
// file type and mod time comparisons. It is evaluated against a single
// input value.
type T struct {
	items []Item
}

// HasOperand returns true if the matcher's expression contains an instance
// of the specified operand.
func (m T) Needs(typ any) bool {
	return needs(reflect.TypeOf(typ), m.items)
}

func needs(want reflect.Type, items []Item) bool {
	for _, it := range items {
		switch it.typ {
		case operand:
			if it.op.Needs(want) {
				return true
			}
		case subExpression:
			if needs(want, it.sub) {
				return true
			}
		}
	}
	return false
}

func newExpression(input <-chan Item) ([]Item, error) {
	expr := []Item{}
	for cur := range input {
		switch cur.typ {
		case operand:
			op, err := cur.op.Prepare()
			if err != nil {
				return nil, err
			}
			expr = append(expr, NewOperandItem(op))
		case andOp, orOp:
			if len(expr) == 0 {
				return nil, fmt.Errorf("missing left operand for %v", cur.typ)
			}
			if !expr[len(expr)-1].isOperand() {
				return nil, fmt.Errorf("missing operand for %v", cur.typ)
			}
			expr = append(expr, cur)
		case notOp:
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
		return T{}, nil
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
	return T{items: tree}, nil
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

// Eval evaluates the matcher against the supplied value. An empty, default
// matcher will always return false.
func (m T) Eval(v any) bool {
	if len(m.items) == 0 {
		return false
	}
	return eval(itemChan(m.items), v)
}

func eval(exprs <-chan Item, v any) bool {
	values := []bool{}
	operators := []itemType{}
	for cur := range exprs {
		switch cur.typ {
		case operand:
			values = append(values, cur.op.Eval(v))
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
