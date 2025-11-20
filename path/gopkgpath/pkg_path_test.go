// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package gopkgpath_test

import (
	"testing"

	"cloudeng.io/path/gopkgpath"
)

func TestPackagePath(t *testing.T) {
	p, err := gopkgpath.Caller()
	if err != nil {
		t.Errorf("%v", err)
	}
	if got, want := p, "cloudeng.io/path/gopkgpath"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for i, tc := range []struct {
		depth  int
		result string
	}{
		{0, "cloudeng.io/path/gopkgpath"},
		{1, "std/testing"},
		{2, "std/runtime"},
	} {
		p, err := gopkgpath.CallerDepth(tc.depth)
		if err != nil {
			t.Errorf("%v: %v", i, err)
		}
		if got, want := p, tc.result; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	type definedType int
	for i, tc := range []struct {
		variable any
		result   string
	}{
		{3, ""},
		{definedType(3), "cloudeng.io/path/gopkgpath_test"},
		{gopkgpath.Caller, ""},
	} {
		if got, want := gopkgpath.Type(tc.variable), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}
