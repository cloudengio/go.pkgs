// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package find

import (
	"fmt"
	"strings"
	"unicode"

	"cloudeng.io/file/matcher"
)

// Parse parses the supplied input into a matcher.T.
// The supported syntax is a boolean expression with
// and (&&), or (||) and grouping, via ().
// The supported operands are:
//
//		name='glob-pattern'
//		iname='glob-pattern'
//		re='regexp'
//		type='f|d|l'
//		newer='date' in time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly
//
//	 Note that the single quotes are optional unless a white space is present
//	 in the pattern.
func Parse(input string) (matcher.T, error) {
	tokenizer := &tokenizer{}
	tokens, err := tokenizer.run(input)
	if err != nil {
		return matcher.T{}, err
	}
	merged, err := mergeOperandsAndValues(tokens)
	if err != nil {
		return matcher.T{}, err
	}
	return matcher.New(merged...)
}

type token struct {
	text                      string
	operator                  bool
	operandName, operandValue bool
}

func operatorFor(text string) matcher.Item {
	switch text {
	case "||":
		return matcher.OR()
	case "&&":
		return matcher.AND()
	case "(":
		return matcher.LeftBracket()
	case ")":
		return matcher.RightBracket()
	}
	return matcher.Item{}
}

var bultinOperands = map[string]func(name, value string) matcher.Item{
	"name":  func(_, v string) matcher.Item { return matcher.Glob(v, false) },
	"iname": func(_, v string) matcher.Item { return matcher.Glob(v, true) },
	"re":    func(_, v string) matcher.Item { return matcher.Regexp(v) },
	"type":  func(_, v string) matcher.Item { return matcher.FileType(v) },
	"newer": func(_, v string) matcher.Item { return matcher.NewerThanParsed(v) },
}

func operandFor(text, value string) (matcher.Item, error) {
	if fn, ok := bultinOperands[text]; ok {
		return fn(text, value), nil
	}
	return matcher.Item{}, fmt.Errorf("unsupported operand: %v", text)
}

type tokenizer struct {
	seen   strings.Builder
	tokens tokenList
}

type tokenList []token

func (tl tokenList) String() string {
	var sb strings.Builder
	for _, t := range tl {
		if t.operator {
			sb.WriteString(t.text + " ")
			continue
		}
		if t.operandName {
			sb.WriteString(t.text + "=")
			continue
		}
		if t.operandValue {
			sb.WriteString(fmt.Sprintf("'%v' ", t.text))
		}
	}
	return strings.TrimSpace(sb.String())
}

