// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/matcher"
)

func parse(input string) []matcher.Item {
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")
	input = strings.ReplaceAll(input, "||", " || ")
	input = strings.ReplaceAll(input, "&&", " && ")
	items := []matcher.Item{}
	tokens := strings.Split(input, " ")
	for i := 0; i < len(tokens); i++ {
		if len(tokens[i]) == 0 {
			continue
		}
		switch tokens[i] {
		case "||":
			items = append(items, matcher.OR())
		case "&&":
			items = append(items, matcher.AND())
		case "(":
			items = append(items, matcher.LeftBracket())
		case ")":
			items = append(items, matcher.RightBracket())
		case "ft:":
			i++
			items = append(items, matcher.FileType(tokens[i]))
		case "nt:":
			date := ""
			i++
			for j := i; j < len(tokens); j++ {
				i++
				if tokens[j] == ":nt" {
					break
				}
				date += tokens[j] + " "
			}
			items = append(items, matcher.NewerThanParsed(strings.TrimSpace(date)))
		default:
			items = append(items, matcher.Regexp(tokens[i]))
		}
	}
	return items
}

func TestFormating(t *testing.T) {
	for _, tc := range []struct {
		in  string
		out string
	}{
		{"", ""},
		{"foo", "foo"},
		{"foo || bar", "foo || bar"},
		{"foo && bar", "foo && bar"},
		{"foo && bar || baz", "foo && bar || baz"},
		{"foo || bar && baz", "foo || bar && baz"},
		{"foo && (bar||baz)", "foo && (bar || baz)"},
		{"(bar || baz) && foo", "(bar || baz) && foo"},
		{"( bar && baz )", "(bar && baz)"},
		{"(bar && (baz || foo)) || else", "(bar && (baz || foo)) || else"},
		{"ft: f || nt: 2023-10-22", `f || 2023-10-22`},
	} {
		expr, err := matcher.New(parse(tc.in)...)
		if err != nil {
			t.Errorf("%v: failed to create expression: %v", tc.in, err)
			continue
		}
		if got, want := expr.String(), tc.out; got != want {
			t.Errorf("%v: got %v, want %v", tc.in, got, want)
		}
	}
}

func fn(name string) any {
	return file.NewInfo(name, 0, 0, time.Time{}, nil)
}

func ft(mode fs.FileMode) any {
	return file.NewInfo("", 0, mode, time.Time{}, nil)
}

func fm(modTime time.Time) any {
	return file.NewInfo("", 0, 0, modTime, nil)
}

func evalTestCase(t *testing.T, in []matcher.Item, val any, want bool) {
	t.Helper()
	expr, err := matcher.New(in...)
	if err != nil {
		t.Errorf("failed to create expression: %v", err)
		return
	}
	r := expr.Eval(val)
	if got := r; got != want {
		t.Errorf("%v: %v: got %v, want %v", expr, val, got, want)
	}
}

func TestOperands(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		in  string
		val any
		out bool
	}{
		{"foo", fn("foo"), true},
		{"foo", fn("f"), false},
		{"ft: f", ft(0), true},
		{"ft: d", ft(fs.ModeDir), true},
		{"ft: l", ft(fs.ModeSymlink), true},
		{"ft: f", ft(fs.ModeDevice), false},
		{"ft: d", ft(fs.ModeDevice), false},
		{"ft: l", ft(fs.ModeDevice), false},
		{"nt: " + now.Format(time.DateTime) + " :nt", fm(now.Add(-time.Hour)), false},
		{"nt: " + now.Format(time.DateTime) + " :nt", fm(now.Add(time.Hour)), true},
	} {
		evalTestCase(t, parse(tc.in), tc.val, tc.out)
	}
}

func TestOperators(t *testing.T) {
	for _, tc := range []struct {
		in  string
		val any
		out bool
	}{
		{"", fn("foo"), false},
		{`^fo && .ext$`, fn("foo.ext"), true},
		{`^fo && .ext$`, fn("foo"), false},
		{`^fo || .ext$`, fn("foo"), true},
		{`^fo || .ext$`, fn("x.ext"), true},
		{`^fo || .ext$`, fn("not"), false},
		{`^fo || .ext$ || wombat`, fn("foo.ext"), true},
		{`^fo || .ext$ || wombat`, fn("wombat"), true},
		{`^fo || .ext$ && wombat`, fn("wombat.ext"), true},
		{`^fo && .ext$ || wombat`, fn("foo.ext"), true},
		{`^fo && .ext$ || wombat`, fn("wombat"), true},
		{`^fo && (.ext$ || wombat)`, fn("wombat"), false},
		{`^fo && (.ext$ || wombat)`, fn("wombat.ext"), false},
		{`^fo && .ext$ && wombat`, fn("wombat"), false},
	} {
		evalTestCase(t, parse(tc.in), tc.val, tc.out)
	}
}

