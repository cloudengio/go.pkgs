// Copyright 2020 cloudeng LLC. All rights reserved.
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
			cloudpath.UnixFileSystem, "localhost", "", "/", '/', nil,
		},
		{
			"/a/b",
			cloudpath.UnixFileSystem, "localhost", "", "/a/b", '/', nil,
		},
	}
	if err := testMatcher(cloudpath.UnixMatcher, data); err != nil {
		t.Errorf("%v", err)
	}
	if err := testNoMatch(cloudpath.UnixMatcher, []string{
		"",
	}); err != nil {
		t.Errorf("%v", err)
	}
}
