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
		for i := 0; i < nb; i++ {
			reversed[lp+i] = runeBytes[i]
		}
	}
	return reversed
}

// ReverseString is like ReverseBytes but returns a string.
func ReverseString(input string) string {
	return BytesToString(ReverseBytes(input))
}