func TestSubExpressions(t *testing.T) {
	for _, tc := range []struct {
		in  string
		val any
		out bool
	}{
		{`(foo || bar)`, fn("foo"), true},
		{`(foo || bar)`, fn("wombat"), false},
		{`(foo || bar) || wom.*`, fn("wombat"), true},
		{`wo* || ( foo || bar )`, fn("wombat"), true},
		{`wo.* && (foo || bar)`, fn("wombat"), false},
		{`wo.* && (foo || .ext$)`, fn("wombat.ext"), true},
		{`wo.* && (foo || .ext$)`, fn("foo.ext"), false},
		{`(foo || \.ext$) && (wombat && \.ext$)`, fn("wombat.ext"), true},
		{`(foo || \.ext$) && (wombat && \.ext$)`,
			fn("wombat"), false},
		{`(foo || \.ext$) || (wombat && \.ext$)`, fn("wombat"), false},
		{`(foo || \.ext$) || (wombat && \.ext$)`, fn("wombat"), false},
		{`(foo && (baz || bar))`, fn("foo"), false},
		{`(foo && (baz || bar))`, fn("baz"), false},
		{`(foo && (baz || bar))`, fn("foobar"), true},
		{`(^foo && (^baz || ^bar))`, fn("foobar"), false},
		{`(ft: f || ft: d)`, ft(0), true},
		{`(ft: f || ft: d)`, ft(fs.ModeDir), true},
		{`(ft: f || ft: d) && ft: l`, ft(fs.ModeDir), false},
	} {
		evalTestCase(t, parse(tc.in), tc.val, tc.out)
	}
}

func TestErrors(t *testing.T) {
	for _, tc := range []struct {
		in  string
		err string
	}{
		{`(`, "unbalanced brackets"},
		{`()`, "missing left operand for )"},
		{`(foo || bar`, "unbalanced brackets"},
		{`foo || bar)`, "unbalanced brackets"},
		{`)(`, "unbalanced brackets"},
		{`||`, "missing left operand for ||"},
		{`|| a`, "missing left operand for ||"},
		{`a ||`, "incomplete expression: [a ||]"},
		{`&&`, "missing left operand for &&"},
		{`&& a`, "missing left operand for &&"},
		{`a &&`, "incomplete expression: [a &&]"},
		{`a || b || ()`, "missing left operand for )"},
		{`( a || )`, "missing operand for )"},
		{`( a () )`, "missing operator for ("},
		{`|| ||`, "missing left operand for ||"},
		{`a || ||`, "missing operand for ||"},
		{`&& &&`, "missing left operand for &&"},
		{`a && &&`, "missing operand for &&"},
		{`a (a)`, "missing operator for ("},
		{`[a-z+`, "error parsing regexp: missing closing ]: `[a-z+`"},
		{`ft: x`, "invalid file type: x, use one of d, f or l"},
		{`nt: xxx :nt`, "invalid time: xxx, use one of RFC3339, Date and Time, Date or Time only formats"},
	} {
		m, err := matcher.New(parse(tc.in)...)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%v: got %v, want %v", tc.in, err, tc.err)
		}
		if got, want := m.Eval(fn("foo")), false; got != want {
			t.Errorf("%v: got %v, want %v", tc.in, got, want)
		}
	}
}

type nameOnly struct{}

func (nameOnly) Name() string { return "" }

type modeOnly struct{}

func (modeOnly) Mode() fs.FileMode { return 0 }

type typeOnly struct{}

func (typeOnly) Type() fs.FileMode { return 0 }

type modTimeOnly struct{}

func (modTimeOnly) ModTime() time.Time { return time.Time{} }

func TestNeedsOps(t *testing.T) {
	var err error
	assert := func(m matcher.T, needsName, needsMode, needsTime bool) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := m.Needs(nameOnly{}), needsName; got != want {
			t.Errorf("mode: got %v, want %v", got, want)
		}
		if got, want := m.Needs(modeOnly{}), needsMode; got != want {
			t.Errorf("mode: got %v, want %v", got, want)
		}
		if got, want := m.Needs(typeOnly{}), needsMode; got != want {
			t.Errorf("mode: got %v, want %v", got, want)
		}
		if got, want := m.Needs(modTimeOnly{}), needsTime; got != want {
			t.Errorf("time: got %v, want %v", got, want)
		}
	}

	var m matcher.T
	assert(m, false, false, false)
	if got, want := m.Needs(file.Info{}), false; got != want {
		t.Errorf("file.Info: got %v, want %v", got, want)
	}

	m, err = matcher.New(parse("xx")...)
	if err != nil {
		t.Fatal(err)
	}
	assert(m, true, false, false)

	m, err = matcher.New(parse("xx || ft: f")...)
	assert(m, true, true, false)
	m, err = matcher.New(parse("xx || nt: 2022-12-12 :nt")...)
	assert(m, true, false, true)
	m, err = matcher.New(parse("xx || ft: d || nt: 2022-12-12 :nt")...)
	assert(m, true, true, true)

	m, err = matcher.New(parse("a && ( xx || ft: f )")...)
	assert(m, true, true, false)
	m, err = matcher.New(parse("a && (xx || nt: 2022-12-12 :nt )")...)
	assert(m, true, false, true)
	m, err = matcher.New(parse("a && (xx || ft: d || nt: 2022-12-12 :nt )")...)
	assert(m, true, true, true)
	m, err = matcher.New(parse(" (ft: d || nt: 2022-12-12 :nt )")...)
	assert(m, false, true, true)

	if got, want := m.Needs(file.Info{}), true; got != want {
		t.Errorf("file.Info: got %v, want %v", got, want)
	}
}
