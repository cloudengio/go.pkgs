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
	sub expression
	op  Operand
}

type expression []Item

func (e expression) String() string {
	var res strings.Builder
	for _, i := range e {
		res.WriteString(i.String())
	}
	return res.String()
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

func (e expression) newOperand(op Operand) (expression, error) {
	op, err := op.Prepare()
	if err != nil {
		return nil, err
	}
	return append(e, NewOperandItem(op)), nil
}

func (e expression) newBinaryOp(cur Item) (expression, error) {
	if len(e) == 0 {
		return nil, fmt.Errorf("missing left operand for %v", cur.typ)
	}
	if !e[len(e)-1].isOperand() {
		return nil, fmt.Errorf("missing operand preceding %v", cur.typ)
	}
	return append(e, cur), nil
}

func (e expression) newUnaryOp(cur Item) (expression, error) {
	if len(e) > 0 {
		l := e[len(e)-1]
		if !l.isOperator() {
			return nil, fmt.Errorf("misplaced negation after %v", l.typ)
		}
	}
	return append(e, cur), nil
}

func (e expression) newSubExpression(input <-chan Item, cur Item) (expression, error) {
	if len(e) > 0 {
		l := e[len(e)-1]
		if !l.isOperator() && l.typ != notOp {
			return nil, fmt.Errorf("missing operator preceding %v", cur.typ)
		}
	}
	sub, err := newExpression(input)
	if err != nil {
		return nil, err
	}
	return append(e, Item{typ: subExpression, sub: sub}), nil
}

func (e expression) endSubExpression(cur Item) (expression, error) {
	if len(e) == 0 {
		return nil, fmt.Errorf("missing left operand for %v", cur.typ)
	}
	if !e[len(e)-1].isOperand() {
		return nil, fmt.Errorf("missing operand preceding %v", cur.typ)
	}
	return e, nil
}

func newExpression(input <-chan Item) (expr expression, err error) {
	for cur := range input {
		switch cur.typ {
		case operand:
			expr, err = expr.newOperand(cur.op)
		case andOp, orOp:
			expr, err = expr.newBinaryOp(cur)
		case notOp:
			expr, err = expr.newUnaryOp(cur)
		case leftBracket:
			expr, err = expr.newSubExpression(input, cur)
		case rightBracket:
			return expr.endSubExpression(cur)
		}
		if err != nil {
			return nil, err
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
	if incompleteExpression(tree) {
		return T{}, fmt.Errorf("incomplete expression: %v", items)
	}
	return T{items: tree}, nil
}

func incompleteExpression(items []Item) bool {
	switch len(items) {
	case 0:
		return false
	case 1:
		switch items[0].typ {
		case operand:
			return false
		case subExpression:
			return incompleteExpression(items[0].sub)
		}
		return true
	case 2:
		return items[0].typ != notOp
	case 3:
		return false
	}
	return false
}

func (m T) String() string {
	var out strings.Builder
	for _, it := range m.items {
		if it.typ == subExpression {
			se := T{items: it.sub}
			out.WriteString("(" + se.String() + ") ")
			continue
		}
		out.WriteString(it.String())
		if it.typ != notOp {
			out.WriteRune(' ')
		}
	}
	return strings.TrimSpace(out.String())
}

// Eval evaluates the matcher against the supplied value. An empty, default
// matcher will always return false.
func (m T) Eval(v any) bool {
	if len(m.items) == 0 {
		return false
	}
	ev := evaluator{}
	return ev.run(itemChan(m.items), v)
}

type evaluator struct {
	values    []bool
	operators []itemType
}

func (e *evaluator) pushVal(v bool) {
	e.values = append(e.values, v)
}

func (e *evaluator) pushOp(it itemType) {
	e.operators = append(e.operators, it)
}

func (e *evaluator) eval() bool {
	// left to right evaluation
	if len(e.values) == 1 && len(e.operators) == 1 && e.operators[0] == notOp {
		// handle negation of single value
		e.operators = e.operators[1:]
		e.values[len(e.values)-1] = !e.values[len(e.values)-1]
	}

	if len(e.values) == 2 && len(e.operators) >= 1 {
		if len(e.operators) >= 2 && e.operators[len(e.operators)-1] == notOp {
			// handle negation in the right hand side of an expression.
			e.values[1] = !e.values[1]
			e.operators = e.operators[:len(e.operators)-1]
		}

		switch e.operators[0] {
		case andOp:
			e.values = []bool{e.values[0] && e.values[1]}
		case orOp:
			e.values = []bool{e.values[0] || e.values[1]}
			if e.values[0] {
				// short circuit evaluation of OR
				return true
			}
		}
	}
	return false
}

func (e *evaluator) run(exprs <-chan Item, v any) bool {
	for cur := range exprs {
		switch cur.typ {
		case operand:
			e.pushVal(cur.op.Eval(v))
		case orOp, andOp, notOp:
			e.pushOp(cur.typ)
		case subExpression:
			sub := &evaluator{}
			e.pushVal(sub.run(itemChan(cur.sub), v))
		}
		if e.eval() {
			return true
		}
	}
	if len(e.values) > 1 {
		panic(fmt.Sprintf("invalid expression: %v", e.values))
	}
	return e.values[0]
}
