// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import "unicode/utf8"

// ReverseBytes returns a new slice containing the runes in the input string
// in reverse order.
func ReverseBytes(input string) []byte {
	lp := len(input)
	reversed := make([]byte, lp)
	runeBytes := []byte{0x0, 0x0, 0x0, 0x0}
	for _, r := range input {
		nb := utf8.EncodeRune(runeBytes, r)
		lp -= nb
		for i := range nb {
			reversed[lp+i] = runeBytes[i]
		}
	}
	return reversed
}

// ReverseString is like ReverseBytes but returns a string.
func ReverseString(input string) string {
	return BytesToString(ReverseBytes(input))
}
