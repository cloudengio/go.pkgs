package lcs_test

import (
	"fmt"
	"testing"
	"unicode/utf8"

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
	_, _ = tostr, l
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

		// More exhaustive cases.
		{"", "", 0, "", l(""), ""},
		{"A", "A", 0, "A", l("A"), ""},
		{"AA", "AA", 0, "AA", l("AA"), ""},
		{"AA", "AAA", 1, "AA", l("AA"), ""},
		{"AAA", "AA", 1, "AA", l("AA"), ""},
		{"AA", "AXA", 1, "AA", l("AA"), ""},
		{"AAA", "AAX", 1, "AA", l("AA"), ""},
		{"AAA", "AAXY", 1, "AA", l("AA"), ""},
		{"AAXY", "AA", 1, "AA", l("AA"), ""},

		/*{"ABCABBA", "XXXX", 11, "", l(""), ""},

		{"XMJYAUZ", "MZJAWXU", 6, "MJAU", l("MJAU"), "-X -M -J -Y -A -U +M =Z +J +A +W +X +U"},
		{"ABC", "ABxyC", 2, "ABC", l("ABC"), "+日 +本 +d +e -日 -本 =語"},

		{"日本語", "日本de語", 2, "日本語", l("日本語"), "+日 +本 +d +e -日 -本 =語"},
		{"123", "23", 1, "23", l("23"), "-1 =2 =3"},
		{"23", "123", 1, "23", l("23"), "+1 =2 =3"},
		{"23", "253", 1, "23", l("23"), "+2 +5 -2 =3"},
		{"233", "253", 2, "23", l("23", "x", "x"), "-2 -3 +2 +5 =3"},
		{"A", "A", 0, "A", []string{"x"}, ""},
		{"", "", 0, "", []string{"x"}, ""},*/
	} {
		/*differ := lcs.New([]byte(tc.a), []byte(tc.b), utf8.DecodeRune)
		if got, want := tostr(differ.Find()...), tc.lcs; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := differ.Diff().String(), tc.edit; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}*/

		myers := lcs.NewMyers([]byte(tc.a), []byte(tc.b), utf8.DecodeRune)
		if got, want := string(myers.LCS()), tc.lcs; got != want {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		fmt.Printf("+++ %v\n", i)
	}
}
