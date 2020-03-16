// Copyright 2020 cloudeng LLC. All rights reserved.
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
		slice              []string
		absolute, complete bool
		joined             string
	}{
		{"", nil, false, false, ""},
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
		{"", nil, false, false, ""},
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
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cloudpath.IsAbsolute(path), tc.absolute; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cloudpath.IsFilepath(path), tc.complete; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cloudpath.Join(filepath.Separator, path...), tc.joined; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}

type matcherTestSpec struct {
	input                      string
	scheme, host, volume, path string
	separator                  rune
	parameters                 map[string][]string
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
		scheme := ms.Scheme(tc.input)
		volume := ms.Volume(tc.input)
		host := ms.Host(tc.input)
		path, sep := ms.Path(tc.input)
		parameters := ms.Parameters(tc.input)
		if got, want := scheme, tc.scheme; got != want {
			errs.Append(fmt.Errorf("%v: %v: scheme: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := volume, tc.volume; got != want {
			errs.Append(fmt.Errorf("%v: %v: volume: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := host, tc.host; got != want {
			errs.Append(fmt.Errorf("%v: %v: host: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := path, tc.path; got != want {
			errs.Append(fmt.Errorf("%v: %v: path: got %v, want %v", i, tc.input, got, want))
		}
		if got, want := sep, tc.separator; got != want {
			errs.Append(fmt.Errorf("%v: %v: separator: got %v, want %v", i, tc.input, got, want))
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
		if fn(tc) != nil {
			errs.Append(fmt.Errorf("%v: unexpected match for %q", i, tc))
		}
	}
	return errs.Err()
}

func TestLongestPrefixSuffix(t *testing.T) {
	ij := func(paths ...string) [][]string {
		r := [][]string{}
		for _, p := range paths {
			r = append(r, cloudpath.Split(p, '/'))
		}
		return r
	}
	oj := func(path string) []string {
		if len(path) == 0 {
			return []string{}
		}
		return cloudpath.Split(path, '/')
	}
	for i, tc := range []struct {
		input          [][]string
		prefix, suffix []string
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
		if got, want := cloudpath.LongestCommonPrefix(tc.input...), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: prefix: got %#v, want %#v", i, got, want)
		}
		if got, want := cloudpath.LongestCommonSuffix(tc.input...), tc.suffix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: suffix: got %#v, want %#v", i, got, want)
		}
	}
}

func TestPrefix(t *testing.T) {
	sl := func(p string) []string {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input   []string
		prefix  []string
		trimmed []string
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
		input  []string
		prefix []string
	}{
		{sl(""), sl("")},
		{sl("a"), sl("")},
		{sl("a/"), sl("a/")},
		{sl("/a/"), sl("/a/")},
		{sl("/a/b/c"), sl("/a/b/")},
		{sl("/a/b/c/"), sl("/a/b/c/")},
	} {
		if got, want := cloudpath.Prefix(tc.input), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input  []string
		prefix []string
	}{
		{sl(""), sl("")},
		{sl("a"), sl("a/")},
		{sl("a/"), sl("a/")},
	} {
		if got, want := cloudpath.AsPrefix(tc.input...), tc.prefix; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}

}

func TestSuffix(t *testing.T) {
	sl := func(p string) []string {
		return cloudpath.Split(p, '/')
	}
	for i, tc := range []struct {
		input   []string
		suffix  []string
		trimmed []string
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
		if got, want := cloudpath.HasSuffix(tc.input, tc.suffix), tc.has; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := cloudpath.TrimSuffix(tc.input, tc.suffix), tc.trimmed; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
	for i, tc := range []struct {
		input []string
		base  string
	}{
		{sl(""), ""},
		{sl("a"), "a"},
		{sl("/a"), "a"},
		{sl("/a/"), ""},
	} {
		if got, want := cloudpath.Base(tc.input), tc.base; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %#v, want %#v", i, got, want)
		}
	}
}
