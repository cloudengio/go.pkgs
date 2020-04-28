// Package lcs provides implementations of alogorithms to find the
// longest common subsequence/shortest edit script (LCS/SES) suitable for
// use with unicode/utf8 and other alphabets.
package lcs

type Solver interface {
	LCS() []int32
	SES() EditScript
	All() [][]int32
}

// Decoder is used to decode a byte slice into a slice of int32s. Decoding
// is required to allow for operation on utf8 data but also be used to
// represent lines or fields using a 32bit hash with sufficient collision
// performance.

type Decoder32 func([]byte) (int32, int)
type Decoder64 func([]byte) (int64, int)
type DecoderByte func([]byte) (int64, int)

func decode32(input []byte, blank int, decoder Decoder32) []int32 {
	li := len(input)
	cursor := 0
	tmp := make([]int32, li+blank)
	i := blank
	for {
		tok, n := decoder(input[cursor:])
		if n == 0 {
			continue
		}
		tmp[i] = tok
		i++
		cursor += n
		if cursor >= li {
			break
		}
	}
	return tmp[:i]
}
