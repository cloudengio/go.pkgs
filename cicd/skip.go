// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import "runtime"

type TestingT interface {
	Helper()
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
	Name() string
}

// SkipIf skips t if skipping is true, using msg as the skip message.
func SkipIf(t TestingT, msg string, skipping bool) {
	t.Helper()
	if skipping {
		t.Skipf("%s", msg)
	}
}

// SkipMacOS skips t if running on macOS.
func SkipMacOS(t TestingT) {
	t.Helper()
	SkipIf(t, "skipping on macOS", runtime.GOOS == "darwin")
}

// SkipLinux skips t if running on Linux.
func SkipLinux(t TestingT) {
	t.Helper()
	SkipIf(t, "skipping on Linux", runtime.GOOS == "linux")
}

// SkipWindows skips t if running on Windows.
func SkipWindows(t TestingT) {
	t.Helper()
	SkipIf(t, "skipping on Windows", runtime.GOOS == "windows")
}
