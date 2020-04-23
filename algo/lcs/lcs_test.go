package lcs_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"cloudeng.io/algo/lcs"
)

func TestLCS(t *testing.T) {
	l := func(s ...string) []string {
		return s
	}
	tostr := func(paths [][]int32) []string {
		o := []string{}
		for _, p := range paths {
			o = append(o, string(p))
		}
		return o
	}
	for i, tc := range []struct {
		a, b string
		lcs  []string
	}{
		{"GAC", "AGCAT", l("GA", "GA", "GC", "AC")},
		{"XMJYAUZ", "MZJAWXU", l("MJAU")},
		{"日本語", "日本de語", l("日本語")},
	} {
		differ := lcs.New([]byte(tc.a), []byte(tc.b), utf8.DecodeRune)
		if got, want := tostr(differ.Find()), tc.lcs; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
	/*
		l := lcs.New([]byte("GAC"), []byte("AGCAT"), utf8.DecodeRune)
		for _, l := range l.Find() {
			fmt.Printf("l %v\n", string(l))
		}
		l = lcs.New([]byte("XMJYAUZ"), []byte("MZJAWXU"), utf8.DecodeRune)
		for _, l := range l.Find() {
			fmt.Printf("l %v\n", string(l))
		}
		l = lcs.New([]byte("日本語"), []byte("日本de語"), utf8.DecodeRune)
		for _, l := range l.Find() {
			fmt.Printf("l %v\n", string(l))
		}*/
	t.Fail()
}
