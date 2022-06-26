package testtext_test

import (
	"testing"
	"unicode/utf8"

	"cloudeng.io/text/testing/testtext"
)

func TestFixedRuneLens(t *testing.T) {
	gen := testtext.NewRandom()
	strLen := 100
	for _, s := range []int{1, 2, 3, 4} {
		r := gen.WithRuneLen(s, strLen)
		if got, want := len(r), s*strLen; got != want {
			t.Errorf("%v: got %v, want: %v", s, got, want)
		}
		for i, c := range r {
			if got, want := utf8.RuneLen(c), s; got != want {
				t.Errorf("%v: '%c' @ %v: got %v, want: %v", s, c, i, got, want)
			}
		}
	}
}

func TestMixedRuneLens(t *testing.T) {
	gen := testtext.NewRandom()
	strLen := 100
	r := gen.AllRuneLens(strLen)
	sizes := map[int]bool{}
	for _, c := range r {
		sizes[utf8.RuneLen(c)] = true
	}
	if got, want := len(sizes), 4; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
