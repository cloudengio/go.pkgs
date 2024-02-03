// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"slices"
	"testing"

	"cloudeng.io/path/cloudpath"
)

func TestJoin(t *testing.T) {
	j := func(p ...string) []string {
		return p
	}
	var sep byte = '/'
	for _, tc := range []struct {
		in  []string
		out string
	}{
		{j(), ""},
		{j(""), ""},
		{j("", ""), ""},
		{j("", "b"), "b"},
		{j("/", ""), "/"},
		{j("a", ""), "a"},
		{j("a", "b"), "a/b"},
		{j("a", "b/"), "a/b/"},
		{j("a/", "b"), "a/b"},
		{j("a", "/b"), "a/b"},
		{j("a/", "/b"), "a/b"},
		{j("a", "", "b"), "a/b"},
		{j("a/", "", "b"), "a/b"},
		{j("a", "", "/b"), "a/b"},
		{j("a/", "", "/b"), "a/b"},
		{j("a/", "", "/b/"), "a/b/"},
		{j("a/", ""), "a/"},
	} {
		if got, want := cloudpath.Join(sep, tc.in), tc.out; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestBasePrefix(t *testing.T) {
	var sep byte = '/'
	for _, scheme := range []string{"", "http://", "s3://"} {
		for _, tc := range []struct {
			in           string
			prefix, base string
		}{
			{"", "", ""},
			{"a", "", "a"},
			{"a/", "a", ""},
			{"a/b", "a", "b"},
			{"a/b/", "a/b", ""},
			{"a/b/c", "a/b", "c"},
		} {
			prefix := cloudpath.Prefix(scheme, sep, tc.in)
			basename := cloudpath.Base(scheme, sep, tc.in)
			if got, want := prefix, tc.prefix; got != want {
				t.Errorf("%q (%v): got %v, want %v", tc.in, scheme, got, want)
			}
			if got, want := basename, tc.base; got != want {
				t.Errorf("%q (%v): got %v, want %v", tc.in, scheme, got, want)
			}

			jp := prefix
			if len(jp) > 0 {
				jp += string(sep)
			}
			if got, want := cloudpath.Join(sep, []string{jp, basename}), tc.in; got != want {
				t.Errorf("%q (%v) join(%q %q): got %v, want %v", tc.in, scheme, jp, basename, got, want)
			}
		}
	}

	parents := []string{}
	children := []string{}
	start := "s3://bucket/a/b/c/d/object"
	p := start
	for {
		c := cloudpath.Base("s3://", '/', p)
		p = cloudpath.Prefix("s3://", '/', p)
		parents = append(parents, p)
		children = append(children, c)
		if len(p) == 0 {
			break
		}
	}
	if got, want := parents, []string{
		"bucket/a/b/c/d",
		"bucket/a/b/c",
		"bucket/a/b",
		"bucket/a",
		"bucket",
		"",
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := children, []string{
		"object",
		"d",
		"c",
		"b",
		"a",
		"bucket",
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
