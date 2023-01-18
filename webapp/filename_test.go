// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"testing"

	"cloudeng.io/webapp"
)

func TestSafePaths(t *testing.T) {
	for _, tc := range []struct {
		path string
		err  string
	}{
		{"../y", "contains relative path components"},
		{"./y", "contains relative path components"},
		{"x/../y", "contains relative path components"},
		{"x/./y", "contains relative path components"},
		{"x/?/y", "contains unix reserved characters"},
		{"con", "contains windows reserved characters"},
	} {
		err := webapp.SafePath(tc.path)
		if got, want := err.Error(), tc.err; got != want {
			t.Errorf("%v: got %v, want %v", tc.path, got, want)
		}
	}
}
