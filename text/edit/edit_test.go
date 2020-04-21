// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package edit_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/text/edit"
)

func ExampleDo() {
	content := "world"
	helloWorld := edit.DoString(content, edit.InsertString(0, "hello "))
	bonjourWorld := edit.DoString(helloWorld, edit.ReplaceString(0, 5, "bonjour"))
	hello := edit.DoString(helloWorld, edit.Delete(5, 6))
	sentence := edit.DoString("some random things",
		edit.ReplaceString(0, 1, "S"),
		edit.ReplaceString(5, 6, "thoughts"),
		edit.Delete(12, 6),
		edit.InsertString(18, "for the day."))
	fmt.Println(helloWorld)
	fmt.Println(bonjourWorld)
	fmt.Println(hello)
	fmt.Println(sentence)
	// Output:
	// hello world
	// bonjour world
	// hello
	// Some thoughts for the day.
}

func TestString(t *testing.T) {
	for i, tc := range []struct {
		op   edit.Delta
		text string
	}{
		{edit.InsertString(0, "xy"), "> @0#2"},
		{edit.InsertString(32, "xyyz"), "> @32#4"},
		{edit.Delete(0, 1), "< @0#1"},
		{edit.Delete(32, 10), "< @32#10"},
		{edit.ReplaceString(0, 2, "xy"), "~ @0#2/2"},
	} {
		if got, want := tc.op.String(), tc.text; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}

func TestEdits(t *testing.T) {
	ins := edit.InsertString
	rpl := edit.ReplaceString
	del := edit.Delete
	j := func(ds ...edit.Delta) []edit.Delta {
		return ds
	}
	for i, tc := range []struct {
		contents []byte
		deltas   []edit.Delta
		edited   []byte
	}{
		// insertion test cases
		{[]byte{}, j(ins(0, "ab")), []byte("ab")},
		{[]byte{}, j(ins(0, "ab"), ins(0, "cd")), []byte("abcd")},
		{[]byte{}, j(ins(0, "cd"), ins(0, "ab")), []byte("cdab")},

		{[]byte("ab"), j(ins(0, "cd")), []byte("cdab")},
		{[]byte("ab"), j(ins(1, "cd")), []byte("acdb")},
		{[]byte("ab"), j(ins(2, "cd")), []byte("abcd")},

		{[]byte("ab"), j(ins(0, "cd"), ins(0, "xy")), []byte("cdxyab")},
		{[]byte("ab"), j(ins(1, "cd"), ins(1, "xy")), []byte("acdxyb")},
		{[]byte("ab"), j(ins(2, "cd"), ins(2, "xy")), []byte("abcdxy")},

		{[]byte("ab"), j(ins(0, "cd"), ins(1, "xy")), []byte("cdaxyb")},
		{[]byte("ab"), j(ins(1, "cd"), ins(1, "xy")), []byte("acdxyb")},
		{[]byte("ab"), j(ins(2, "cd"), ins(1, "xy")), []byte("axybcd")},

		// deletions
		{[]byte("ab"), j(del(0, 2)), []byte{}},
		{[]byte("ab"), j(del(0, 1)), []byte("b")},
		{[]byte("ab"), j(del(1, 1)), []byte("a")},
		{[]byte("abcde"), j(del(2, 2)), []byte("abe")},
		{[]byte("abcde"), j(del(1, 1), del(1, 2), del(1, 3)), []byte("ae")},
		{[]byte("ab"), j(del(0, 1), del(0, 1)), []byte("b")},
		{[]byte("ab"), j(del(0, 1), del(1, 1)), []byte{}},
		{[]byte("ab"), j(del(1, 1), del(0, 1)), []byte{}},

		// replacements
		{[]byte("a"), j(rpl(0, 1, "A")), []byte("A")},
		{[]byte("ab"), j(rpl(0, 1, "A"), rpl(1, 1, "B")), []byte("AB")},
		{[]byte("abcd"), j(rpl(0, 4, "xy")), []byte("xy")},
		{[]byte("abcd"), j(rpl(0, 2, "xy")), []byte("xycd")},
		{[]byte("abcd"), j(rpl(0, 1, "xy")), []byte("xybcd")},
		{[]byte("abcd"), j(rpl(1, 3, "xy")), []byte("axy")},
		{[]byte("abcd"), j(rpl(1, 2, "xy")), []byte("axyd")},
		{[]byte("abcd"), j(rpl(1, 1, "xy")), []byte("axycd")},
		{[]byte("abcd"), j(rpl(1, 1, "xyz")), []byte("axyzcd")},

		{[]byte("abcd"), j(rpl(0, 4, "xy"), rpl(0, 4, "01")), []byte("01")},
		{[]byte("abcd"), j(rpl(0, 4, "xy"), rpl(0, 4, "0")), []byte("0y")},
		{[]byte("abcd"), j(rpl(0, 3, "xy")), []byte("xyd")},
		{[]byte("abcd"), j(rpl(0, 3, "xy"), rpl(0, 4, "0")), []byte("0y")},

		// 2 step combinations that are disjoint
		// Note: that since operations are sorted the set of test cases is
		//       reduced to del+{rpl,ins}, and rpl+ins
		{[]byte("abcdedf"), j(del(0, 4), rpl(4, 2, "xy")), []byte("xyf")},
		{[]byte("abcdedf"), j(del(0, 4), rpl(4, 2, "xyz")), []byte("xyzf")},
		{[]byte("abcdedf"), j(del(1, 3), rpl(4, 2, "xy")), []byte("axyf")},
		{[]byte("abcdedf"), j(del(1, 3), rpl(4, 2, "xyz")), []byte("axyzf")},
		{[]byte("abcdedf"), j(del(0, 4), ins(4, "xy")), []byte("xyedf")},
		{[]byte("abcdedf"), j(del(0, 4), ins(4, "xyz")), []byte("xyzedf")},
		{[]byte("abcdedf"), j(del(1, 3), ins(4, "xy")), []byte("axyedf")},
		{[]byte("abcdedf"), j(del(1, 3), ins(4, "xyz")), []byte("axyzedf")},

		{[]byte("abcded"), j(rpl(0, 2, "0"), ins(4, "xyz")), []byte("0cdxyzed")},
		{[]byte("abcded"), j(rpl(0, 2, "01"), ins(4, "xyz")), []byte("01cdxyzed")},
		{[]byte("abcded"), j(rpl(0, 2, "012"), ins(4, "xyz")), []byte("012cdxyzed")},
		{[]byte("abcded"), j(rpl(1, 2, "0"), ins(4, "xyz")), []byte("a0dxyzed")},
		{[]byte("abcded"), j(rpl(1, 2, "01"), ins(4, "xyz")), []byte("a01dxyzed")},
		{[]byte("abcded"), j(rpl(1, 2, "012"), ins(4, "xyz")), []byte("a012dxyzed")},

		// 2 step combinations that overlap
		{[]byte("abcdedf"), j(del(0, 4), rpl(2, 2, "xy")), []byte("xyedf")},
		{[]byte("abcdedf"), j(del(0, 4), rpl(2, 2, "xyz")), []byte("xyzedf")},
		{[]byte("abcdedf"), j(del(1, 3), rpl(2, 2, "xy")), []byte("axyedf")},
		{[]byte("abcdedf"), j(del(1, 3), rpl(2, 2, "xyz")), []byte("axyzedf")},
		{[]byte("abcdedf"), j(del(0, 4), ins(2, "xy")), []byte("xyedf")},
		{[]byte("abcdedf"), j(del(0, 4), ins(2, "xyz")), []byte("xyzedf")},
		{[]byte("abcdedf"), j(del(1, 3), ins(2, "xy")), []byte("axyedf")},
		{[]byte("abcdedf"), j(del(1, 3), ins(2, "xyz")), []byte("axyzedf")},

		{[]byte("abcded"), j(rpl(0, 3, "0"), ins(2, "xyz")), []byte("0xyzded")},
		{[]byte("abcded"), j(rpl(0, 3, "01"), ins(2, "xyz")), []byte("01xyzded")},
		{[]byte("abcded"), j(rpl(0, 3, "012"), ins(2, "xyz")), []byte("012xyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "0"), ins(2, "xyz")), []byte("a0xyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "01"), ins(2, "xyz")), []byte("a01xyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "012"), ins(2, "xyz")), []byte("a012xyzded")},

		{[]byte("abcded"), j(rpl(0, 3, "0"), rpl(0, 3, "A"), ins(2, "xyz")), []byte("Axyzded")},
		{[]byte("abcded"), j(rpl(0, 3, "01"), rpl(0, 3, "AB"), ins(2, "xyz")), []byte("ABxyzded")},
		{[]byte("abcded"), j(rpl(0, 3, "012"), rpl(0, 3, "ABC"), ins(2, "xyz")), []byte("ABCxyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "0"), rpl(1, 2, "A"), ins(2, "xyz")), []byte("aAxyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "01"), rpl(1, 2, "AB"), ins(2, "xyz")), []byte("aABxyzded")},
		{[]byte("abcded"), j(rpl(1, 2, "012"), rpl(1, 2, "ABC"), ins(2, "xyz")), []byte("aABCxyzded")},

		// 3+ step combinations.
		{[]byte("abcded"), j(del(0, 2), rpl(0, 3, "A"), ins(0, "xyz")), []byte("Axyzded")},
		{[]byte("abcded"), j(del(2, 2), rpl(2, 3, "A"), ins(2, "xyz")), []byte("abAxyzd")},
		{[]byte("abcded"), j(del(2, 2), rpl(2, 3, "ABC"), ins(2, "xyz")), []byte("abABCxyzd")},
		{[]byte("abcded"), j(del(2, 2), rpl(2, 3, "ABC"), rpl(2, 2, "XX"), ins(2, "xyz")), []byte("abXXCxyzd")},
	} {
		if err := edit.Validate(tc.contents, tc.deltas...); err != nil {
			t.Errorf("%v: unexpected error: %v", i, err)
		}
		if got, want := edit.Do(tc.contents, tc.deltas...), tc.edited; !bytes.Equal(got, want) {
			t.Errorf("%v: got %s, want %s", i, got, want)
		}
	}

	// out of range test cases.
	for i, tc := range []struct {
		contents []byte
		deltas   []edit.Delta
		edited   []byte
	}{
		{[]byte{}, j(ins(100, "ab")), []byte{}},
		{[]byte("ab"), j(del(3, 1)), []byte("ab")},
		{[]byte("ab"), j(del(1, 10)), []byte("ab")},
		{[]byte("ab"), j(rpl(3, 1, "xy")), []byte("ab")},
		{[]byte("ab"), j(rpl(1, 10, "xy"), del(10, 20)), []byte("ab")},
	} {
		err := edit.Validate(tc.contents, tc.deltas...)
		if err != nil {
			if got, want := err.Error(), "out of range"; !strings.Contains(got, want) {
				t.Errorf("%v: %s does not contain %s", i, got, want)
			}
		} else {
			t.Errorf("%v: expected an error:", i)
		}
		if got, want := edit.Do(tc.contents, tc.deltas...), tc.edited; !bytes.Equal(got, want) {
			t.Errorf("%v: got %s, want %s", i, got, want)
		}
	}

}
