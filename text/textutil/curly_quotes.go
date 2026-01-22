// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"strings"
	"unicode"
)

// TrimUnicodeQuotes trims leading and trailing UTF-8 curly quotes from text
// using unicode properties (Pi and Pf).
func TrimUnicodeQuotes(text string) string {
	text = strings.TrimLeftFunc(text, func(r rune) bool {
		// Punctuation, Initial quote (e.g. “, ‘, «)
		return unicode.Is(unicode.Pi, r)
	})
	return strings.TrimRightFunc(text, func(r rune) bool {
		// Punctuation, Final quote   (e.g. ”, ’, »)
		return unicode.Is(unicode.Pf, r)
	})
}
