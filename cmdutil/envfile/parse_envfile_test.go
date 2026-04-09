// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/envfile"
)

func env(pairs ...string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		desc  string
		input string
		want  map[string]string
	}{
		{
			desc:  "empty input",
			input: "",
			want:  env(),
		},
		{
			desc:  "blank lines only",
			input: "\n\n   \n",
			want:  env(),
		},
		{
			desc:  "comment lines only",
			input: "# comment\n  # indented comment\n",
			want:  env(),
		},
		{
			desc:  "simple unquoted",
			input: "FOO=bar\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "unquoted with inline comment",
			input: "FOO=bar # comment\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "unquoted empty value",
			input: "FOO=\n",
			want:  env("FOO", ""),
		},
		{
			desc:  "unquoted value trims leading whitespace",
			input: "FOO=  bar\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "unquoted value trims trailing whitespace",
			input: "FOO=bar   \n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "unquoted value with spaces in middle",
			input: "FOO=hello world\n",
			want:  env("FOO", "hello world"),
		},
		{
			desc:  "single-quoted value",
			input: "FOO='bar baz'\n",
			want:  env("FOO", "bar baz"),
		},
		{
			desc:  "single-quoted preserves hash",
			input: "FOO='bar # not a comment'\n",
			want:  env("FOO", "bar # not a comment"),
		},
		{
			desc:  "single-quoted empty",
			input: "FOO=''\n",
			want:  env("FOO", ""),
		},
		{
			desc:  "single-quoted preserves backslash literally",
			input: `FOO='a\nb'` + "\n",
			want:  env("FOO", `a\nb`),
		},
		{
			desc:  "double-quoted value",
			input: `FOO="bar baz"` + "\n",
			want:  env("FOO", "bar baz"),
		},
		{
			desc:  "double-quoted preserves hash",
			input: `FOO="bar # not a comment"` + "\n",
			want:  env("FOO", "bar # not a comment"),
		},
		{
			desc:  "double-quoted empty",
			input: `FOO=""` + "\n",
			want:  env("FOO", ""),
		},
		{
			desc:  "double-quoted escape sequences",
			input: `FOO="a\nb\tc\rd"` + "\n",
			want:  env("FOO", "a\nb\tc\rd"),
		},
		{
			desc:  "double-quoted escaped quote",
			input: `FOO="say \"hello\""` + "\n",
			want:  env("FOO", `say "hello"`),
		},
		{
			desc:  "double-quoted escaped backslash",
			input: `FOO="a\\b"` + "\n",
			want:  env("FOO", `a\b`),
		},
		{
			desc:  "double-quoted escaped dollar",
			input: `FOO="\$HOME"` + "\n",
			want:  env("FOO", "$HOME"),
		},
		{
			desc:  "double-quoted unrecognised escape kept",
			input: `FOO="\q"` + "\n",
			want:  env("FOO", `\q`),
		},
		{
			desc:  "export prefix",
			input: "export FOO=bar\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "export prefix with tab",
			input: "export\tFOO=bar\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "leading whitespace on line",
			input: "   FOO=bar\n",
			want:  env("FOO", "bar"),
		},
		{
			desc:  "line without = is skipped",
			input: "JUSTNAME\n",
			want:  env(),
		},
		{
			desc:  "export without = is skipped",
			input: "export JUSTNAME\n",
			want:  env(),
		},
		{
			desc:  "underscore-only name",
			input: "_=value\n",
			want:  env("_", "value"),
		},
		{
			desc:  "name with underscores and digits",
			input: "_FOO_123=value\n",
			want:  env("_FOO_123", "value"),
		},
		{
			desc:  "duplicate name: last value wins",
			input: "FOO=first\nFOO=second\n",
			want:  env("FOO", "second"),
		},
		{
			desc: "multiple vars",
			input: `
# database settings
DB_HOST=localhost
DB_PORT=5432
DB_NAME="mydb"
DB_PASS='s3cr3t!'
export DB_USER=admin
`,
			want: env(
				"DB_HOST", "localhost",
				"DB_PORT", "5432",
				"DB_NAME", "mydb",
				"DB_PASS", "s3cr3t!",
				"DB_USER", "admin",
			),
		},
		{
			desc:  "hash without preceding whitespace is part of value",
			input: "FOO=#notacomment\n",
			want:  env("FOO", "#notacomment"),
		},
		{
			desc:  "hash embedded without whitespace is part of value",
			input: "FOO=a#b\n",
			want:  env("FOO", "a#b"),
		},
		{
			desc:  "hash preceded by whitespace starts inline comment",
			input: "FOO=  # nothing\n",
			want:  env("FOO", ""),
		},
	} {
		got, err := envfile.Parse(strings.NewReader(tc.input))
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tc.desc, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("%s: got %d entries, want %d: %v", tc.desc, len(got), len(tc.want), got)
			continue
		}
		for k, wantV := range tc.want {
			if gotV, ok := got[k]; !ok {
				t.Errorf("%s: missing key %q", tc.desc, k)
			} else if gotV != wantV {
				t.Errorf("%s: key %q: got %q, want %q", tc.desc, k, gotV, wantV)
			}
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, tc := range []struct {
		desc   string
		input  string
		errMsg string
	}{
		{
			desc:   "unterminated single quote",
			input:  "FOO='oops\n",
			errMsg: "unterminated single-quoted value",
		},
		{
			desc:   "unterminated double quote",
			input:  `FOO="oops` + "\n",
			errMsg: "unterminated double-quoted value",
		},
		{
			desc:   "invalid name starting with digit",
			input:  "1FOO=bar\n",
			errMsg: "invalid variable name",
		},
		{
			desc:   "invalid name with hyphen",
			input:  "FOO-BAR=bar\n",
			errMsg: "invalid variable name",
		},
	} {
		_, err := envfile.Parse(strings.NewReader(tc.input))
		if err == nil {
			t.Errorf("%s: expected error, got nil", tc.desc)
			continue
		}
		if !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("%s: got error %q, want it to contain %q", tc.desc, err.Error(), tc.errMsg)
		}
	}
}
