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
	runeDecoder, _ := codec.NewDecoder(utf8.DecodeRune)
	a, b := runeDecoder.Decode([]byte("ABCABBA")), runeDecoder.Decode([]byte("CBABAC"))
	fmt.Printf("%s\n", string(lcs.NewMyers(a, b).LCS().([]int32)))
	// Output:
	// BABA
}

func ExampleDP() {
	runeDecoder, _ := codec.NewDecoder(utf8.DecodeRune)
	a, b := runeDecoder.Decode([]byte("AGCAT")), runeDecoder.Decode([]byte("GAC"))
	all := lcs.NewDP(a, b).AllLCS().([][]int32)
	for _, lcs := range all {
		fmt.Printf("%s\n", string(lcs))
	}
	// Output:
	// GA
	// GA
	// GC
	// AC
}

func isOneOf(got string, want []string) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	for _, w := range want {
		if got == w {
			return true
		}
	}
	return false
}

func lcsFromEdits(typ interface{}, script lcs.EditScript) interface{} {
	switch typ.(type) {
	case int64:
		r := []int64{}
		for _, op := range script {
			if op.Op == lcs.Identical {
				r = append(r, op.Val.(int64))
			}
		}
		return r
	case int32:
		r := []int32{}
		for _, op := range script {
			if op.Op == lcs.Identical {
				r = append(r, op.Val.(int32))
			}
		}
		return r
	case uint8:
		r := []uint8{}
		for _, op := range script {
			if op.Op == lcs.Identical {
				r = append(r, op.Val.(uint8))
			}
		}
		return r
	}
	panic(fmt.Sprintf("unsupported type %T", typ))
}

func validateInsertions(t *testing.T, i int, edits lcs.EditScript, b interface{}) {
	for _, e := range edits {
		if e.Op != lcs.Insert {
			continue
		}
		switch v := e.Val.(type) {
		case int64:
			if got, want := v, b.([]int64)[e.B]; got != want {
				t.Errorf("%v: %v: got %v, want %v", errors.Caller(2, 1), i, got, want)
			}
		case int32:
			if got, want := v, b.([]int32)[e.B]; got != want {
				t.Errorf("%v: %v: got %c, want %c", errors.Caller(2, 1), i, got, want)
			}
		case uint8:
			if got, want := v, b.([]uint8)[e.B]; got != want {
				t.Errorf("%v: %v: got %c, want %c", errors.Caller(2, 1), i, got, want)
			}
		}
	}
}

func decoders(t *testing.T) (i32, u8 codec.Decoder) {
	i32, err := codec.NewDecoder(utf8.DecodeRune)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	u8, err = codec.NewDecoder(func(input []byte) (byte, int) {
		return input[0], 1
	})
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}
	return
}

type implementation interface {
	LCS() interface{}
	SES() lcs.EditScript
}

func testutf8(t *testing.T, impl implementation, i int, a, b []int32, all []string) {
	lcs32 := impl.LCS().([]int32)
	if got, want := string(lcs32), all; !isOneOf(got, want) {
		t.Errorf("%v: got %v is not one of %v", i, got, want)
	}

	edit := impl.SES()
	if got, want := lcsFromEdits(int32(0), edit).([]int32), lcs32; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got %v, want %v", i, string(got), string(want))
	}

	// test edit string by recreating 'b' from 'a'.
	validateInsertions(t, i, edit, b)
	if got, want := string(edit.Apply(a).([]int32)), string(b); got != want {
		t.Errorf("%v: got %v want %v for %s -> %s via %s", i, got, want, string(a), string(b), edit.String())
	}

	// and 'a' from 'b'
	reverse := lcs.Reverse(edit)
	validateInsertions(t, i, reverse, a)
	if got, want := string(reverse.Apply(b).([]int32)), string(a); got != want {
		t.Errorf("%v: got %v want %v for %s -> %s via %s", i, got, want, string(b), string(a), edit.String())
	}
}

func testbyte(t *testing.T, impl implementation, i int, a, b []uint8, all []string) {
	lcs32 := impl.LCS().([]uint8)
	if got, want := string(lcs32), all; !isOneOf(got, want) {
		t.Errorf("%v: got %v is not one of %v", i, got, want)
	}

	// test edit string by recreating 'b' from 'a'.
	edit := impl.SES()
	if got, want := lcsFromEdits(uint8(0), edit).([]uint8), lcs32; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got %v, want %v", i, string(got), string(want))
	}
	validateInsertions(t, i, edit, b)
	if got, want := string(edit.Apply(a).([]uint8)), string(b); got != want {
		t.Errorf("%v: got %v want %v for %s -> %s via %s", i, got, want, string(a), string(b), edit.String())
	}

	// and 'a' from 'b'
	reverse := lcs.Reverse(edit)
	validateInsertions(t, i, reverse, a)
	if got, want := string(reverse.Apply(b).([]uint8)), string(a); got != want {
		t.Errorf("%v: got %v want %v for %s -> %s via %s", i, got, want, string(b), string(a), edit.String())
	}
}

func TestLCS(t *testing.T) {
	l := func(s ...string) []string {
		if len(s) == 0 {
			return []string{}
		}
		return s
	}
	i32, u8 := decoders(t)

	for i, tc := range []struct {
		a, b string

		all []string
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
		a, b := i32.Decode([]byte(tc.a)), i32.Decode([]byte(tc.b))
		myers := lcs.NewMyers(a, b)
		testutf8(t, myers, i, a.([]int32), b.([]int32), tc.all)

		dp := lcs.NewDP(a, b)
		testutf8(t, dp, i, a.([]int32), b.([]int32), tc.all)

		a, b = u8.Decode([]byte(tc.a)), u8.Decode([]byte(tc.b))
		myers = lcs.NewMyers(a, b)
		testbyte(t, myers, i, a.([]uint8), b.([]uint8), tc.all)

		dp = lcs.NewDP(a, b)
		testbyte(t, dp, i, a.([]uint8), b.([]uint8), tc.all)

	}

}

func TestUTF8(t *testing.T) {
	i32, u8 := decoders(t)
	// Test case for correct utf8 handling.
	// a: 日本語
	// b: 日本語 with the middle byte of the middle rune changed.
	// A correct rune aware lcs will be 日語, whereas a byte based one will
	// include the 0xe6 first byte from the middle rune but skip the two
	// trailing bytes.
	ra := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x9c, 0xac, 0xe8, 0xaa, 0x9e}
	rb := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x00, 0x00, 0xe8, 0xaa, 0x9e}
	a, b := i32.Decode(ra), i32.Decode(rb)
	myers := lcs.NewMyers(a, b)
	if got, want := string(myers.LCS().([]int32)), "日語"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	a, b = u8.Decode(ra), u8.Decode(rb)
	myers = lcs.NewMyers(a, b)
	if got, want := string(myers.LCS().([]byte)), "日\xe6語"; got != want {
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
		lcs.FormatHorizontal(out, a, edit)
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

	ld, err := codec.NewDecoder(lineDecoder)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	a, b := ld.Decode([]byte(la)), ld.Decode([]byte(lb))
	myers := lcs.NewMyers(a, b)
	edits := myers.SES()
	validateInsertions(t, 0, edits, b)

	var reconstructed string
	for _, op := range edits {
		switch op.Op {
		case lcs.Identical:
			reconstructed += lines[uint64(a.([]int64)[op.A])] + "\n"
		case lcs.Insert:
			reconstructed += lines[uint64(op.Val.(int64))] + "\n"
		}
	}
	if got, want := reconstructed, lb; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out := &strings.Builder{}
	lcs.FormatVertical(out, a, edits)
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
