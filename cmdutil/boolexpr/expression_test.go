// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
)

type regexOp struct {
	val  string
	name string // allow some tests to use a different name for this operand.
	re   *regexp.Regexp
}

func (eq regexOp) String() string {
	if eq.name == "" {
		eq.name = "re"
	}
	return eq.name + "=" + eq.val
}

func (eq regexOp) Document() string {
	if eq.name == "" {
		eq.name = "re"
	}
	return eq.name + ": regular expression"
}

func (eq regexOp) Prepare() (boolexpr.Operand, error) {
	var err error
	eq.re, err = regexp.Compile(eq.val)
	return eq, err
}

func (eq regexOp) Eval(v any) bool {
	return eq.re.MatchString(v.(string))
}

func (eq regexOp) Needs(t reflect.Type) bool {
	return t == reflect.TypeOf("")
}

func parse(input string) []boolexpr.Item {
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")
	input = strings.ReplaceAll(input, "||", " || ")
	input = strings.ReplaceAll(input, "&&", " && ")
	items := []boolexpr.Item{}
	tokens := strings.Split(input, " ")
	for i := 0; i < len(tokens); i++ {
		if len(tokens[i]) == 0 {
			continue
		}
		switch tokens[i] {
		case "||":
			items = append(items, boolexpr.OR())
		case "&&":
			items = append(items, boolexpr.AND())
		case "(":
			items = append(items, boolexpr.LeftBracket())
		case ")":
			items = append(items, boolexpr.RightBracket())
		default:
			items = append(items, boolexpr.NewOperandItem(&regexOp{val: tokens[i]}))
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
	} {
		expr, err := boolexpr.New(parse(tc.in)...)
		if err != nil {
			t.Errorf("%v: failed to create expression: %v", tc.in, err)
			continue
		}
		if got, want := expr.String(), tc.out; got != want {
			t.Errorf("%v: got %v, want %v", tc.in, got, want)
		}
	}
}

func evalTestCase(t *testing.T, in []boolexpr.Item, val any, want bool) {
	t.Helper()
	expr, err := boolexpr.New(in...)
	if err != nil {
		t.Errorf("failed to create expression: %v", err)
		return
	}
	r := expr.Eval(val)
	if got := r; got != want {
		t.Errorf("%v: %v: got %v, want %v", expr, val, got, want)
	}
}

func TestOperators(t *testing.T) {
	for _, tc := range []struct {
		in  string
		val any
		out bool
	}{
		{"", "foo", false},
		{`^fo && .ext$`, "foo.ext", true},
		{`^fo && .ext$`, "foo", false},
		{`^fo || .ext$`, "foo", true},
		{`^fo || .ext$`, "x.ext", true},
		{`^fo || .ext$`, "not", false},
		{`^fo || .ext$ || wombat`, "foo.ext", true},
		{`^fo || .ext$ || wombat`, "wombat", true},
		{`^fo || .ext$ && wombat`, "wombat.ext", true},
		{`^fo && .ext$ || wombat`, "foo.ext", true},
		{`^fo && .ext$ || wombat`, "wombat", true},
		{`^fo && (.ext$ || wombat)`, "wombat", false},
		{`^fo && (.ext$ || wombat)`, "wombat.ext", false},
		{`^fo && .ext$ && wombat`, "wombat", false},
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
		{`(foo || bar)`, "foo", true},
		{`(foo || bar)`, "wombat", false},
		{`(foo || bar) || wom.*`, "wombat", true},
		{`wo* || ( foo || bar )`, "wombat", true},
		{`wo.* && (foo || bar)`, "wombat", false},
		{`wo.* && (foo || .ext$)`, "wombat.ext", true},
		{`wo.* && (foo || .ext$)`, "foo.ext", false},
		{`(foo || \.ext$) && (wombat && \.ext$)`, "wombat.ext", true},
		{`(foo || \.ext$) && (wombat && \.ext$)`,
			"wombat", false},
		{`(foo || \.ext$) || (wombat && \.ext$)`, "wombat", false},
		{`(foo || \.ext$) || (wombat && \.ext$)`, "wombat", false},
		{`(foo && (baz || bar))`, "foo", false},
		{`(foo && (baz || bar))`, "baz", false},
		{`(foo && (baz || bar))`, "foobar", true},
		{`(^foo && (^baz || ^bar))`, "foobar", false},
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
	} {
		m, err := boolexpr.New(parse(tc.in)...)
		if err == nil || err.Error() != tc.err {
			t.Errorf("%v: got %v, want %v", tc.in, err, tc.err)
		}
		if got, want := m.Eval("foo"), false; got != want {
			t.Errorf("%v: got %v, want %v", tc.in, got, want)
		}
	}
}

type nameIfc interface {
	Name() string
}

type regexNameOp struct{}

func (rno regexNameOp) Needs(t reflect.Type) bool {
	needs := reflect.TypeOf((*nameIfc)(nil)).Elem()
	return t.Implements(needs)
}

func (rno regexNameOp) String() string {
	return "nre="
}

func (rno regexNameOp) Document() string {
	return "nre: regular expression"
}

func (rno regexNameOp) Prepare() (boolexpr.Operand, error) {
	return rno, nil
}

func (rno regexNameOp) Eval(v any) bool {
	return false
}

type nifcImpl struct{}

func (n nifcImpl) Name() string {
	return "foo"
}

func TestNeeds(t *testing.T) {
	var err error
	var expr boolexpr.T
	assert := func(strT, nameT, nameTPtr bool) {
		t.Helper()
		if err != nil {
			t.Errorf("failed to create expression: %v", err)
		}
		if got, want := expr.Needs(""), strT; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := expr.Needs(nifcImpl{}), nameT; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := expr.Needs(&nifcImpl{}), nameTPtr; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	expr, err = boolexpr.New()
	assert(false, false, false)

	expr, _ = boolexpr.New(boolexpr.NewOperandItem(regexOp{}))
	assert(true, false, false)

	expr, err = boolexpr.New(
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.OR(),
		boolexpr.NewOperandItem(&regexNameOp{}))

	assert(true, true, true)

	expr, err = boolexpr.New(boolexpr.NewOperandItem(&regexNameOp{}))
	assert(false, true, true)

	expr, err = boolexpr.New(boolexpr.LeftBracket(),
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.RightBracket())
	assert(true, false, false)

	expr, err = boolexpr.New(
		boolexpr.LeftBracket(),
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.OR(),
		boolexpr.LeftBracket(),
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.RightBracket(),
		boolexpr.RightBracket())
	assert(true, false, false)

	expr, err = boolexpr.New(
		boolexpr.LeftBracket(),
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.OR(),
		boolexpr.LeftBracket(),
		boolexpr.NewOperandItem(&regexNameOp{}),
		boolexpr.RightBracket(),
		boolexpr.RightBracket())
	assert(true, true, true)

}
