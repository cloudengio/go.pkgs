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
	text     string
	operator bool
	operand  bool
	value    string
}

func operatorFor(text string) matcher.Item {
	switch text {
	case "or":
		return matcher.OR()
	case "and":
		return matcher.AND()
	case "(":
		return matcher.LeftBracket()
	case ")":
		return matcher.RightBracket()
	}
	return matcher.Item{}
}

func operandFor(text, value string) matcher.Item {
	switch text {
	case "name":
		return matcher.Glob(value, false)
	case "iname":
		return matcher.Glob(value, true)
	case "re":
		return matcher.Regexp(value)
	case "type":
		return matcher.FileType(value)
	case "newer":
		return matcher.NewerThanParsed(value)
	}
	return matcher.Item{}
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
		if t.operand {
			sb.WriteString(t.text + "=")
			continue
		}
		if len(t.value) > 0 {
			sb.WriteString(fmt.Sprintf("'%v' ", t.value))
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
		if tok.operand {
			fmt.Printf("XXX %v %v %v\n", tok.text, i, len(tl))
			if i+1 >= len(tl) {
				return nil, fmt.Errorf("missing operand value for: %v", tok.text)
			}
			next := tl[i+1]
			fmt.Printf("next... %#v\n", next)
			if next.operand || next.operator || len(next.value) == 0 {
				return nil, fmt.Errorf("missing operand value for: %v", tok.text)
			}
			merged = append(merged, operandFor(tok.text, next.value))
			i++
			continue
		}
		if len(tok.value) > 0 {
			return nil, fmt.Errorf("unexpected value: %v", tok.value)
		}
	}
	return merged, nil
}

type state int

const (
	start state = iota
	item
	operand
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
		case item: // seen the first character of an operand or operator
			state, err = t.item(r)
		case operandValue: // seen an operand, i,e. <text>=, expecting a value
			// for that operand next.
			state, err = t.operandValue(r)
		case quotedValue:
			state, err = t.quotedValue(r)
		case escapedValue:
			state, err = t.escapedValue(r)
		case escapedRune:
			state, err = t.escapedRune(r)
		}
		if err != nil {
			return t.tokens, err
		}
	}
	switch state {
	case item:
		return t.tokens, fmt.Errorf("incomplete operator or operand: %v", t.seen.String())
	case operandValue:
		seen := t.seen.String()
		if len(t.tokens) > 0 {
			seen = t.tokens[len(t.tokens)-1].text
		}
		return t.tokens, fmt.Errorf("missing operand value: %v", seen)
	case quotedValue:
		return t.tokens, fmt.Errorf("incomplete quoted value: %v", t.seen.String())
	case escapedValue:
		t.tokens = append(t.tokens, token{
			value: t.seen.String(),
		})
		return t.tokens, nil
	case escapedRune:
		return t.tokens, fmt.Errorf("incomplete escaped rune")
	}
	return t.tokens, nil
}

func (t *tokenizer) start(r rune) (state, error) {
	if unicode.IsSpace(r) {
		return start, nil // consume white space
	}
	if r == '(' || r == ')' {
		t.tokens = append(t.tokens, token{
			text:     string(r),
			operator: true,
		})
		return start, nil
	}
	t.seen.WriteRune(r)
	return item, nil
}

func (t *tokenizer) item(r rune) (state, error) {
	if r == '=' {
		t.tokens = append(t.tokens, token{
			text:    t.seen.String(),
			operand: true,
		})
		t.seen.Reset()
		return operandValue, nil
	}
	if unicode.IsSpace(r) {
		op := t.seen.String()
		if op == "and" || op == "or" || op == "(" || op == ")" {
			t.tokens = append(t.tokens, token{
				text:     op,
				operator: true,
			})
		} else {
			return start, fmt.Errorf("unknown operator: %v, should be one of 'or', 'and', '( or ')'", op)
		}
		t.seen.Reset()
		return start, nil
	}
	t.seen.WriteRune(r)
	return item, nil
}

func (t *tokenizer) operandValue(r rune) (state, error) {
	if r == '\'' {
		return quotedValue, nil // quoted value
	}
	if r == '\\' {
		return escapedRune, nil
	}
	t.seen.WriteRune(r)
	return escapedValue, nil
}

func (t *tokenizer) quotedValue(r rune) (state, error) {
	if r == '\'' {
		t.tokens = append(t.tokens, token{
			value: t.seen.String(),
		})
		t.seen.Reset()
		return start, nil
	}
	t.seen.WriteRune(r)
	return quotedValue, nil
}

func (t *tokenizer) escapedValue(r rune) (state, error) {
	if unicode.IsSpace(r) || r == '(' || r == ')' {
		t.tokens = append(t.tokens, token{
			value: t.seen.String(),
		})
		t.seen.Reset()
		if r == '(' || r == ')' {
			t.tokens = append(t.tokens, token{
				text:     string(r),
				operator: true,
			})
		}
		return start, nil
	}
	if r == '\\' {
		return escapedRune, nil
	}
	t.seen.WriteRune(r)
	return escapedValue, nil
}

func (t *tokenizer) escapedRune(r rune) (state, error) {
	t.seen.WriteRune(r)
	return escapedValue, nil
}
