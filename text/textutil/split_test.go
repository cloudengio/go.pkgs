// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil_test

import (
	"slices"
	"strings"
	"testing"

	"cloudeng.io/text/textutil"
)

func TestStringSplitIterator(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		sep   rune
	}{
		{"empty", "", '/'},
		{"simple", "a/b/c", '/'},
		{"no sep", "abc", '/'},
		{"leading sep", "/a/b/c", '/'},
		{"trailing sep", "a/b/c/", '/'},
		{"consecutive sep", "a//b", '/'},
		{"multiple sep", "a/b/c/d/e", '/'},
		{"unicode", "a/b/c", 'b'},
		{"multi-byte sep", "a⌘b⌘c", '⌘'},
		{"only seps", "///", '/'},
		{"sep at start", "⌘a", '⌘'},
		{"sep at end", "a⌘", '⌘'},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			var last int
			for i, s := range textutil.SplitString(tc.input, tc.sep) {
				if i != last {
					t.Errorf("StringSplitIterator(%q, %q) index %d != %d", tc.input, tc.sep, i, last)
				}
				got = append(got, s)
				last = i + 1
			}
			expected := strings.Split(tc.input, string(tc.sep))
			if got, want := got, expected; !slices.Equal(got, want) {
				t.Errorf("StringSplitIterator(%q, %q) = %v, want strings.Split(%q, %q) = %v", tc.input, tc.sep, got, tc.input, tc.sep, want)
			}
		})
	}
}

func TestHeadString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{"empty", "", 2, ""},
		{"zero lines requested", "a\nb\nc", 0, "a\nb\nc"},
		{"fewer lines than n", "a\nb", 5, "a\nb"},
		{"fewer lines than n, trailing newline", "a\nb\n", 5, "a\nb\n"},
		{"exact line count, no trailing newline", "a\nb\nc", 3, "a\nb\nc"},
		{"first line", "a\nb\nc", 1, "a"},
		{"first two lines", "a\nb\nc", 2, "a\nb"},
		{"trailing newline", "a\nb\nc\n", 2, "a\nb"},
		{"single line, no newline", "hello", 1, "hello"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := textutil.HeadString(tc.input, tc.n); got != tc.want {
				t.Errorf("HeadString(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
			}
		})
	}
}

func TestTailString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{"empty", "", 2, ""},
		{"zero lines requested", "a\nb\nc", 0, "a\nb\nc"},
		{"fewer lines than n", "a\nb", 5, "a\nb"},
		{"fewer lines than n, trailing newline", "a\nb\n", 5, "a\nb\n"},
		{"exact line count, no trailing newline", "a\nb\nc", 3, "a\nb\nc"},
		{"exact line count, trailing newline", "a\nb\nc\n", 3, "a\nb\nc\n"},
		{"last line", "a\nb\nc", 1, "c"},
		{"last two lines", "a\nb\nc", 2, "b\nc"},
		{"trailing newline", "a\nb\nc\n", 2, "b\nc"},
		{"single line, no newline", "hello", 1, "hello"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := textutil.TailString(tc.input, tc.n); got != tc.want {
				t.Errorf("TailString(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.want)
			}
		})
	}
}

func TestHead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		sep   byte
		n     int
		want  string
	}{
		{"empty", "", '\n', 2, ""},
		{"zero lines requested", "a\nb\nc", '\n', 0, "a\nb\nc"},
		{"fewer lines than n", "a\nb", '\n', 5, "a\nb"},
		{"exact line count, no trailing sep", "a\nb\nc", '\n', 3, "a\nb\nc"},
		{"first two lines", "a\nb\nc", '\n', 2, "a\nb"},
		{"trailing sep", "a\nb\nc\n", '\n', 2, "a\nb"},
		{"single line, no sep", "hello", '\n', 1, "hello"},
		{"custom separator", "a,b,c", ',', 2, "a,b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := textutil.Head([]byte(tc.input), tc.sep, tc.n)
			if string(got) != tc.want {
				t.Errorf("Head(%q, %q, %d) = %q, want %q", tc.input, tc.sep, tc.n, got, tc.want)
			}
		})
	}
}

func TestTail(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		sep   byte
		n     int
		want  string
	}{
		{"empty", "", '\n', 2, ""},
		{"zero lines requested", "a\nb\nc", '\n', 0, "a\nb\nc"},
		{"fewer lines than n", "a\nb", '\n', 5, "a\nb"},
		{"exact line count, no trailing sep", "a\nb\nc", '\n', 3, "a\nb\nc"},
		{"exact line count, trailing sep", "a\nb\nc\n", '\n', 3, "a\nb\nc\n"},
		{"last two lines", "a\nb\nc", '\n', 2, "b\nc"},
		{"trailing sep", "a\nb\nc\n", '\n', 2, "b\nc"},
		{"single line, no sep", "hello", '\n', 1, "hello"},
		{"custom separator", "a,b,c", ',', 2, "b,c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := textutil.Tail([]byte(tc.input), tc.sep, tc.n)
			if string(got) != tc.want {
				t.Errorf("Tail(%q, %q, %d) = %q, want %q", tc.input, tc.sep, tc.n, got, tc.want)
			}
		})
	}
}
