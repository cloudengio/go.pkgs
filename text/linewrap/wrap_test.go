package linewrap_test

import (
	"testing"

	"cloudeng.io/text/linewrap"
)

const stringsText = `FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c) and returns an array of slices of s. If all code points in s satisfy f(c) or the string is empty, an empty slice is returned. FieldsFunc makes no guarantees about the order in which it calls f(c). If f does not return consistent results for a given c, FieldsFunc may crash.`

const wrappedStringsText = `  FieldsFunc splits the string s at each run of Unicode code points c satisfying
  f(c) and returns an array of slices of s. If all code points in s satisfy f(c) or
  the string is empty, an empty slice is returned. FieldsFunc makes no guarantees
  about the order in which it calls f(c). If f does not return consistent results for
  a given c, FieldsFunc may crash.`

func TestWrap(t *testing.T) {
	for i, tc := range []struct {
		input, output string
	}{
		{"aa", "  aa"},
		{"aa bb\n", "  aa bb"},
		{"aa bb ccc dddd eeee\n", "  aa bb ccc\n  dddd eeee"},
	} {
		if got, want := linewrap.SimpleWrap(2, 10, tc.input), tc.output; got != want {
			t.Errorf("%v: got \n%v, want \n%v\n", i, got, want)
		}
	}
	if got, want := linewrap.SimpleWrap(2, 78, stringsText), wrappedStringsText; got != want {
		t.Errorf("got \n%v, want \n%v\n", got, want)
	}
}
