// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"unicode"
	"unicode/utf8"
)

// TrimUnicodeQuotes trims leading and trailing UTF-8 curly quotes from text
// using unicode properties (Pi and Pf).
func TrimUnicodeQuotes(text string) string {
	start := 0
	for start < len(text) {
		r, size := utf8.DecodeRuneInString(text[start:])
		if !unicode.Is(unicode.Pi, r) {
			break
		}
		start += size
	}
	end := len(text)
	for end > start {
		r, size := utf8.DecodeLastRuneInString(text[:end])
		if !unicode.Is(unicode.Pf, r) {
			break
		}
		end -= size
	}
	return text[start:end]
}
