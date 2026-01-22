// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil_test

import (
	"testing"

	"cloudeng.io/text/textutil"
)

func TestTrimUnicodeQuotes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "double_curly_quotes",
			input: "“hello”",
			want:  "hello",
		},
		{
			name:  "single_curly_quotes",
			input: "‘hello’",
			want:  "hello",
		},
		{
			name:  "guillemets_with_inner_quotes",
			input: "«“hello”»",
			want:  "hello",
		},
		{
			name:  "angle_quotes",
			input: "‹hello›",
			want:  "hello",
		},
		{
			name:  "leading_only",
			input: "“hello",
			want:  "hello",
		},
		{
			name:  "trailing_only",
			input: "hello”",
			want:  "hello",
		},
		{
			name:  "only_quotes",
			input: "«»",
			want:  "",
		},
		{
			name:  "ascii_quotes_untouched",
			input: "\"hello\"",
			want:  "\"hello\"",
		},
		{
			name:  "non_quote_punctuation_untouched",
			input: "(hello)",
			want:  "(hello)",
		},
		{
			name:  "quotes_inside_text",
			input: "h“ello «»“world”",
			want:  "h“ello «»“world",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := textutil.TrimUnicodeQuotes(tc.input); got != tc.want {
				t.Fatalf("%v: got %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
