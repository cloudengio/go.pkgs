// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"testing"

	"cloudeng.io/path/cloudpath"
)

func TestUnix(t *testing.T) {
	data := []matcherTestSpec{
		{
			"/",
			cloudpath.UnixFileSystem, "", "", "", "/", "/", '/', nil,
		},
		{
			"./",
			cloudpath.UnixFileSystem, "", "", "", "./", "./", '/', nil,
		},
		{
			".",
			cloudpath.UnixFileSystem, "", "", "", ".", ".", '/', nil,
		},
		{
			"..",
			cloudpath.UnixFileSystem, "", "", "", "..", "..", '/', nil,
		},
		{
			"/a/b",
			cloudpath.UnixFileSystem, "", "", "", "/a/b", "/a/b", '/', nil,
		},
		{
			"file:///a/b/c/",
			cloudpath.UnixFileSystem, "", "", "", "/a/b/c/", "/a/b/c/", '/', nil,
		},
		{
			"file://host/",
			cloudpath.UnixFileSystem, "host", "", "", "/", "/", '/', nil,
		},
		{
			"file://host/a/b/c/",
			cloudpath.UnixFileSystem, "host", "", "", "/a/b/c/", "/a/b/c/", '/', nil,
		},
		{
			"file:",
			cloudpath.UnixFileSystem, "", "", "", "file:", "file:", '/', nil,
		},
		{
			"file://a:/c",
			cloudpath.UnixFileSystem, "a:", "", "", "/c", "/c", '/', nil,
		},
	}
	if err := testMatcher(cloudpath.UnixMatcher, data); err != nil {
		t.Errorf("%v", err)
	}
	if err := testNoMatch(cloudpath.UnixMatcher, []string{
		"file://",
		"file://..",
		"file:///a:/c",
	}); err != nil {
		t.Errorf("%v", err)
	}

	for _, d := range data {
		if got, want := cloudpath.IsLocal(d.input), true; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
