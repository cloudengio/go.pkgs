// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Parser parses text boolean expressions using registered Operand factories.
type Parser struct {
	mu  sync.RWMutex
	ops map[string]func(name, value string) Operand
}

// RegisterOperand registers an operand factory under name. To be usable via
// Parse, name should start with a letter or '_' and contain only letters,
// digits, '-', or '_'. The factory must tolerate an empty value when called by
// ListOperands for documentation.
func (p *Parser) RegisterOperand(name string, factory func(name, value string) Operand) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ops[name] = factory
}

// RemoveOperand removes any operand factory registered under name.
func (p *Parser) RemoveOperand(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.ops, name)
}

// HasOperand returns true if an operand factory is registered under name.
func (p *Parser) HasOperand(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.ops[name]
	return ok
}

// NewParser returns a new Parser with no registered operands.
func NewParser() *Parser {
	return &Parser{ops: make(map[string]func(name, value string) Operand)}
}

type registeredOperand struct {
	name string
	fn   func(name, value string) Operand
}

func (p *Parser) snapshotOperands() []registeredOperand {
	p.mu.RLock()
	defer p.mu.RUnlock()
	keys := make([]string, 0, len(p.ops))
	for k := range p.ops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	registered := make([]registeredOperand, 0, len(keys))
	for _, k := range keys {
		if fn := p.ops[k]; fn != nil {
			registered = append(registered, registeredOperand{name: k, fn: fn})
		}
	}
	return registered
}

// ListOperands returns the list of registered operands in alphanumeric order.
func (p *Parser) ListOperands() []Operand {
	registered := p.snapshotOperands()
	ops := make([]Operand, 0, len(registered))
	for _, entry := range registered {
		ops = append(ops, entry.fn(entry.name, ""))
	}
	return ops
}

// Parse parses the supplied input into a boolexpr.T. The supported operators
// are ! (negation), && (and), || (or), with standard precedence (! > && > ||)
// and grouping via (). Operands are represented as <operand>=<value> where the
// value is interpreted by the operand. The <value> may be raw single-quoted
// ('...'), double-quoted ("..." with \ escapes), or unquoted with \-escaped
// runes. An empty or whitespace-only input returns an empty T whose Eval
// returns false. The set of available operands is those registered with the
// parser before Parse is called.
func (p *Parser) Parse(input string) (T, error) {
	tokenizer := &tokenizer{}
	tokens, err := tokenizer.run(input)
	if err != nil {
		return T{}, err
	}
	merged, err := p.mergeOperandsAndValues(tokens)
	if err != nil {
		return T{}, err
	}
	return New(merged...)
}

type token struct {
	text                      string
	pos                       int
	operator                  bool
	operandName, operandValue bool
}

func operatorFor(text string) Item {
	switch text {
	case "||":
		return OR()
	case "&&":
		return AND()
	case "!":
		return NOT()
	case "(":
		return LeftBracket()
	case ")":
		return RightBracket()
	}
	return Item{}
}

type tokenizer struct {
	seen     strings.Builder
	tokens   tokenList
	tokStart int
}

type tokenList []token

func (tl tokenList) String() string {
	var sb strings.Builder
	for _, t := range tl {
		if t.operator {
			sb.WriteString(t.text)
			sb.WriteByte(' ')
			continue
		}
		if t.operandName {
			sb.WriteString(t.text)
			sb.WriteByte('=')
			continue
		}
		if t.operandValue {
			sb.WriteByte('\'')
			sb.WriteString(t.text)
			sb.WriteString("' ")
		}
	}
	return strings.TrimSpace(sb.String())
}

func (p *Parser) mergeOperandsAndValues(tl []token) ([]Item, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	merged := make([]Item, 0, len(tl))
	for i := 0; i < len(tl); i++ {
		tok := tl[i]
		if tok.operator {
			merged = append(merged, operatorFor(tok.text))
			continue
		}
		if tok.operandName {
			fn, ok := p.ops[tok.text]
			if !ok {
				return nil, fmt.Errorf("unsupported operand %q at position %d", tok.text, tok.pos)
			}
			if i+1 >= len(tl) || !tl[i+1].operandValue {
				return nil, fmt.Errorf("missing operand value for %q at position %d", tok.text, tok.pos)
			}
			if fn == nil {
				return nil, fmt.Errorf("nil factory for operand %q at position %d", tok.text, tok.pos)
			}
			merged = append(merged, NewOperandItem(fn(tok.text, tl[i+1].text)))
			i++
		}
	}
	return merged, nil
}

type state int

const (
	start state = iota
	operandName
	operatorAnd
	operatorOr
	operandValue
	quotedValue
	doubleQuotedValue
	doubleQuotedEscapedRune
	escapedValue
	escapedRune
)

func (t *tokenizer) runStateMachine(input string) (state, error) {
	state := start
	for pos, r := range input {
		var err error
		switch state {
		case start:
			state, err = t.start(pos, r)
		case operatorAnd:
			state, err = t.matchOperator('&', "&&", r)
		case operatorOr:
			state, err = t.matchOperator('|', "||", r)
		case operandName:
			state, err = t.operandName(pos, r)
		case operandValue:
			state, err = t.operandValue(pos, r)
		case quotedValue:
			state, err = t.quotedValue(r)
		case doubleQuotedValue:
			state, err = t.doubleQuotedValue(r)
		case doubleQuotedEscapedRune:
			state, err = t.doubleQuotedEscapedRune(r)
		case escapedValue:
			state, err = t.escapedValue(pos, r)
		case escapedRune:
			state, err = t.escapedRune(r)
		}
		if err != nil {
			return start, err
		}
	}
	return state, nil
}

