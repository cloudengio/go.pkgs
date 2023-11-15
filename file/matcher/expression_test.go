// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"fmt"
	"io/fs"
	"testing"
	"time"

	"cloudeng.io/file/matcher"
)

func or() matcher.Item {
	return matcher.OR()
}

func and() matcher.Item {
	return matcher.AND()
}

func re(e string) matcher.Item {
	return matcher.Regexp(e)
}

func ftyp(t string) matcher.Item {
	return matcher.FileType(t)
}

func newer(t string) matcher.Item {
	return matcher.NewerThan(t)
}

func lb() matcher.Item {
	return matcher.LeftBracket()
}

func rb() matcher.Item {
	return matcher.RightBracket()
}

func mi(items ...matcher.Item) []matcher.Item {
	return items
}

func TestFormating(t *testing.T) {
	for _, tc := range []struct {
		in  []matcher.Item
		out string
	}{
		{in: mi(re("foo")),
			out: "foo"},
		{in: mi(re("foo"), or(), re("bar")),
			out: "foo || bar"},
		{in: mi(re("foo"), and(), re("bar")),
			out: "foo && bar"},
		{in: mi(re("foo"), and(), re("bar"), or(), re("baz")),
			out: "foo && bar || baz"},
		{in: mi(re("foo"), or(), re("bar"), and(), re("baz")),
			out: "foo || bar && baz"},
		{in: mi(re("foo"), and(), lb(), re("bar"), or(), re("baz"), rb()),
			out: "foo && (bar || baz)"},
		{in: mi(lb(), re("bar"), or(), re("baz"), rb(), and(), re("foo")),
			out: "(bar || baz) && foo"},
		{in: mi(lb(), re("bar"), and(), re("baz"), rb()),
			out: "(bar && baz)"},
		{in: mi(lb(), re("bar"), and(), lb(), re("baz"), or(), re("foo"), rb(), rb(), or(), re("else")),
			out: "(bar && (baz || foo)) || else"},
		{in: mi(ftyp("f"), or(), newer("2023-10-22")), out: "filetype(f) || newerthan(2023-10-22)"},
	} {
		expr, err := matcher.NewExpression(tc.in...)
		if err != nil {
			t.Errorf("failed to create expression: %v", err)
			continue
		}
		if got, want := expr.String(), tc.out; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestErrors(t *testing.T) {
	t.Fail()
}

type evaluable struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
}

func (e evaluable) Name() string {
	return e.name
}

func (e evaluable) Mode() fs.FileMode {
	return e.mode
}

func (e evaluable) ModTime() time.Time {
	return e.modTime
}

func fn(name string) evaluable {
	return evaluable{name: name}
}

func ft(mode fs.FileMode) evaluable {
	return evaluable{mode: mode}
}

func fm(modTime time.Time) evaluable {
	return evaluable{modTime: modTime}
}

func (e evaluable) String() string {
	return fmt.Sprintf("name: %q, mode: %v, modtime: %v", e.name, e.mode, e.modTime)
}

func TestEval(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		in  []matcher.Item
		val evaluable
		out bool
	}{
		{in: mi(re("foo")), val: fn("foo"), out: true},
		{in: mi(re("foo")), val: fn("f"), out: false},
		{in: mi(ftyp("f")), val: ft(0), out: true},
		{in: mi(ftyp("d")), val: ft(fs.ModeDir), out: true},
		{in: mi(ftyp("l")), val: ft(fs.ModeSymlink), out: true},
		{in: mi(ftyp("f")), val: ft(fs.ModeDevice), out: false},
		{in: mi(ftyp("d")), val: ft(fs.ModeDevice), out: false},
		{in: mi(ftyp("l")), val: ft(fs.ModeDevice), out: false},
		{in: mi(newer(now.Format(time.DateTime))), val: fm(now.Add(-time.Hour)), out: false},
		{in: mi(newer(now.Format(time.DateTime))), val: fm(now.Add(time.Hour)), out: true},

		{in: mi(re("^fo"), and(), re(".ext$")), val: fn("foo.ext"), out: true},
		{in: mi(re("^fo"), and(), re(".ext$")), val: fn("foo"), out: false},
		{in: mi(re("^fo"), or(), re(".ext$")), val: fn("foo"), out: true},
		{in: mi(re("^fo"), or(), re(".ext$")), val: fn("x.ext"), out: true},
		{in: mi(re("^fo"), or(), re(".ext$")), val: fn("not"), out: false},

		/*
			{in: mi(re("foo"), or(), re("bar")),
				out: "foo || bar"},
			{in: mi(re("foo"), and(), re("bar")),
				out: "foo && bar"},
			{in: mi(re("foo"), and(), re("bar"), or(), re("baz")),
				out: "foo && bar || baz"},
			{in: mi(re("foo"), or(), re("bar"), and(), re("baz")),
				out: "foo || bar && baz"},
			{in: mi(re("foo"), and(), lb(), re("bar"), or(), re("baz"), rb()),
				out: "foo && (bar || baz)"},
			{in: mi(lb(), re("bar"), or(), re("baz"), rb(), and(), re("foo")),
				out: "(bar || baz) && foo"},
			{in: mi(lb(), re("bar"), and(), re("baz"), rb()),
				out: "(bar && baz)"},
			{in: mi(lb(), re("bar"), and(), lb(), re("baz"), or(), re("foo"), rb(), rb(), or(), re("else")),
				out: "(bar && (baz || foo)) || else"},
			{in: mi(ftyp("f"), or(), newer("2023-10-22")), out: "filetype(f) || newerthan(2023-10-22)"},*/
	} {
		expr, err := matcher.NewExpression(tc.in...)
		if err != nil {
			t.Errorf("failed to create expression: %v", err)
			continue
		}
		r, err := expr.Eval(tc.val)
		if err != nil {
			t.Errorf("failed to evaluate expression: %v", err)
			continue
		}
		if got, want := r, tc.out; got != want {
			t.Errorf("%v: %v: got %v, want %v", expr, tc.val, got, want)
		}
	}
}
