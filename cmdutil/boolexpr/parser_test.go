// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr_test

import (
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
)

func newParserRE() *boolexpr.Parser {
	parser := boolexpr.NewParser()
	parser.RegisterOperand("re", func(n, v string) boolexpr.Operand { return regexOp{val: v} })
	return parser
}

func TestOperandRegistration(t *testing.T) {
	p := boolexpr.NewParser()
	p.RegisterOperand("newOp", func(n, v string) boolexpr.Operand { return regexOp{val: v} })
	m, err := p.Parse("newOp=foo")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.Eval("foo"), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := m.Eval("bar"), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParser(t *testing.T) {
	parser := newParserRE()
	parser.RegisterOperand("op2", func(n, v string) boolexpr.Operand { return regexOp{val: v, name: "op2"} })

	for _, tc := range []struct {
		input  string
		output string
	}{
		{"", ""},
		{"re=foo", "re=foo"},
		{"re=\\ a", "re= a"},
		{"re=''", "re="},
		{"re=foo || re=bar && re=baz", "re=foo || re=bar && re=baz"},
		{"(re=foo || op2=baz) && op2=f", "(re=foo || op2=baz) && op2=f"},
	} {
		m, err := parser.Parse(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
		}
		if got, want := m.String(), tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
}

func TestParserErrors(t *testing.T) {
	parser := newParserRE()
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"anything", "unsupported operand: anything"},
		{"anything=", "unsupported operand: anything"},
		{"anything=|", "incomplete operator: |"},
		{"re=|", "incomplete operator: |"},
		{"re=&&", "incomplete expression: [re= &&]"},
		{"regexp=", "unsupported operand: regexp"},
		{"(", "unbalanced brackets"},
		{")", "unbalanced brackets"},
		{")(", "unbalanced brackets"},
		{"(re=f && ))(", "unbalanced brackets"},
	} {
		_, err := parser.Parse(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%q: missing or wrong error: %v", tc.input, err)
		}
	}
}

func TestList(t *testing.T) {
	parser := newParserRE()
	parser.RegisterOperand("op2", func(n, v string) boolexpr.Operand { return regexOp{val: v, name: "op2"} })

	operands := parser.ListOperands()
	if got, want := len(operands), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	doc := ""
	for _, op := range operands {
		doc += op.Document() + "\n"
	}

	if got, want := doc, "op2: regular expression\nre: regular expression\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
