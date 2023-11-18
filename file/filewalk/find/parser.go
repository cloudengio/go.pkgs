// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package find

import (
	"strings"

	"cloudeng.io/file/matcher"
)

// Parse parses the supplied input into a matcher.T.
// The supported syntax is a boolean expression with
// and (&&), or (||) and grouping, via ().
// The supported operands are:
//
//		name='glob-pattern'
//		iname='glob-pattern'
//		re='regexp'
//		type='f|d|l'
//		newer='date' in time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly
//
//	 Note that the single quotes are optional unless a white space is present
//	 in the pattern.
func Parse(input string) (matcher.T, error) {
	tokens := make(chan string, 100)

	for token := range tokens {
		switch {

		}

	}

	return nil, nil
}

func tokenize(input string, ch chan<- string) {
	var seen strings.Builder
	quoted := false
	for _, t := range input {
		if t == '=' {
			ch <- seen.String()
			seen.Reset()
		}
		seen.WriteRune(t)
		ch <- t
	}
	close(ch)
}
