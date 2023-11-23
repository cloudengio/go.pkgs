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
		t.Fatal(err)
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
	}
}

func TestParserErros(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{"name=foo or", "unexpected end of input"},
		{"name=foo or name=", "unexpected end of input"},
	} {
		tok := tokenizer{}
		_, err := tok.run(tc.input)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%qq: missing or wrong error: %v", tc.input, err)
		}
	}
}