// input stream looks like:
// <op>='value' - no escaping within single quotes
// <op>="value" - backslash escaping within double quotes
// <op>=value   - unquoted with backslash escaping
// operators: &&, ||, !, (, )
func (t *tokenizer) run(input string) ([]token, error) {
	state, err := t.runStateMachine(input)
	if err != nil {
		return t.tokens, err
	}
	endPos := len(input)
	switch state {
	case operandName:
		t.appendOperandName()
	case operandValue:
		return t.tokens, fmt.Errorf("missing operand value at position %d", endPos)
	case escapedValue:
		t.appendOperandValue()
	case operatorAnd:
		return t.tokens, fmt.Errorf("incomplete operator '&' at position %d", t.tokStart)
	case operatorOr:
		return t.tokens, fmt.Errorf("incomplete operator '|' at position %d", t.tokStart)
	case quotedValue:
		return t.tokens, fmt.Errorf("missing close quote ' for value %q at position %d", t.seen.String(), t.tokStart)
	case doubleQuotedValue, doubleQuotedEscapedRune:
		return t.tokens, fmt.Errorf("missing close quote \" for value %q at position %d", t.seen.String(), t.tokStart)
	case escapedRune:
		return t.tokens, fmt.Errorf("missing escaped rune at position %d", endPos)
	}
	return t.tokens, nil
}

func (t *tokenizer) appendOperator(text string, pos int) {
	t.tokens = append(t.tokens, token{
		text:     text,
		pos:      pos,
		operator: true,
	})
}

func (t *tokenizer) appendOperandName() {
	t.tokens = append(t.tokens, token{
		text:        t.seen.String(),
		pos:         t.tokStart,
		operandName: true,
	})
	t.seen.Reset()
}

func (t *tokenizer) appendOperandValue() {
	t.tokens = append(t.tokens, token{
		text:         t.seen.String(),
		pos:          t.tokStart,
		operandValue: true,
	})
	t.seen.Reset()
}

func (t *tokenizer) start(pos int, r rune) (state, error) {
	if unicode.IsSpace(r) {
		return start, nil // consume white space
	}
	switch r {
	case '(', ')', '!':
		t.appendOperator(string(r), pos)
		return start, nil
	case '&':
		t.tokStart = pos
		return operatorAnd, nil
	case '|':
		t.tokStart = pos
		return operatorOr, nil
	}
	if unicode.IsLetter(r) || r == '_' {
		t.tokStart = pos
		t.seen.WriteRune(r)
		return operandName, nil
	}
	return start, fmt.Errorf("unexpected character %q at position %d", r, pos)
}

func (t *tokenizer) matchOperator(want rune, op string, r rune) (state, error) {
	if r == want {
		t.appendOperator(op, t.tokStart)
		return start, nil
	}
	return start, fmt.Errorf("%q is not a valid operator at position %d, should be '%s'", want, t.tokStart, op)
}

func (t *tokenizer) operandName(pos int, r rune) (state, error) {
	if r == '=' {
		t.appendOperandName()
		return operandValue, nil
	}
	if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' {
		t.seen.WriteRune(r)
		return operandName, nil
	}
	return start, fmt.Errorf("%q: expected '=', got %q at position %d", t.seen.String(), r, pos)
}

func (t *tokenizer) operandValueDone(pos int, r rune) (bool, state, error) {
	if unicode.IsSpace(r) {
		t.appendOperandValue()
		return true, start, nil
	}
	switch r {
	case '(', ')':
		t.appendOperandValue()
		t.appendOperator(string(r), pos)
		return true, start, nil
	case '&':
		t.appendOperandValue()
		t.tokStart = pos
		return true, operatorAnd, nil
	case '|':
		t.appendOperandValue()
		t.tokStart = pos
		return true, operatorOr, nil
	}
	t.seen.WriteRune(r)
	return false, escapedValue, nil
}

func (t *tokenizer) operandValue(pos int, r rune) (state, error) {
	t.tokStart = pos
	switch r {
	case '\'':
		return quotedValue, nil
	case '"':
		return doubleQuotedValue, nil
	case '\\':
		return escapedRune, nil
	}
	if unicode.IsSpace(r) || r == '(' || r == ')' || r == '&' || r == '|' {
		return start, fmt.Errorf("missing operand value at position %d", pos)
	}
	t.seen.WriteRune(r)
	return escapedValue, nil
}

func (t *tokenizer) quotedValue(r rune) (state, error) {
	if r == '\'' {
		t.appendOperandValue()
		return start, nil
	}
	t.seen.WriteRune(r)
	return quotedValue, nil
}

func (t *tokenizer) doubleQuotedValue(r rune) (state, error) {
	switch r {
	case '"':
		t.appendOperandValue()
		return start, nil
	case '\\':
		return doubleQuotedEscapedRune, nil
	}
	t.seen.WriteRune(r)
	return doubleQuotedValue, nil
}

func (t *tokenizer) doubleQuotedEscapedRune(r rune) (state, error) {
	t.seen.WriteRune(r)
	return doubleQuotedValue, nil
}

func (t *tokenizer) escapedValue(pos int, r rune) (state, error) {
	if r == '\\' {
		return escapedRune, nil
	}
	if done, state, err := t.operandValueDone(pos, r); done {
		return state, err
	}
	return escapedValue, nil
}

func (t *tokenizer) escapedRune(r rune) (state, error) {
	t.seen.WriteRune(r)
	return escapedValue, nil
}
