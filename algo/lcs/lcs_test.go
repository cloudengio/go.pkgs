package lcs_test

import (
	"reflect"
	"sort"
	"testing"
	"unicode/utf8"

	"cloudeng.io/algo/codec"
	"cloudeng.io/algo/lcs"
)

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

func all32(lcs [][]int32) []string {
	dedup := map[string]bool{}
	for _, l := range lcs {
		dedup[string(l)] = true
	}
	str := []string{}
	for k := range dedup {
		str = append(str, k)
	}
	sort.Strings(str)
	return str
}

func all8(lcs [][]uint8) []string {
	dedup := map[string]bool{}
	for _, l := range lcs {
		dedup[string(l)] = true
	}
	str := []string{}
	for k := range dedup {
		str = append(str, k)
	}
	sort.Strings(str)
	return str
}

func TestLCS(t *testing.T) {
	l := func(s ...string) []string {
		if len(s) == 0 {
			return []string{}
		}
		return s
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

	for i, tc := range []struct {
		a, b string
		ses  int
		all  []string
	}{
		// Example from myer's 1986 paper.
		{"ABCABBA", "CBABAC", 5, l("BABA", "CABA", "CBBA")},
		// Wikipedia dynamic programming example.
		{"AGCAT", "GAC", 4, l("AC", "GA", "GC")},
		{"XMJYAUZ", "MZJAWXU", 4, l("MJAU")},
		// Longer examples.
		{"ABCADEFGH", "ABCIJKFGH", 6, l("ABCFGH")},
		{"ABCDEF1234", "PQRST2UV4", 1, l("24")},

		// More exhaustive cases.
		{"", "", 0, l()},
		{"", "B", 0, l()},
		{"B", "", 0, l()},
		{"A", "A", 0, l("A")},
		{"AB", "AB", 0, l("AB")},
		{"AB", "ABC", 1, l("AB")},
		{"ABC", "AB", 1, l("AB")},
		{"AC", "AXC", 1, l("AC")},
		{"ABC", "ABX", 1, l("AB")},
		{"ABC", "ABXY", 1, l("AB")},
		{"ABXY", "AB", 1, l("AB")},

		// rune and byte example where the results are identical.
		{"日本語", "日本de語", 2, l("日本語")},
	} {

		a, b := u32.Decode([]byte(tc.a)), u32.Decode([]byte(tc.b))
		myers := lcs.NewMyers(a, b)
		lcs32 := myers.LCS().([]int32)
		if got, want := string(lcs32), tc.all; !isOneOf(got, want) {
			t.Errorf("%v: got %v is not one of %v", i, got, want)
		}

		// test edit string by recreating 'b' from 'a'.
		edit := myers.SES()
		if got, want := string(edit.Apply(a).([]int32)), string(b.([]int32)); got != want {
			t.Errorf("%v: got %v want %v for %s -> %s via %s", i, got, want, string(a.([]int32)), string(b.([]int32)), edit.String())
		}

		dp := lcs.NewDP(a, b)
		lcs32 = dp.LCS().([]int32)
		if got, want := string(lcs32), tc.all; !isOneOf(got, want) {
			t.Errorf("%v: got %v is not one of %v", i, got, want)
		}
		if got, want := all32(dp.AllLCS().([][]int32)), tc.all; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}

		// test edit string by recreating 'b' from 'a'.
		edit = dp.SES()
		if got, want := string(edit.Apply(a).([]int32)), string(b.([]int32)); got != want {
			t.Errorf("%v: got %v, want %v for %s -> %s", i, got, want, string(a.([]int32)), edit.String())
		}

		a, b = u8.Decode([]byte(tc.a)), u8.Decode([]byte(tc.b))
		myers = lcs.NewMyers(a, b)
		lcs8 := myers.LCS().([]uint8)
		if got, want := string(lcs8), tc.all; !isOneOf(got, want) {
			t.Errorf("%v: got %v is not one of %v", i, got, want)
		}

		dp = lcs.NewDP(a, b)
		lcs8 = dp.LCS().([]uint8)
		if got, want := string(lcs8), tc.all; !isOneOf(got, want) {
			t.Errorf("%v: got %v is not one of %v", i, got, want)
		}

		if got, want := all8(dp.AllLCS().([][]uint8)), tc.all; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}

		/*
			if got, want := myers.LCS().([]uint8), []byte(tc.lcs); !bytes.Equal(got, want) {
				t.Errorf("%v: got %#v, want %#v", i, got, want)
			}*/

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
