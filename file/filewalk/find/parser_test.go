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

func TestParser(t *testing.T) {
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"name=foo", "name='foo'"},
		{"name=foo or name=bar", "name='foo' or name='bar'"},
		{"name=foo and iname=bar", "name='foo' and iname='bar'"},
		{"regexp='foo.*' or type=d and newer=2012-01-01", "regexp='foo.*' or type='d' and newer='2012-01-01'"},
		{"newer=2012-01-01\\ 20:00:00 or newer='2012-01-01 20:00:00'", "newer='2012-01-01 20:00:00' or newer='2012-01-01 20:00:00'"},
		{"name='f' or (name='d' and newer=2012-01-01)", "name='f' or ( name='d' and newer='2012-01-01' )"},
	} {
		toks := runParser(t, tc.input)
		if got, want := toks.String(), tc.output; got != want {
			t.Errorf("%q: got %v, want %v", tc.input, got, want)
		}
		t.Log(tc.input)
	}
}

func TestParserErros(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"name=foo or", "incomplete operator or operand: or"},
		{"name=foo or name=", "missing operand value: name"},
		{"=", "incomplete operator or operand: ="},
		{"or", "incomplete operator or operand: or"},
		{"name='foo", "incomplete quoted value: foo"},
		{"name='foo''", "incomplete operator or operand: '"},
		{`name=\`, "incomplete escaped rune"},
		{"foo bar", "unknown operator: foo, should be one of 'or', 'and', '( or ')'"},
	} {
		tok := tokenizer{}
		_, err := tok.run(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%q: missing or wrong error: %v", tc.input, err)
		}
	}
}
