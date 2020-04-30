package lcs_test

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"cloudeng.io/algo/codec"
	"cloudeng.io/algo/lcs"
)

func TestLCS(t *testing.T) {
	l := func(s ...string) []string {
		return s
	}
	tostr := func(paths ...[]int32) []string {
		o := []string{}
		for _, p := range paths {
			o = append(o, string(p))
		}
		return o
	}

	u32, err := codec.NewDecoder(utf8.DecodeRune)
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	u8, err := codec.NewDecoder(func(input []byte) (byte, int) {
		return input[0], 1
	})
	if err != nil {
		t.Fatalf("NewDecoder: %v", err)
	}

	_ = tostr
	for i, tc := range []struct {
		a, b string
		ses  int
		lcs  string
		all  []string
		edit string
	}{
		// Example from myer's 86 paper.
		{"ABCABBA", "CBABAC", 5, "BABA", l("BABA"), ""},
		// Wikipedia dynamic programming example.
		{"GAC", "AGCAT", 4, "AC", l("GA", "GA", "GC", "AC"), "-G =A +G =C +A +T"},

		// Longer examples.
		{"ABCADEFGH", "ABCIJKFGH", 6, "ABCFGH", l("ABCFGH"), ""},
		{"ABCDEF1234", "PQRST2UV4", 1, "24", l("24"), ""},

		// More exhaustive cases.
		{"", "", 0, "", l(""), ""},
		{"", "B", 0, "", l(""), ""},
		{"B", "", 0, "", l(""), ""},
		{"A", "A", 0, "A", l("A"), ""},
		{"AB", "AB", 0, "AB", l("AA"), ""},
		{"AB", "ABC", 1, "AB", l("AA"), ""},
		{"ABC", "AB", 1, "AB", l("AA"), ""},
		{"AC", "AXC", 1, "AC", l("AA"), ""},
		{"ABC", "ABX", 1, "AB", l("AA"), ""},
		{"ABC", "ABXY", 1, "AB", l("AA"), ""},
		{"ABXY", "AB", 1, "AB", l("AA"), ""},

		// rune and byte example where the results are identical.
		{"日本語", "日本de語", 2, "日本語", l("日本語"), "+日 +本 +d +e -日 -本 =語"},
	} {

		if i != 0 {
			continue
		}

		a, b := u32.Decode([]byte(tc.a)), u32.Decode([]byte(tc.b))
		myers := lcs.NewMyers(a, b)
		if got, want := string(myers.LCS().([]int32)), tc.lcs; got != want {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}

		dp := lcs.NewDP(a, b)
		if got, want := string(dp.LCS().([]int32)), tc.lcs; got != want {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}

		a, b = u8.Decode([]byte(tc.a)), u8.Decode([]byte(tc.b))
		myers = lcs.NewMyers(a, b)
		if got, want := myers.LCS().([]uint8), []byte(tc.lcs); !bytes.Equal(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}

	// Test case for correct utf8 handling.
	// a: 日本語
	// b: 日本語 with the middle byte of the middle rune changed.
	// A correct rune aware lcs will be 日語, whereas a byte based one will
	// include the 0xe6 first byte from the middle rune but skip the two
	// trailing bytes.
	ra := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x9c, 0xac, 0xe8, 0xaa, 0x9e}
	rb := []byte{0xe6, 0x97, 0xa5, 0xe6, 0x00, 0x00, 0xe8, 0xaa, 0x9e}
	a, b := u32.Decode(ra), u32.Decode(rb)
	myers := lcs.NewMyers(a, b)
	if got, want := string(myers.LCS().([]int32)), "日語"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	a, b = u8.Decode(ra), u8.Decode(rb)
	myers = lcs.NewMyers(a, b)
	if got, want := string(myers.LCS().([]byte)), "日\xe6語"; got != want {
		t.Errorf("got %#v, want %x %v", got, want, want)
	}

}
