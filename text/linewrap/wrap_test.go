// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package linewrap_test

import (
	"testing"

	"cloudeng.io/text/linewrap"
)

const stringsText = `FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c) and returns an array of slices of s. If all code points in s satisfy f(c) or the string is empty, an empty slice is returned. FieldsFunc makes no guarantees about the order in which it calls f(c). If f does not return consistent results for a given c, FieldsFunc may crash.`

const multiParagraphText = `FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c) and returns an array of slices of s. If all code points in s satisfy f(c) or the string is empty, an empty slice is returned. FieldsFunc makes no guarantees about the order in which it calls f(c). If f does not return consistent results for a given c, FieldsFunc may crash.

FieldsFunc splits the string s at each run of Unicode code points c satisfying f(c) and returns an array of slices of s. If all code points in s satisfy f(c) or the string is empty, an empty slice is returned. FieldsFunc makes no guarantees about the order in which it calls f(c). If f does not return consistent results for a given c, FieldsFunc may crash.
`

const blockStringsText = `    FieldsFunc splits the string s at each run of Unicode code points c satisfying
    f(c) and returns an array of slices of s. If all code points in s satisfy
    f(c) or the string is empty, an empty slice is returned. FieldsFunc makes
    no guarantees about the order in which it calls f(c). If f does not return
    consistent results for a given c, FieldsFunc may crash.`

const paragraphStringsText = `  FieldsFunc splits the string s at each run of Unicode code points c satisfying
    f(c) and returns an array of slices of s. If all code points in s satisfy
    f(c) or the string is empty, an empty slice is returned. FieldsFunc makes
    no guarantees about the order in which it calls f(c). If f does not return
    consistent results for a given c, FieldsFunc may crash.`

const essayStringsText = `    FieldsFunc splits the string s at each run of Unicode code points c satisfying
  f(c) and returns an array of slices of s. If all code points in s satisfy
  f(c) or the string is empty, an empty slice is returned. FieldsFunc makes no
  guarantees about the order in which it calls f(c). If f does not return
  consistent results for a given c, FieldsFunc may crash.`

const commentStringsText = `  // FieldsFunc splits the string s at each run of Unicode code points c
  // satisfying f(c) and returns an array of slices of s. If all code points
  // in s satisfy f(c) or the string is empty, an empty slice is returned.
  // FieldsFunc makes no guarantees about the order in which it calls f(c). If
  // f does not return consistent results for a given c, FieldsFunc may crash.`

func TestWrap(t *testing.T) {
	for i, tc := range []struct {
		input, output string
	}{
		{"aa", "  aa"},
		{"aa bb ccc\n", "  aa bb ccc"},
		{"aa bb cc d e f\n", "  aa bb cc\n  d e f"},
		{"aa bb ccc d e f\n", "  aa bb ccc\n  d e f"},
		{"aa bb ccc d e f hello world again\n", "  aa bb ccc\n  d e f\n  hello\n  world\n  again"},
	} {
		if got, want := linewrap.Block(2, 10, tc.input), tc.output; got != want {
			t.Errorf("%v: got \n%v, want \n%v\n", i, got, want)
		}
	}
	for i, tc := range []struct {
		input, output string
	}{
		{"aa", " aa"},
		{"aa bb ccc\n", " aa bb ccc"},
		{"aa bb cc d e f\n", " aa bb cc\n  d e f"},
		{"aa bb ccc d e f\n", " aa bb ccc\n  d e f"},
		{"aa bb ccc d e f\n", " aa bb ccc\n  d e f"},
		{"aa bb ccc d e f hello world again\n", " aa bb ccc\n  d e f\n  hello\n  world\n  again"},
	} {
		if got, want := linewrap.Paragraph(1, 2, 10, tc.input), tc.output; got != want {
			t.Errorf("%v: got \n%v, want \n%v\n", i, got, want)
		}
	}

	if got, want := linewrap.Block(4, 78, stringsText), blockStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

	if got, want := linewrap.Paragraph(2, 4, 78, stringsText), paragraphStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

	if got, want := linewrap.Paragraph(4, 2, 78, stringsText), essayStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

	if got, want := linewrap.Comment(2, 78, "// ", stringsText), commentStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

	if got, want := linewrap.Block(4, 78, multiParagraphText), blockStringsText+"\n\n"+blockStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

	if got, want := linewrap.Comment(2, 78, "// ", multiParagraphText), commentStringsText+"\n  //\n"+commentStringsText; got != want {
		t.Errorf("got \n>\n%v, want \n%v\n", got, want)
	}

}
