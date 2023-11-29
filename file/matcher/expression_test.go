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
		case "glob:":
			i++
			items = append(items, matcher.Glob(tokens[i], false))
		case "iglob:":
			i++
			items = append(items, matcher.Glob(tokens[i], true))
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
		{"foo", "re=foo"},
		{"foo || bar", "re=foo || re=bar"},
		{"foo && bar", "re=foo && re=bar"},
		{"foo && bar || baz", "re=foo && re=bar || re=baz"},
		{"foo || bar && baz", "re=foo || re=bar && re=baz"},
		{"foo && (bar||baz)", "re=foo && (re=bar || re=baz)"},
		{"(bar || baz) && foo", "(re=bar || re=baz) && re=foo"},
		{"( bar && baz )", "(re=bar && re=baz)"},
		{"(bar && (baz || foo)) || else", "(re=bar && (re=baz || re=foo)) || re=else"},
		{"ft: f || nt: 2023-10-22", `type=f || newer=2023-10-22`},
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
		{`glob: foo`, fn("foo"), true},
		{`glob: foo`, fn("Foo"), false},
		{`iglob: foo`, fn("Foo"), true},
		{`glob: f*foo`, fn("ffoo"), true},
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
		{`a ||`, "incomplete expression: [re=a ||]"},
		{`&&`, "missing left operand for &&"},
		{`&& a`, "missing left operand for &&"},
		{`a &&`, "incomplete expression: [re=a &&]"},
		{`a || b || ()`, "missing left operand for )"},
		{`( a || )`, "missing operand for )"},
		{`( a () )`, "missing operator for ("},
		{`|| ||`, "missing left operand for ||"},
		{`a || ||`, "missing operand for ||"},
		{`&& &&`, "missing left operand for &&"},
		{`a && &&`, "missing operand for &&"},
		{`a (a)`, "missing operator for ("},
		{`[a-z+`, "error parsing regexp: missing closing ]: `[a-z+`"},
		{`ft: z`, "invalid file type: \"z\", use one of d, f, l or x"},
		{`nt: xxx :nt`, "invalid time: xxx, use one of RFC3339, Date and Time, Date or Time only formats"},
		{`glob: [x`, "syntax error in pattern"},
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
