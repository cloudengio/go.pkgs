// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr

import (
	"testing"
)

func runTokenizer(t *testing.T, input string) tokenList {
	t.Helper()
	tok := tokenizer{}
	toks, err := tok.run(input)
	if err != nil {
		t.Fatalf("input %q err: %v", input, err)
	}
	return toks
}

func TestTokenizer(t *testing.T) {
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
		toks := runTokenizer(t, tc.input)
		if got, want := toks.String(), tc.output; got != want {
			t.Logf("tokens: #tokens: %v, %v\n", len(toks), toks)
			t.Errorf("%q: got %v, want %v", tc.input, got, want)
		}
	}
}

func TestTokenizerErros(t *testing.T) {
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
