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
	return t == reflect.TypeFor[string]()
}

func parse(input string) []boolexpr.Item {
	input = strings.ReplaceAll(input, "(", " ( ")
	input = strings.ReplaceAll(input, ")", " ) ")
	input = strings.ReplaceAll(input, "||", " || ")
	input = strings.ReplaceAll(input, "&&", " && ")
	input = strings.ReplaceAll(input, "!", " ! ")
	items := []boolexpr.Item{}
	tokens := strings.Split(input, " ")
	for i := range tokens {
		if len(tokens[i]) == 0 {
			continue
		}
		switch tokens[i] {
		case "||":
			items = append(items, boolexpr.OR())
		case "&&":
			items = append(items, boolexpr.AND())
		case "!":
			items = append(items, boolexpr.NOT())
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
		{"!foo", "!re=foo"},
		{"foo || bar", "re=foo || re=bar"},
		{"foo || !bar", "re=foo || !re=bar"},
		{"!foo || bar", "!re=foo || re=bar"},
		{"foo && bar", "re=foo && re=bar"},
		{"foo && bar || baz", "re=foo && re=bar || re=baz"},
		{"foo && !bar || baz", "re=foo && !re=bar || re=baz"},
		{"foo || bar && baz", "re=foo || re=bar && re=baz"},
		{"foo && (bar||baz)", "re=foo && (re=bar || re=baz)"},
		{"(bar || baz) && foo", "(re=bar || re=baz) && re=foo"},
		{"( bar && baz )", "(re=bar && re=baz)"},
		{"!( bar && baz )", "!(re=bar && re=baz)"},
		{"(bar && (baz || foo)) || else", "(re=bar && (re=baz || re=foo)) || re=else"},
		{"(bar && !(baz || foo)) || else", "(re=bar && !(re=baz || re=foo)) || re=else"},
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
		{"foo", "foo", true},
		{"!foo", "foo", false},
		{"foo", "bar", false},
		{"!foo", "bar", true},

		{"foo && bar", "neither", false},
		{"foo && bar", "foobar", true},
		{"foo && bar", "foo", false},
		{"foo && bar", "bar", false},
		{"foo && !bar", "neither", false},
		{"!foo && bar", "neither", false},
		{"!foo && !bar", "neither", true},
		{"foo && !bar", "foo", true},
		{"!foo && bar", "bar", true},
		{"!foo && !bar", "foo", false},

		{"foo || bar", "neither", false},
		{"foo || bar", "foobar", true},
		{"foo || bar", "foo", true},
		{"foo || bar", "bar", true},
		{"foo || !bar", "neither", true},
		{"!foo || bar", "neither", true},
		{"!foo || !bar", "neither", true},
		{"foo || !bar", "foo", true},
		{"!foo || bar", "bar", true},
		{"!foo || !bar", "foo", true},
		{"!foo || !bar", "foobar", false},

		{"foo && bar && baz", "neither", false},
		{"foo && bar && baz", "foobarbaz", true},
		{"!foo && !bar && !baz", "neither", true},
		{"!foo && !bar && !baz", "foo", false},
		{"!foo && !bar && !baz", "bar", false},
		{"!foo && !bar && !baz", "baz", false},
		{"!foo && bar && baz", "foobarbaz", false},
		{"foo && !bar && baz", "foobarbaz", false},
		{"foo && bar && !baz", "foobarbaz", false},

		{"foo || bar || baz", "neither", false},
		{"foo || bar || baz", "foobarbaz", true},
		{"!foo || !bar || !baz", "foobarbaz", false},
		{"!foo || !bar || !baz", "neither", true},
		{"!foo || !bar || !baz", "foo", true},
		{"!foo || !bar || !baz", "bar", true},
		{"!foo || !bar || !baz", "baz", true},
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
		{`(foo || bar)`, "bar", true},
		{`!(foo || bar)`, "foo", false},
		{`!(foo || bar)`, "bar", false},
		{`(foo && bar)`, "foo", false},
		{`(foo && bar)`, "bar", false},
		{`(foo && bar)`, "foobar", true},
		{`!(foo && bar)`, "foobar", false},
		{`!(foo && bar)`, "bar", true},
		{`(foo && bar) || baz`, "foobar", true},
		{`(foo && bar) || baz`, "baz", true},
		{`baz || (foo && bar)`, "foobar", true},
		{`!((foo && bar) || baz)`, "foobar", false},
		{`!((foo && bar) || baz)`, "baz", false},
		{`(foo && bar) || (baz && bat)`, "foobar", true},
		{`(foo && bar) || (baz && bat)`, "batbaz", true},
		{`(foo && bar) || !(baz && bat)`, "batbaz", false},
		{`!(foo && bar) || !(baz && bat)`, "batbaz", true},
		{`!(!(foo && bar) || !(baz && bat))`, "batbaz", false},
	} {
		evalTestCase(t, parse(tc.in), tc.val, tc.out)
	}
}

func TestErrors(t *testing.T) {
	for _, tc := range []struct {
		in  string
		err string
	}{
		{`(`, "unbalanced brackets: missing ')'"},
		{`()`, "empty subexpression '()' at item 0"},
		{`!`, "incomplete expression: expected operand or '(' at end of expression"},
		{`!!`, "incomplete expression: expected operand or '(' at end of expression"},
		{`!&&`, "missing operand before && at item 1"},
		{`&&!`, "missing operand before && at item 0"},
		{`&&&&`, "missing operand before && at item 0"},
		{`(foo || bar`, "unbalanced brackets: missing ')'"},
		{`(foo || bar)!`, "missing operator before ! at item 5"},
		{`(foo !|| bar)`, "missing operator before ! at item 2"},
		{`foo || bar)`, "unbalanced brackets: unexpected ')' at item 3"},
		{`)(`, "unbalanced brackets: unexpected ')' at item 0"},
		{`||`, "missing operand before || at item 0"},
		{`|| a`, "missing operand before || at item 0"},
		{`a ||`, "incomplete expression: expected operand or '(' at end of expression"},
		{`a && b &&`, "incomplete expression: expected operand or '(' at end of expression"},
		{`a && !`, "incomplete expression: expected operand or '(' at end of expression"},
		{`&&`, "missing operand before && at item 0"},
		{`&& a`, "missing operand before && at item 0"},
		{`a &&`, "incomplete expression: expected operand or '(' at end of expression"},
		{`a || b || ()`, "empty subexpression '()' at item 4"},
		{`( a || )`, "unbalanced brackets: unexpected ')' at item 3"},
		{`( a () )`, "missing operator before ( at item 2"},
		{`|| ||`, "missing operand before || at item 0"},
		{`a || ||`, "missing operand before || at item 2"},
		{`&& &&`, "missing operand before && at item 0"},
		{`a && &&`, "missing operand before && at item 2"},
		{`a (a)`, "missing operator before ( at item 1"},
		{`a b c`, "missing operator before operand at item 1"},
		{`a || b c`, "missing operator before operand at item 3"},
		{`(a) b`, "missing operator before operand at item 3"},
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

	if _, err := boolexpr.New(boolexpr.NewOperandItem(nil)); err == nil {
		t.Errorf("expected error for nil operand")
	}
	var zeroItem boolexpr.Item
	if _, err := boolexpr.New(zeroItem); err == nil {
		t.Errorf("expected error for zero-value Item")
	}
	if zeroItem.String() == "||" {
		t.Errorf("zero-value Item should not format as ||")
	}
}

func TestPrecedenceAndMixedOperators(t *testing.T) {
	for _, tc := range []struct {
		in  string
		val any
		out bool
	}{
		// && binds tighter than ||
		{"a || b && c", "a", true},
		{"a || b && c", "b", false},
		{"a || b && c", "bc", true},
		{"a && b || c", "c", true},
		{"a && b || c", "ab", true},
		{"a && b || c", "a", false},
		{"(a || b) && c", "a", false},
		{"(a || b) && c", "ac", true},
		{"!!foo", "foo", true},
		{"!!foo", "bar", false},
	} {
		evalTestCase(t, parse(tc.in), tc.val, tc.out)
	}
}

type countOp struct {
	result bool
	calls  *int
}

func (c countOp) Prepare() (boolexpr.Operand, error) { return c, nil }
func (c countOp) Eval(_ any) bool                    { *c.calls++; return c.result }
func (c countOp) Needs(reflect.Type) bool            { return false }
func (c countOp) Document() string                   { return "count" }
func (c countOp) String() string                     { return "count" }

func TestShortCircuit(t *testing.T) {
	var rhsCalls int
	rhs := boolexpr.NewOperandItem(countOp{result: true, calls: &rhsCalls})
	falseOp := boolexpr.NewOperandItem(countOp{result: false, calls: new(int)})
	trueOp := boolexpr.NewOperandItem(countOp{result: true, calls: new(int)})

	// false && rhs skips rhs
	expr, err := boolexpr.New(falseOp, boolexpr.AND(), rhs)
	if err != nil {
		t.Fatal(err)
	}
	if expr.Eval("x") {
		t.Fatalf("expected false")
	}
	if rhsCalls != 0 {
		t.Fatalf("expected short-circuit on &&, got %d rhs calls", rhsCalls)
	}

	// true || (false && rhs) skips both false and rhs
	rhsCalls = 0
	expr, err = boolexpr.New(trueOp, boolexpr.OR(), boolexpr.LeftBracket(), falseOp, boolexpr.AND(), rhs, boolexpr.RightBracket())
	if err != nil {
		t.Fatal(err)
	}
	if !expr.Eval("x") {
		t.Fatalf("expected true")
	}
	if rhsCalls != 0 {
		t.Fatalf("expected short-circuit on ||, got %d rhs calls", rhsCalls)
	}
}

type staticOp struct{}

func (staticOp) Prepare() (boolexpr.Operand, error) { return staticOp{}, nil }
func (staticOp) Eval(v any) bool                    { s, ok := v.(string); return ok && s == "match" }
func (staticOp) Needs(reflect.Type) bool            { return false }
func (staticOp) Document() string                   { return "static" }
func (staticOp) String() string                     { return "static=match" }

func TestEvalZeroAllocs(t *testing.T) {
	op := boolexpr.NewOperandItem(staticOp{})
	expr, err := boolexpr.New(
		op, boolexpr.AND(), boolexpr.LeftBracket(), op, boolexpr.OR(), boolexpr.NOT(), op, boolexpr.RightBracket(),
	)
	if err != nil {
		t.Fatal(err)
	}
	var val any = "match"
	allocs := testing.AllocsPerRun(100, func() {
		if !expr.Eval(val) {
			t.Fatal("expected match")
		}
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocations per Eval, got %v", allocs)
	}
}

func BenchmarkEval(b *testing.B) {
	expr, err := boolexpr.New(parse(`(foo && (bar || !baz)) || else`)...)
	if err != nil {
		b.Fatal(err)
	}
	var val any = "foobar"
	b.ReportAllocs()
	for b.Loop() {
		if !expr.Eval(val) {
			b.Fatal("expected match")
		}
	}
}

type nameIfc interface {
	Name() string
}

type regexNameOp struct{}

func (rno regexNameOp) Needs(t reflect.Type) bool {
	needs := reflect.TypeFor[nameIfc]()
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

func (rno regexNameOp) Eval(_ any) bool {
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
	if !expr.IsEmpty() {
		t.Errorf("expected IsEmpty() == true for empty expression")
	}
	if expr.Needs(nil) {
		t.Errorf("expected Needs(nil) == false")
	}
	if expr.NeedsType(nil) {
		t.Errorf("expected NeedsType(nil) == false")
	}

	expr, _ = boolexpr.New(boolexpr.NewOperandItem(regexOp{}))
	assert(true, false, false)
	if expr.IsEmpty() {
		t.Errorf("expected IsEmpty() == false for non-empty expression")
	}
	if !expr.NeedsFor[string]() {
		t.Errorf("expected NeedsFor[string] == true")
	}
	if expr.NeedsFor[nameIfc]() {
		t.Errorf("expected NeedsFor[nameIfc] == false")
	}

	expr, err = boolexpr.New(
		boolexpr.NewOperandItem(&regexOp{}),
		boolexpr.OR(),
		boolexpr.NewOperandItem(&regexNameOp{}))

	assert(true, true, true)
	if !expr.NeedsFor[nameIfc]() {
		t.Errorf("expected NeedsFor[nameIfc] == true")
	}

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
