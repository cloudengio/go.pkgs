package textutil_test

import (
	"bytes"
	"testing"

	"cloudeng.io/text/testing/testtext"
	"cloudeng.io/text/textutil"
)

func TestReverse(t *testing.T) {
	for i, tc := range []struct {
		input  string
		output []byte
	}{
		{"Hello 世界", []byte("界世 olleH")},
		{"世界文中", []byte("中文界世")},
		{"世界h文中", []byte("中文h界世")},
	} {
		cpy := textutil.ReverseBytes(tc.input)
		if got, want := len(cpy), len(tc.input); got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := len(cpy), len(tc.output); got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cpy, tc.output; !bytes.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	testStrings := genTestStrings()
	for _, tc := range testStrings {
		rv := textutil.ReverseString(tc)
		if tc == rv {
			t.Errorf("reverse failed for %v", tc)
		}
		rrv := textutil.ReverseString(rv)
		if tc != rrv {
			t.Errorf("reverse failed for %v", tc)
		}
	}
}

func genTestStrings() []string {
	tc := []string{}
	gen := testtext.NewRandom()
	strLen := 1033
	for _, s := range []int{1, 2, 3, 4} {
		tc = append(tc, gen.WithRuneLen(s, strLen))
	}
	return append(tc, gen.AllRuneLens(strLen))
}

func BenchmarkReverse(b *testing.B) {
	testStrings := genTestStrings()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testStrings {
			textutil.ReverseString(tc)
		}
	}
}

func BenchmarkReverseCopy(b *testing.B) {
	testStrings := genTestStrings()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testStrings {
			_ = string(textutil.ReverseBytes(tc))
		}
	}
}
