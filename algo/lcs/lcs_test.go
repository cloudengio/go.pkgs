// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package lcs_test

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"cloudeng.io/algo/codec"
	"cloudeng.io/algo/lcs"
	"cloudeng.io/errors"
)

func ExampleMyers() {
	runeDecoder := codec.NewDecoder(utf8.DecodeRune)
	a, b := runeDecoder.Decode([]byte("ABCABBA")), runeDecoder.Decode([]byte("CBABAC"))
	fmt.Printf("%v\n", string(lcs.NewMyers(a, b).LCS()))
	// Output:
	// BABA
}

func ExampleDP() {
	runeDecoder := codec.NewDecoder(utf8.DecodeRune)
	a, b := runeDecoder.Decode([]byte("AGCAT")), runeDecoder.Decode([]byte("GAC"))
	all := lcs.NewDP(a, b).AllLCS()
	for _, lcs := range all {
		fmt.Printf("%s\n", string(lcs))
	}
	// Output:
	// GA
	// GA
	// GC
	// AC
}

func isOneOf[T comparable](got []T, want [][]T) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	for _, w := range want {
		if reflect.DeepEqual(got, w) {
			return true
		}
	}
	return false
}

func lcsFromEdits[T comparable](script *lcs.EditScript[T]) interface{} {
	r := []T{}
	for _, op := range *script {
		if op.Op == lcs.Identical {
			r = append(r, op.Val)
		}
	}
	return r
}

func validateInsertions[T comparable](t *testing.T, i int, edits *lcs.EditScript[T], b []T) {
	for _, e := range *edits {
		if e.Op != lcs.Insert {
			continue
		}
		if got, want := e.Val, b[e.B]; got != want {
			t.Errorf("%v: %v: got %v, want %v", errors.Caller(2, 1), i, got, want)
		}
	}
}

func testLCSImpl[T comparable](t *testing.T, i int, lcs []T, edit *lcs.EditScript[T], a, b []T, all [][]T) {
	if got, want := lcs, all; !isOneOf(got, want) {
		t.Errorf("%v: got %v is not one of %v", i, got, want)
	}
	if got, want := lcsFromEdits(edit), lcs; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got %v, want %v", i, got, want)
	}

	// test edit string by recreating 'b' from 'a'.
	validateInsertions(t, i, edit, b)
	if got, want := edit.Apply(a), b; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got %v want %v for %v -> %v via %s", i, got, want, a, b, edit.String())
	}

	// and 'a' from 'b'
	reverse := edit.Reverse()
	validateInsertions(t, i, reverse, a)
	if got, want := reverse.Apply(b), a; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got %v want %v for %v -> %v via %s", i, got, want, b, a, edit.String())
	}
}

func TestLCS(t *testing.T) {

	l := func(s ...string) []string {
		if len(s) == 0 {
			return []string{}
		}
		return s
	}

	for i, tc := range []struct {
		a, b string
		all  []string
	}{
		// Example from myer's 1986 paper.
		{"ABCABBA", "CBABAC", l("BABA", "CABA", "CBBA")},

		// Wikipedia dynamic programming example.
		{"AGCAT", "GAC", l("AC", "GA", "GC")},
		{"XMJYAUZ", "MZJAWXU", l("MJAU")},

		// Longer examples.
		{"ABCADEFGH", "ABCIJKFGH", l("ABCFGH")},
		{"ABCDEF1234", "PQRST2UV4", l("24")},
		{"SABCDE", "SC", l("SC")},
		{"SABCDE", "SSC", l("SC")},

		// More exhaustive cases.
		{"", "", l()},
		{"", "B", l()},
		{"B", "", l()},
		{"A", "A", l("A")},
		{"AB", "AB", l("AB")},
		{"AB", "ABC", l("AB")},
		{"ABC", "AB", l("AB")},
		{"AC", "AXC", l("AC")},
		{"ABC", "ABX", l("AB")},
		{"ABC", "ABXY", l("AB")},
		{"ABXY", "AB", l("AB")},

		// Example where rune and byte results are identical.
		{"日本語", "日本de語", l("日本語")},
	} {
		runeDecoder := codec.NewDecoder(utf8.DecodeRune)

		allRunes := make([][]rune, len(tc.all))
		for i := range tc.all {
			allRunes[i] = []rune(tc.all[i])
		}

		sa, sb := runeDecoder.Decode([]byte(tc.a)), runeDecoder.Decode([]byte(tc.b))
		smyers := lcs.NewMyers(sa, sb)

		testLCSImpl(t, i, smyers.LCS(), smyers.SES(), sa, sb, allRunes)

		sdp := lcs.NewDP(sa, sb)
		testLCSImpl(t, i, sdp.LCS(), sdp.SES(), sa, sb, allRunes)

		byteDecoder := codec.NewDecoder(func(input []byte) (byte, int) {
			return input[0], 1
		})

		allBytes := make([][]byte, len(tc.all))
		for i := range tc.all {
			allBytes[i] = []byte(tc.all[i])
		}

		ba, bb := byteDecoder.Decode([]byte(tc.a)), byteDecoder.Decode([]byte(tc.b))
		bmyers := lcs.NewMyers(ba, bb)
		testLCSImpl(t, i, bmyers.LCS(), bmyers.SES(), ba, bb, allBytes)

		bdp := lcs.NewDP(ba, bb)
		testLCSImpl(t, i, bdp.LCS(), bdp.SES(), ba, bb, allBytes)
	}
}