func mergeOperandsAndValues(tl []token) ([]matcher.Item, error) {
	var merged []matcher.Item
	for i := 0; i < len(tl); i++ {
		tok := tl[i]
		if tok.operator {
			merged = append(merged, operatorFor(tok.text))
			continue
		}
		if tok.operandName {
			if i+1 >= len(tl) {
				return nil, fmt.Errorf("missing operand value: %v", tok.text)
			}
			next := tl[i+1]
			if !next.operandValue {
				return nil, fmt.Errorf("missing operand value: %v", tok.text)
			}
			op, err := operandFor(tok.text, next.text)
			if err != nil {
				return nil, err
			}
			merged = append(merged, op)
			i++
			continue
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
	escapedValue
	escapedRune
)

// input stream looks like:
// <op>='value' - no escaping within quotes
// <op>=value - with possible escaping of white space
// or, and, (, )
func (t *tokenizer) run(input string) ([]token, error) {
	state := start
	for _, r := range input {
		var err error
		switch state {
		case start: // white space, in-between tokens
			state, err = t.start(r)
		case operatorAnd: // seen a single &
			state, err = t.operatorAnd(r)
		case operatorOr: // seen a single |
			state, err = t.operatorOr(r)
		case operandName: // expecting an operand name, <text> terminated by =
			state, err = t.operandName(r)
		case operandValue: // seen an operand =, i,e. <text>=, expecting a value
			state, err = t.operandValue(r)
		case quotedValue: // quoted value, i.e. <text>='value'
			state, err = t.quotedValue(r)
		case escapedValue: // escaped rune, i.e. \s
			state, err = t.escapedValue(r)
		case escapedRune: // seen a \, always returns to escapedValue
			state, err = t.escapedRune(r)
		}
		if err != nil {
			return t.tokens, err
		}
	}
	switch state {
	case operandName:
		t.appendOperandName()
	case operandValue, escapedValue:
		t.appendOperandValue()
	case operatorAnd:
		return t.tokens, fmt.Errorf("incomplete operator: &")
	case operatorOr:
		return t.tokens, fmt.Errorf("incomplete operator: |")
	case quotedValue:
		return t.tokens, fmt.Errorf("missing close quote: %v", t.seen.String())
	case escapedRune:
		return t.tokens, fmt.Errorf("missing escaped rune")
	}
	return t.tokens, nil
}

func (t *tokenizer) appendOperator(text string) {
	t.tokens = append(t.tokens, token{
		text:     text,
		operator: true,
	})
}

func (t *tokenizer) appendOperandName() {
	t.tokens = append(t.tokens, token{
		text:        t.seen.String(),
		operandName: true,
	})
	t.seen.Reset()
}

func (t *tokenizer) appendOperandValue() {
	t.tokens = append(t.tokens, token{
		text:         t.seen.String(),
		operandValue: true,
	})
	t.seen.Reset()
}

func (t *tokenizer) start(r rune) (state, error) {
	if unicode.IsSpace(r) {
		return start, nil // consume white space
	}
	switch r {
	case '(', ')':
		t.appendOperator(string(r))
		return start, nil
	case '&':
		return operatorAnd, nil
	case '|':
		return operatorOr, nil
	}
	if unicode.IsLetter(r) {
		t.seen.WriteRune(r)
		return operandName, nil
	}
	return start, fmt.Errorf("unexpected character: %c", r)
}

func (t *tokenizer) operatorAnd(r rune) (state, error) {
	if r == '&' {
		t.tokens = append(t.tokens, token{text: "&&", operator: true})
		t.seen.Reset()
		return start, nil
	}
	return start, fmt.Errorf("& is not a valid operator, should be &&")
}

func (t *tokenizer) operatorOr(r rune) (state, error) {
	if r == '|' {
		t.tokens = append(t.tokens, token{text: "||", operator: true})
		t.seen.Reset()
		return start, nil
	}
	return start, fmt.Errorf("| is not a valid operator, should be ||")
}

func (t *tokenizer) operandName(r rune) (state, error) {
	if r == '=' {
		t.appendOperandName()
		return operandValue, nil
	}
	if unicode.IsLetter(r) || unicode.IsNumber(r) {
		t.seen.WriteRune(r)
		return operandName, nil
	}
	return start, fmt.Errorf("%q: expected =, got '%c'", t.seen.String(), r)
}

func (t *tokenizer) operandValueDone(r rune) (bool, state, error) {
	if unicode.IsSpace(r) {
		t.appendOperandValue()
		return true, start, nil
	}
	switch r {
	case '(', ')':
		t.appendOperandValue()
		t.appendOperator(string(r))
		return true, start, nil
	case '&':
		t.appendOperandValue()
		return true, operatorAnd, nil
	case '|':
		t.appendOperandValue()
		return true, operatorOr, nil
	}
	t.seen.WriteRune(r)
	return false, escapedValue, nil
}

func (t *tokenizer) operandValue(r rune) (state, error) {
	if r == '\'' {
		return quotedValue, nil // quoted value
	}
	if r == '\\' {
		// escapedRune always returns to escapedValue
		return escapedRune, nil
	}
	if done, state, err := t.operandValueDone(r); done {
		return state, err
	}
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

func (t *tokenizer) escapedValue(r rune) (state, error) {
	if r == '\\' {
		return escapedRune, nil
	}
	if done, state, err := t.operandValueDone(r); done {
		return state, err
	}
	return escapedValue, nil
}

func (t *tokenizer) escapedRune(r rune) (state, error) {
	t.seen.WriteRune(r)
	return escapedValue, nil
}
