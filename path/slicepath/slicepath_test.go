// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package slicepath_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"cloudeng.io/path/slicepath"
)

const sep = string(filepath.Separator)

func sliceFromComponents(components ...string) []string {
	return components
}

func joinComponents(components ...string) string {
	out := ""
	for _, c := range components {
		out += c
	}
	return out
}

func TestSplit(t *testing.T) {
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
		path := slicepath.Split(tc.path, filepath.Separator)
		if got, want := path, tc.slice; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := slicepath.IsAbs(path), tc.absolute; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := slicepath.IsComplete(path), tc.complete; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := slicepath.Join(filepath.Separator, path...), tc.joined; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}

// test
// Dir, Filename, Volumename

// Test scheme, path, parameters.

/*
func TestDirBase(t *testing.T) {
	s := sliceFromStrings
	j := joinStrings
	for i, tc := range []struct {
		path  string
		dir []string
		base string
	}{
	}
}
*/
