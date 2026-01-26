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
