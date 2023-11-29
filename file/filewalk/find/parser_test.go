// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package find

import (
	"testing"
)

func runParser(t *testing.T, input string) tokenList {
	t.Helper()
	tok := tokenizer{}
	toks, err := tok.run(input)
	if err != nil {
		t.Fatalf("input %q err: %v", input, err)
	}
	return toks
}

func TestTokenParser(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"anything", "anything="},
		{"name=foo", "name='foo'"},
		{"name=''", "name=''"},
		{"name=foo || name=bar", "name='foo' || name='bar'"},
		{"name=foo && iname=bar", "name='foo' && iname='bar'"},
		{"name=foo&&iname=bar", "name='foo' && iname='bar'"},
		{"regexp='foo.*' || type=d && newer=2012-01-01", "regexp='foo.*' || type='d' && newer='2012-01-01'"},
		{"newer=2012-01-01\\ 20:00:00|| newer='2012-01-01 20:00:00'", "newer='2012-01-01 20:00:00' || newer='2012-01-01 20:00:00'"},
		{"name='f'|| (name='d' && newer=2012-01-01)", "name='f' || ( name='d' && newer='2012-01-01' )"},
		{"name='f'||(name='d'&& newer=2012-01-01)", "name='f' || ( name='d' && newer='2012-01-01' )"},
	} {
		toks := runParser(t, tc.input)
		if got, want := toks.String(), tc.output; got != want {
			t.Logf("tokens: #tokens: %v, %v\n", len(toks), toks)
			t.Errorf("%q: got %v, want %v", tc.input, got, want)
		}
	}
}

func TestTokenParserErros(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"name=foo |", "incomplete operator: |"},
		{"name=foo &", "incomplete operator: &"},
		{"name=foo & name=", "& is not a valid operator, should be &&"},
		{"name=foo | name=", "| is not a valid operator, should be ||"},
		{"=", "unexpected character: ="},
		{"'", "unexpected character: '"},
		{"\\", "unexpected character: \\"},
		{"name='foo", "missing close quote: foo"},
		{"name='foo''", "unexpected character: '"},
		{"name='foo'=", "unexpected character: ="},
		{"name='foo'\\", "unexpected character: \\"},
		{`name=\`, "missing escaped rune"},
		{"foo bar", "\"foo\": expected =, got ' '"},
	} {
		tok := tokenizer{}
		_, err := tok.run(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%q: missing or wrong error: %v", tc.input, err)
		}
	}
}

func TestParser(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"", ""},
		{"name=foo", "name=foo"},
		{"iname=foo", "iname=foo"},
		{"type=f", "type=f"},
		{"re=foo", "re=foo"},
		{"re=''", "re="},
		{"newer=2012-01-01", "newer=2012-01-01"},
		{"re=foo || newer=2012-01-01 && type=f", "re=foo || newer=2012-01-01 && type=f"},
		{"(re=foo || newer=2012-01-01) && type=f", "(re=foo || newer=2012-01-01) && type=f"},
	} {
		m, err := Parse(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
		}
		if got, want := m.String(), tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
}

func TestParserErrors(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"anything", "missing operand value: anything"},
		{"name=|", "incomplete operator: |"},
		{"re=&&", "incomplete expression: [re= &&]"},
		{"regexp=", "unsupported operand: regexp"},
		{"type=", "invalid file type: \"\", use one of d, f, l or x"},
		{"type= ", "invalid file type: \"\", use one of d, f, l or x"},
		{"type=z", "invalid file type: \"z\", use one of d, f, l or x"},
		{"(", "unbalanced brackets"},
		{")", "unbalanced brackets"},
		{")(", "unbalanced brackets"},
		{"(type=f && ))(", "unbalanced brackets"},
	} {
		_, err := Parse(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%q: missing or wrong error: %v", tc.input, err)
		}
	}
}
