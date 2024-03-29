// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil_test

import (
	"testing"

	"cloudeng.io/text/textutil"
)

func TestRewriteParsing(t *testing.T) {
	for _, tc := range []struct {
		input, match, replacement string
	}{
		{"s/a/b/", "a", "b"},
		{"s/a//", "a", ""},
		{"s%a%b%", "a", "b"},
		{"s%a%%", "a", ""},
		{"s/ab/cd/", "ab", "cd"},
		{"s/a%b/c%d/", "a%b", "c%d"},
		{"s%ab%cd%", "ab", "cd"},
		{"s%a/b%c/d%", "a/b", "c/d"},
		{"s/a\\/b/c\\/d/", "a/b", "c/d"},
		{"s%a\\%b%c\\%d%", "a%b", "c%d"},
	} {
		repl, err := textutil.NewRewriteRule(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if got, want := repl.Match.String(), tc.match; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := repl.Replacement, tc.replacement; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
	for _, tc := range []string{
		"", "s/a/b", "s/a/", "s%a%b", "s%a%",
		"s/a/b/c", "s%a%b%c",
		"s/[^//",
	} {
		_, err := textutil.NewRewriteRule(tc)
		if err == nil {
			t.Errorf("%v: did not fail", tc)
			continue
		}
	}
}

func TestRewrite(t *testing.T) {
	repl, err := textutil.NewRewriteRule("s%(?:^s3://)(.*?)(/.*)%$1 -- $2%")
	if err != nil {
		t.Fatal(err)
	}
	out := repl.ReplaceAllString("s3://bucket/file/key.pdf")
	if got, want := out, "bucket -- /file/key.pdf"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRewrites(t *testing.T) {
	repls, err := textutil.NewRewriteRules("s/a/b/", "s/x/y/")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := repls.ReplaceAllStringFirst("a"), "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := repls.ReplaceAllStringFirst("x"), "y"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := repls.ReplaceAllStringFirst("n"), "n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	repls, err = textutil.NewRewriteRules("s/a/b/", "s/a/y/")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := repls.ReplaceAllStringFirst("a"), "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
