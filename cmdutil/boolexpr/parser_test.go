// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
)

func newParserRE() *boolexpr.Parser {
	parser := boolexpr.NewParser()
	parser.RegisterOperand("re", func(_, v string) boolexpr.Operand { return regexOp{val: v} })
	return parser
}

func TestOperandRegistration(t *testing.T) {
	p := boolexpr.NewParser()
	p.RegisterOperand("newOp", func(_, v string) boolexpr.Operand { return regexOp{val: v} })
	if !p.HasOperand("newOp") {
		t.Fatalf("expected HasOperand(newOp) == true")
	}
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
	p.RemoveOperand("newOp")
	if p.HasOperand("newOp") {
		t.Fatalf("expected HasOperand(newOp) == false after removal")
	}
}

func TestParser(t *testing.T) {
	parser := newParserRE()
	parser.RegisterOperand("op2", func(_, v string) boolexpr.Operand { return regexOp{val: v, name: "op2"} })
	parser.RegisterOperand("_priv", func(_, v string) boolexpr.Operand { return regexOp{val: v, name: "_priv"} })

	for _, tc := range []struct {
		input  string
		output string
	}{
		{"", ""},
		{"   ", ""},
		{"re=foo", "re=foo"},
		{"_priv=foo", "_priv=foo"},
		{"!re=foo", "!re=foo"},
		{"!!re=foo", "!!re=foo"},
		{"re=\\ a", "re= a"},
		{"re=''", "re="},
		{`re=""`, "re="},
		{`re="it's \"ok\""`, `re=it's "ok"`},
		{`re='foo\.bar'`, `re=foo\.bar`},
		{"re=foo || re=bar && re=baz", "re=foo || re=bar && re=baz"},
		{"re=foo || re=bar && !re=baz", "re=foo || re=bar && !re=baz"},
		{"(re=foo || op2=baz) && op2=f", "(re=foo || op2=baz) && op2=f"},
	} {
		m, err := parser.Parse(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if got, want := m.String(), tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}

	m, err := parser.Parse("re=a || re=b && re=c")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Eval("a") || m.Eval("b") || !m.Eval("bc") {
		t.Errorf("unexpected precedence result for %q", m)
	}
}

func TestParserErrors(t *testing.T) {
	parser := newParserRE()
	parser.RegisterOperand("nilOp", nil)
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"anything", `unsupported operand "anything" at position 0`},
		{"anything=", "missing operand value at position 9"},
		{"re=", "missing operand value at position 3"},
		{"re= && re=a", "missing operand value at position 3"},
		{"re=&&", "missing operand value at position 3"},
		{"anything=|", "missing operand value at position 9"},
		{"re=|", "missing operand value at position 3"},
		{"regexp=", "missing operand value at position 7"},
		{"nilOp=a", `nil factory for operand "nilOp" at position 0`},
		{"(", "unbalanced brackets: missing ')'"},
		{")", "unbalanced brackets: unexpected ')' at item 0"},
		{")(", "unbalanced brackets: unexpected ')' at item 0"},
		{"(re=f && ))(", "unbalanced brackets: unexpected ')' at item 3"},
		{"!!", "incomplete expression: expected operand or '(' at end of expression"},
		{"re=a re=b re=c", "missing operator before operand at item 1"},
	} {
		_, err := parser.Parse(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%q: missing or wrong error: got %v, want %v", tc.input, err, tc.err)
		}
	}
}

func TestList(t *testing.T) {
	parser := newParserRE()
	parser.RegisterOperand("op2", func(_, v string) boolexpr.Operand { return regexOp{val: v, name: "op2"} })

	operands := parser.ListOperands()
	if got, want := len(operands), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	var doc strings.Builder
	for _, op := range operands {
		doc.WriteString(op.Document() + "\n")
	}

	if got, want := doc.String(), "op2: regular expression\nre: regular expression\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
