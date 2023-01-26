// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"cloudeng.io/errors"
	"cloudeng.io/path/cloudpath"
)

func TestSplitJoin(t *testing.T) {
	s := sliceFromComponents
	j := joinComponents
	for i, tc := range []struct {
		path               string
		slice              cloudpath.T
		absolute, filepath bool
		joined             string
	}{
		{"", cloudpath.T{}, false, false, ""},
		{j(sep), s("", ""), true, false, sep},
		{j(sep, "a"), s("", "a"), true, true, j(sep, "a")},
		{j(sep, "a", sep), s("", "a", ""), true, false, j(sep, "a", sep)},
		{"a", s("a"), false, true, "a"},
		{j("a", sep), s("a", ""), false, false, j("a", sep)},
		{j(sep, "a", sep, "bc"), s("", "a", "bc"), true, true, j(sep, "a", sep, "bc")},
		{j(sep, "a", sep, "bc", sep), s("", "a", "bc", ""), true, false, j(sep, "a", sep, "bc", sep)},
		{j("a", sep, "bc"), s("a", "bc"), false, true, j("a", sep, "bc")},
		{j("a", sep, "bc", sep), s("a", "bc", ""), false, false, j("a", sep, "bc", sep)},

		// Add in some repeat separators.
		{"", cloudpath.T{}, false, false, ""},
		{j(sep, sep), s("", ""), true, false, sep},
		{j(sep, sep, "a"), s("", "a"), true, true, j(sep, "a")},
		{j(sep, sep, "a", sep), s("", "a", ""), true, false, j(sep, "a", sep)},
		{"a", s("a"), false, true, "a"},
		{j(sep, sep, "a", sep, sep, sep), s("", "a", ""), true, false, j(sep, "a", sep)},
		{"a", s("a"), false, true, "a"},
		{j("a", sep, sep), s("a", ""), false, false, j("a", sep)},
		{j(sep, sep, "a", sep, sep, "bc"), s("", "a", "bc"), true, true, j(sep, "a", sep, "bc")},
		{j(sep, "a", sep, sep, "bc", sep), s("", "a", "bc", ""), true, false, j(sep, "a", sep, "bc", sep)},
		{j("a", sep, sep, "bc"), s("a", "bc"), false, true, j("a", sep, "bc")},
		{j("a", sep, "bc", sep, sep), s("a", "bc", ""), false, false, j("a", sep, "bc", sep)},
	} {
		path := cloudpath.Split(tc.path, filepath.Separator)
		if got, want := path, tc.slice; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := path.IsAbsolute(), tc.absolute; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := path.IsFilepath(), tc.filepath; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := path.Join(filepath.Separator), tc.joined; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	if got, want := cloudpath.Split("/", '/').IsRoot(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cloudpath.Split("/a", '/').IsRoot(), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type matcherTestSpec struct {
	input                                   string
	scheme, host, region, volume, path, key string
	separator                               rune
	parameters                              map[string][]string
}

func testMatcher(fn cloudpath.Matcher, testSpecs []matcherTestSpec) error {
	return testMatcherSpec([]cloudpath.Matcher{fn}, testSpecs)
}

func testMatcherSpec(ms cloudpath.MatcherSpec, testSpecs []matcherTestSpec) error {
	errs := &errors.M{}
	for i, tc := range testSpecs {
		if tc.parameters == nil {
			tc.parameters = map[string][]string{}
		}
		testSpecs[i] = tc
	}
	for i, tc := range testSpecs {
		match := ms.Match(tc.input)
		if len(match.Matched) == 0 {
			errs.Append(fmt.Errorf("%v: %v: no match", i, tc.input))
			continue
		}
		scheme := match.Scheme
		volume := match.Volume
		region := match.Region
		host := match.Host
		key, ksep := match.Key, match.Separator
		path := match.Path
		parameters := match.Parameters
		if parameters == nil {
			parameters = map[string][]string{}
		}
		if got, want := match.Matched, tc.input; got != want {
			errs.Append(fmt.Errorf("%v: %v: matched: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := scheme, tc.scheme; got != want {
			errs.Append(fmt.Errorf("%v: %v: scheme: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := volume, tc.volume; got != want {
			errs.Append(fmt.Errorf("%v: %v: volume: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := host, tc.host; got != want {
			errs.Append(fmt.Errorf("%v: %v: host: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := region, tc.region; got != want {
			errs.Append(fmt.Errorf("%v: %v: region: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := path, tc.path; got != want {
			errs.Append(fmt.Errorf("%v: %v: path: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := key, tc.key; got != want {
			errs.Append(fmt.Errorf("%v: %v: key: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := ksep, tc.separator; got != want {
			errs.Append(fmt.Errorf("%v: %v: key separator: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := parameters, tc.parameters; !reflect.DeepEqual(got, want) {
			errs.Append(fmt.Errorf("%v: %v: parameters: got %v, want %v", i, tc.input, got, want))
		}
	}
	return errs.Err()
}

func testNoMatch(fn cloudpath.Matcher, testCases []string) error {
	errs := &errors.M{}
	for i, tc := range testCases {
		if m := fn(tc); len(m.Matched) > 0 {
			errs.Append(fmt.Errorf("%v: unexpected match for %q: %v", i, tc, m.Scheme))
		}
	}
	return errs.Err()
}

func TestLongestPrefixSuffix(t *testing.T) {
	ij := func(paths ...string) []cloudpath.T {
		r := []cloudpath.T{}
		for _, p := range paths {
			r = append(r, cloudpath.Split(p, '/'))
		}
		return r
	}
	oj := func(path string) cloudpath.T {
		if len(path) == 0 {
			return cloudpath.T{}
		}
		return cloudpath.Split(path, '/')
	}
	for i, tc := range []struct {
		input          []cloudpath.T
		prefix, suffix cloudpath.T
	}{
		{nil, oj(""), oj("")},
		{ij("", ""), oj(""), oj("")},
		{ij("a"), oj("a"), oj("a")},
		{ij("a", ""), oj(""), oj("")},
		{ij("a", "aa"), oj(""), oj("")},
		{ij("a/aa", "aa/aaa"), oj(""), oj("")},
		{ij("a/b", "a"), oj("a"), oj("")},
		{ij("a/b", "a/"), oj("a/"), oj("")},
		{ij("a/b/c", "a/b"), oj("a/b"), oj("")},
		{ij("a/b/c/", "a/b/"), oj("a/b/"), oj("")},
		{ij("/a/b/c/", "/a/b/"), oj("/a/b/"), oj("")},
		{ij("/a/b/c/x/y", "/a/b/x/y"), oj("/a/b/"), oj("x/y")},
		{ij("/a/b/c/x/y/", "/a/b/x/y/"), oj("/a/b/"), oj("x/y/")},
		{ij("/a/b/c/x/y/", "/a/b/x/y/", "/a/b/z/x/y/"), oj("/a/b/"), oj("x/y/")},
		{ij("/a/b/c/x/y/", "/a/b/x/y/", "/a/b/z/A/y/"), oj("/a/b/"), oj("y/")},
	} {
		if got, want := cloudpath.LongestCommonPrefix(tc.input), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: prefix: got %#v, want %#v", i, got, want)
		}
		if got, want := cloudpath.LongestCommonSuffix(tc.input), tc.suffix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: suffix: got %#v, want %#v", i, got, want)
		}
	}
}

func TestPrefix(t *testing.T) {
	sl := func(p string) cloudpath.T {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input   cloudpath.T
		prefix  cloudpath.T
		trimmed cloudpath.T
		has     bool
	}{
		{sl(""), sl(""), sl(""), true},
		{sl(""), sl("a"), sl(""), false},
		{sl("a"), sl(""), sl("a"), true},
		{sl("a"), sl("b"), sl("a"), false},

		{sl("a"), sl("a"), sl(""), true},
		{sl("a"), sl("/a"), sl("a"), false},
		{sl("a"), sl("a/"), sl("a"), false},
		{sl("/a"), sl("a"), sl("/a"), false},
		{sl("/a"), sl("/a"), sl(""), true},
		{sl("/a"), sl("a/"), sl("/a"), false},

		{sl("a/b"), sl("/a"), sl("a/b"), false},
		{sl("a/b"), sl("a"), sl("/b"), true},
		{sl("a/b"), sl("a/"), sl("b"), true},
		{sl("a/b/"), sl("/a"), sl("a/b/"), false},
		{sl("a/b/"), sl("a"), sl("/b/"), true},
		{sl("a/b/"), sl("a/"), sl("b/"), true},

		{sl("a/b/c"), sl("/a"), sl("a/b/c"), false},
		{sl("a/b/c"), sl("a"), sl("/b/c"), true},
		{sl("a/b/c"), sl("a/"), sl("b/c"), true},
		{sl("a/b/c/"), sl("/a"), sl("a/b/c/"), false},
		{sl("a/b/c/"), sl("a"), sl("/b/c/"), true},
		{sl("a/b/c/"), sl("a/"), sl("b/c/"), true},

		{sl("a/b/c"), sl("/a/b"), sl("a/b/c"), false},
		{sl("a/b/c"), sl("a/b"), sl("/c"), true},
		{sl("a/b/c"), sl("a/b/"), sl("c"), true},
		{sl("a/b/c/"), sl("/a/b"), sl("a/b/c/"), false},
		{sl("a/b/c/"), sl("a/b"), sl("/c/"), true},
		{sl("a/b/c/"), sl("a/b"), sl("/c/"), true},

		{sl("/a/b/c"), sl("/a"), sl("/b/c"), true},
		{sl("/a/b/c"), sl("/a/"), sl("b/c"), true},
		{sl("/a/b/c"), sl("/a/b"), sl("/c"), true},
		{sl("/a/b/c"), sl("/a/b/"), sl("c"), true},
	} {
		if got, want := cloudpath.HasPrefix(tc.input, tc.prefix), tc.has; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cloudpath.TrimPrefix(tc.input, tc.prefix), tc.trimmed; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input  cloudpath.T
		prefix cloudpath.T
	}{
		{sl(""), sl("")},
		{sl("a"), sl("")},
		{sl("a/"), sl("a/")},
		{sl("/a/"), sl("/a/")},
		{sl("/a/b/c"), sl("/a/b/")},
		{sl("/a/b/c/"), sl("/a/b/c/")},
	} {
		if got, want := tc.input.Prefix(), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input    cloudpath.T
		prefix   cloudpath.T
		filepath cloudpath.T
	}{
		{sl(""), sl(""), sl("")},
		{sl("a"), sl("a/"), sl("a")},
		{sl("a/"), sl("a/"), sl("a")},
	} {
		if got, want := tc.input.AsPrefix(), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := tc.input.AsFilepath(), tc.filepath; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
}

func TestSuffix(t *testing.T) {
	sl := func(p string) cloudpath.T {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input   cloudpath.T
		suffix  cloudpath.T
		trimmed cloudpath.T
		has     bool
	}{
		{sl(""), sl(""), sl(""), true},
		{sl(""), sl("a"), sl(""), false},
		{sl("a"), sl(""), sl("a"), true},
		{sl("a"), sl("b"), sl("a"), false},
		{sl("a/b"), sl("a/b"), sl(""), true},
		{sl("a/b"), sl("a/b/"), sl("a/b"), false},
		{sl("a/b/"), sl("a/b"), sl("a/b/"), false},
		{sl("a/b/"), sl("a/b/"), sl(""), true},
		{sl("/a/b"), sl("a/b"), sl("/"), true},
		{sl("/a/b"), sl("a/b/"), sl("/a/b"), false},
		{sl("/a/b/"), sl("a/b"), sl("/a/b/"), false},
		{sl("/a/b/"), sl("a/b/"), sl("/"), true},
	} {
		if got, want := tc.input.HasSuffix(tc.suffix), tc.has; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := tc.input.TrimSuffix(tc.suffix), tc.trimmed; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input cloudpath.T
		base  string
	}{
		{sl(""), ""},
		{sl("a"), "a"},
		{sl("/a"), "a"},
		{sl("/a/"), ""},
	} {
		if got, want := tc.input.Base(), tc.base; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
}

func TestPop(t *testing.T) {
	sl := func(p string) cloudpath.T {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input     cloudpath.T
		remainder cloudpath.T
		popped    string
	}{
		{sl(""), sl(""), ""},
		{sl("/"), sl("/"), ""},
		{sl("./"), sl(""), "."},
		{sl("a"), sl(""), "a"},
		{sl("a/b"), sl("a/"), "b"},
		{sl("a/b/c"), sl("a/b/"), "c"},
		{sl("a/"), sl(""), "a"},
		{sl("a/b/"), sl("a/"), "b"},
		{sl("a/b/c/"), sl("a/b/"), "c"},
		{sl("/a/"), sl("/"), "a"},
		{sl("/a/b/"), sl("/a/"), "b"},
		{sl("/a/b/c/"), sl("/a/b/"), "c"},
	} {
		rem, popped := tc.input.Pop()
		if got, want := rem, tc.remainder; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := popped, tc.popped; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := rem.IsFilepath(), false; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}

func TestPush(t *testing.T) {
	sl := func(p string) cloudpath.T {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input  cloudpath.T
		push   string
		pushed cloudpath.T
	}{

		{sl(""), "b", sl("b")},
		{sl("/"), "b", sl("/b")},
		{sl("a"), "b", sl("a/b")},
		{sl("/a"), "b", sl("/a/b")},
		{sl("a/"), "b", sl("a/b")},
		{sl("/a/"), "b", sl("/a/b")},
	} {
		pushed := tc.input.Push(tc.push)
		if got, want := pushed, tc.pushed; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := pushed.IsFilepath(), true; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input  cloudpath.T
		pushed cloudpath.T
	}{
		{sl(""), sl("")},
		{sl("/"), sl("/")},
		{sl("a"), sl("a")},
		{sl("/a"), sl("/a")},
		{sl("a/"), sl("a")},
		{sl("/a/"), sl("/a")},
	} {
		pushed := tc.input.Push("")
		if got, want := pushed, tc.pushed; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
		if got, want := pushed, tc.input.AsFilepath(); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
}

func TestString(t *testing.T) {
	for i, tc := range []struct {
		input  string
		output string
	}{
		{"", ""},
		{"+", "/"},
		{"+a", "/a"},
		{"+a+bb", "/a/bb"},
		{"+a+bb+", "/a/bb/"},
	} {
		if got, want := cloudpath.Split(tc.input, '+').String(), tc.output; got != want {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
}
