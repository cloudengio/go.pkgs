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
	invalidOp itemType = iota
	orOp
	andOp
	notOp
	leftBracket
	rightBracket
	operand
)

func (it itemType) String() string {
	switch it {
	case invalidOp:
		return "invalid"
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
	default:
		return fmt.Sprintf("unknown item type: %d", it)
	}
}

// Item represents an operator or operand in an expression. It is exposed
// to allow client packages to create their own parsers.
type Item struct {
	typ itemType
	op  Operand
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
	case operand:
		if it.op == nil {
			return "<nil>"
		}
		return it.op.String()
	default:
		return fmt.Sprintf("unknown item type: %v", it.typ)
	}
}

// OR returns an OR item.
func OR() Item {
	return Item{typ: orOp}
}

// AND returns an AND item.
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

type evalNode interface {
	eval(v any) bool
	format(out *strings.Builder)
}

type leafNode struct {
	op Operand
}

func (n *leafNode) eval(v any) bool             { return n.op.Eval(v) }
func (n *leafNode) format(out *strings.Builder) { out.WriteString(n.op.String()) }

type notNode struct {
	child evalNode
}

func (n *notNode) eval(v any) bool { return !n.child.eval(v) }
func (n *notNode) format(out *strings.Builder) {
	out.WriteByte('!')
	n.child.format(out)
}

type parenNode struct {
	inner evalNode
}

func (n *parenNode) eval(v any) bool { return n.inner.eval(v) }
func (n *parenNode) format(out *strings.Builder) {
	out.WriteByte('(')
	n.inner.format(out)
	out.WriteByte(')')
}

type andNode struct {
	lhs, rhs evalNode
}

func (n *andNode) eval(v any) bool { return n.lhs.eval(v) && n.rhs.eval(v) }
func (n *andNode) format(out *strings.Builder) {
	n.lhs.format(out)
	out.WriteString(" && ")
	n.rhs.format(out)
}

type orNode struct {
	lhs, rhs evalNode
}

func (n *orNode) eval(v any) bool { return n.lhs.eval(v) || n.rhs.eval(v) }
func (n *orNode) format(out *strings.Builder) {
	n.lhs.format(out)
	out.WriteString(" || ")
	n.rhs.format(out)
}

// T represents a compiled boolean expression of Operand predicates evaluated
// against a single input value using standard precedence (! > && > ||) and
// short-circuit evaluation.
type T struct {
	root evalNode
	ops  []Operand
}

// IsEmpty returns true if the expression contains no operands or operators.
func (m T) IsEmpty() bool {
	return m.root == nil
}

// Needs returns true if any operand in the expression needs the type of typ.
func (m T) Needs(typ any) bool {
	if typ == nil {
		return false
	}
	return m.NeedsType(reflect.TypeOf(typ))
}

// NeedsType returns true if any operand in the expression needs t.
func (m T) NeedsType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	for _, op := range m.ops {
		if op.Needs(t) {
			return true
		}
	}
	return false
}

// NeedsFor returns true if any operand in the expression needs type V.
func (m T) NeedsFor[V any]() bool {
	return m.NeedsType(reflect.TypeFor[V]())
}

type itemParser struct {
	items []Item
	pos   int
	ops   []Operand
}

func (p *itemParser) parseBinary(op itemType, next func() (evalNode, error), combine func(evalNode, evalNode) evalNode) (evalNode, error) {
	lhs, err := next()
	if err != nil {
		return nil, err
	}
	for p.pos < len(p.items) && p.items[p.pos].typ == op {
		p.pos++
		rhs, err := next()
		if err != nil {
			return nil, err
		}
		lhs = combine(lhs, rhs)
	}
	return lhs, nil
}

func (p *itemParser) parseOr() (evalNode, error) {
	return p.parseBinary(orOp, p.parseAnd, func(lhs, rhs evalNode) evalNode {
		return &orNode{lhs: lhs, rhs: rhs}
	})
}

func (p *itemParser) parseAnd() (evalNode, error) {
	return p.parseBinary(andOp, p.parseUnary, func(lhs, rhs evalNode) evalNode {
		return &andNode{lhs: lhs, rhs: rhs}
	})
}

func (p *itemParser) parseUnary() (evalNode, error) {
	if p.pos < len(p.items) && p.items[p.pos].typ == notOp {
		p.pos++
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &notNode{child: child}, nil
	}
	return p.parsePrimary()
}

func (p *itemParser) parsePrimary() (evalNode, error) {
	if p.pos >= len(p.items) {
		return nil, fmt.Errorf("incomplete expression: expected operand or '(' at end of expression")
	}
	cur := p.items[p.pos]
	switch cur.typ {
	case operand:
		if cur.op == nil {
			return nil, fmt.Errorf("nil operand at item %d", p.pos)
		}
		op, err := cur.op.Prepare()
		if err != nil {
			return nil, err
		}
		p.pos++
		p.ops = append(p.ops, op)
		return &leafNode{op: op}, nil
	case leftBracket:
		openPos := p.pos
		p.pos++
		if p.pos >= len(p.items) {
			return nil, fmt.Errorf("unbalanced brackets: missing ')'")
		}
		if p.items[p.pos].typ == rightBracket {
			return nil, fmt.Errorf("empty subexpression '()' at item %d", openPos)
		}
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.items) {
			return nil, fmt.Errorf("unbalanced brackets: missing ')'")
		}
		if p.items[p.pos].typ != rightBracket {
			return nil, fmt.Errorf("missing operator before %v at item %d", p.items[p.pos].typ, p.pos)
		}
		p.pos++
		return &parenNode{inner: inner}, nil
	case rightBracket:
		return nil, fmt.Errorf("unbalanced brackets: unexpected ')' at item %d", p.pos)
	case andOp, orOp:
		return nil, fmt.Errorf("missing operand before %v at item %d", cur.typ, p.pos)
	default:
		return nil, fmt.Errorf("unexpected item %v at item %d", cur.typ, p.pos)
	}
}

// New returns a compiled T built from the supplied items. Operator precedence
// is ! (highest), then &&, then ||, with () for explicit grouping.
func New(items ...Item) (T, error) {
	if len(items) == 0 {
		return T{}, nil
	}
	p := itemParser{items: items}
	root, err := p.parseOr()
	if err != nil {
		return T{}, err
	}
	if p.pos < len(p.items) {
		cur := p.items[p.pos]
		if cur.typ == rightBracket {
			return T{}, fmt.Errorf("unbalanced brackets: unexpected ')' at item %d", p.pos)
		}
		return T{}, fmt.Errorf("missing operator before %v at item %d", cur.typ, p.pos)
	}
	return T{root: root, ops: p.ops}, nil
}

func (m T) String() string {
	if m.root == nil {
		return ""
	}
	var out strings.Builder
	m.root.format(&out)
	return out.String()
}

// Eval evaluates the expression against v using short-circuit evaluation.
// An empty, default T always returns false.
func (m T) Eval(v any) bool {
	if m.root == nil {
		return false
	}
	return m.root.eval(v)
}
