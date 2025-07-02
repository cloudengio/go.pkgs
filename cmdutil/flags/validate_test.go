// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func TestValidate(t *testing.T) {
	l := func(args ...any) []any { return args }
	for i, tc := range []struct {
		args                       []any
		exactlyOne, atmostOne, all bool
	}{
		{nil, false, true, true},
		{l(""), false, true, false},
		{l("", ""), false, true, false},
		{l("a"), true, true, true},
		{l("a", ""), true, true, false},
		{l("a", "b"), false, false, true},
		{l([]int{}), false, true, false},
		{l([]int{}, []float64{}), false, true, false},
		{l([]string{"a"}), true, true, true},
		{l([]int{1}, []string{}), true, true, false},
		{l([]int{1}, []int{2}), false, false, true},
		{l([]int{1}, map[int]int{2: 2}), false, false, true},
	} {
		if got, want := flags.ExactlyOneSet(tc.args...), tc.exactlyOne; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := flags.AtMostOneSet(tc.args...), tc.atmostOne; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := flags.AllSet(tc.args...), tc.all; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}

func TestPanic(t *testing.T) {
	defer func() {
		e := recover()
		t.Log(e)
	}()
	flags.ExactlyOneSet(struct{ a int }{1})
	t.Logf("should never get this far.")
	t.FailNow()
}
