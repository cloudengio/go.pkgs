// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"bytes"
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

// Head returns the first n lines of b, where lines are delimited by sep. If
// b has fewer than n lines, it returns the entire slice. The returned slice
// aliases b.
func Head(b []byte, sep byte, n int) []byte {
	idx := 0
	for range n {
		next := bytes.IndexByte(b[idx:], sep)
		if next == -1 {
			return b
		}
		idx += next + 1
	}
	if idx == 0 {
		return b
	}
	return b[:idx-1]
}

// Tail returns the last n lines of b, where lines are delimited by sep. If b
// has fewer than n lines, it returns the entire slice. A trailing separator,
// if present, terminates the last line rather than introducing an additional
// empty line. The returned slice aliases b.
func Tail(b []byte, sep byte, n int) []byte {
	end := len(b)
	if end > 0 && b[end-1] == sep {
		end--
	}
	idx := end
	for range n {
		prev := bytes.LastIndexByte(b[:idx], sep)
		if prev == -1 {
			return b
		}
		idx = prev
	}
	if idx == end {
		return b
	}
	return b[idx+1 : end]
}

// HeadString returns the first n lines of the given string. If the string has
// fewer than n lines, it returns the entire string.
func HeadString(s string, n int) string {
	return BytesToString(Head(StringToBytes(s), '\n', n))
}

// TailString returns the last n lines of the given string. If the string has
// fewer than n lines, it returns the entire string. A trailing newline, if
// present, terminates the last line rather than introducing an additional
// empty line.
func TailString(s string, n int) string {
	return BytesToString(Tail(StringToBytes(s), '\n', n))
}
