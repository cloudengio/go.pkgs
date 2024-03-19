// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import "testing"

func TestValidFilename(t *testing.T) {
	for _, name := range []string{
		formatFilename(0, ""),
		formatFilename(1, "something"),
		"0123456790.chk",
	} {
		if !isValidFilename(name) {
			t.Errorf("expected %v to be valid", name)
		}
	}

	for _, name := range []string{
		"", "0123456.chk", "012345", "xx.chk",
	} {
		if isValidFilename(name) {
			t.Errorf("expected %v to be invalid", name)
		}
	}
}
