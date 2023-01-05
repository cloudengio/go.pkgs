// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines

import "testing"

func TestWindowsParsing(t *testing.T) {
	// The stack information on windows is less defined than on other systems.
	// All of the following have been observed on circleci windows and local
	// VMs.

	for _, tc := range []struct {
		input        string
		file         string
		line, offset int64
	}{
		{"C:/Users/circleci/go/pkg/mod/cloudeng.io/debug@v0.0.0-20230102232444-1226f47ebbea/goroutines/stack.go:36 +0x85", "C:/Users/circleci/go/pkg/mod/cloudeng.io/debug@v0.0.0-20230102232444-1226f47ebbea/goroutines/stack.go", 36, 0x85},
		{"_testmain.go:49 +0x1e8", "_testmain.go", 49, 0x1e8},
		{"C:/Program Files/Go/src/testing/testing.go:1446", "C:/Program Files/Go/src/testing/testing.go", 1446, 0},
	} {
		input := " " + tc.input
		file, line, offset, err := windowsParseFileLine([]byte(input))
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if got, want := file, tc.file; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := line, tc.line; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := offset, tc.offset; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
}