func TestUTF8(t *testing.T) {
	i32, u8 := codec.NewDecoder(utf8.DecodeRune), codec.NewDecoder(func(input []byte) (byte, int) {
		return input[0], 1
	})

	// Test case for correct utf8 handling.
	// a: 日本語
	// b: 日本語 with the middle byte of the middle rune changed.
	// A correct rune aware lcs will be 日語, whereas a byte based one will
	// include the 0xe6 first byte from the middle rune but skip the two
	// trailing bytes.
	ra := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x9c, 0xac, 0xe8, 0xaa, 0x9e}
	rb := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x00, 0x00, 0xe8, 0xaa, 0x9e}
	sa, sb := i32.Decode(ra), i32.Decode(rb)
	if got, want := string(lcs.NewMyers(sa, sb).LCS()), "日語"; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	ba, bb := u8.Decode(ra), u8.Decode(rb)
	if got, want := string(lcs.NewMyers(ba, bb).LCS()), "日\xe6語"; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %x %v", got, want, want)
	}

	for i, tc := range []struct {
		a, b   string
		output string
	}{
		{"ABCABBA", "CBABAC", " CB AB AC\n-+|-||-|+\nA  C  B  \n"},
		{"AGCAT", "GAC", " G A C\n-|-|-+\nA C T \n"},
		{"XMJYAUZ", "MZJAWXU", " MZJ AWXU \n-|+|-|++|-\nX   Y    Z\n"},
	} {
		a, b := i32.Decode([]byte(tc.a)), i32.Decode([]byte(tc.b))
		myers := lcs.NewMyers(a, b)
		edit := myers.SES()
		out := &strings.Builder{}
		edit.FormatHorizontal(out, a)
		if got, want := out.String(), tc.output; got != want {
			t.Errorf("%v: got\n%v, want\n%v", i, got, want)
		}
	}
}

func TestLines(t *testing.T) {
	la := `
line1 a b c
line2 d e f
line3 hello
world
`
	lb := `
line2 d e f
hello
world
`
	lines := map[uint64]string{}
	lineDecoder := func(data []byte) (int64, int) {
		idx := bytes.Index(data, []byte{'\n'})
		if idx <= 0 {
			return 0, 1
		}
		h := fnv.New64a()
		h.Write(data[:idx])
		sum := h.Sum64()
		lines[sum] = string(data[:idx])
		return int64(sum), idx + 1
	}

	ld := codec.NewDecoder(lineDecoder)

	a, b := ld.Decode([]byte(la)), ld.Decode([]byte(lb))
	myers := lcs.NewMyers(a, b)
	edits := myers.SES()
	validateInsertions(t, 0, edits, b)

	var reconstructed string
	for _, op := range *edits {
		switch op.Op {
		case lcs.Identical:
			reconstructed += lines[uint64(a[op.A])] + "\n"
		case lcs.Insert:
			reconstructed += lines[uint64(op.Val)] + "\n"
		}
	}
	if got, want := reconstructed, lb; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out := &strings.Builder{}
	edits.FormatVertical(out, a)
	if got, want := out.String(), `                     0
-  6864772235558415538
  -8997218578518345818
+ -6615550055289275125
- -7192184552745107772
   5717881983045765875
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
