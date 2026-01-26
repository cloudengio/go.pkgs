// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"iter"
	"strings"
	"unicode/utf8"
)

// SplitString splits a string into components separated by the given rune.
// It returns an iterator that yields the components in order and the 0-based
// index of the component in the string.
// It is functionally equivalent to strings.Split but returns an iterator
// instead of a slice, i.e. create a slice by iterating over SplitString
// and appending the components to the slice is identical to the output of
// strings.Split.
//
//	var expected []string
//	for i, s := range SplitString(input, sep) {
//	  expected = append(expected, s)
//	}
//	if !slices.Equal(expected, strings.Split(input, string(sep))) {
//	  t.Errorf("SplitString(%q, %q) = %v, want %v", input, sep, expected, strings.Split(input, string(sep)))
//	}
func SplitString(s string, sep rune) iter.Seq2[int, string] {
	remaining := s
	seplen := utf8.RuneLen(sep)
	return func(yield func(int, string) bool) {
		i := 0
		for {
			idx := strings.IndexRune(remaining, sep)
			if idx == -1 {
				yield(i, remaining)
				return
			}
			component := remaining[:idx]
			remaining = remaining[idx+seplen:]
			if !yield(i, component) {
				return
			}
			i++
		}
	}
}
