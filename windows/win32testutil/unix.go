// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package win32testutil provides functionality for use in tests that
// run on windows. These generally cover differences in functionality in the
// go standard library when running over windows as apposed to unix like
// systems.

//go:build !windows

package win32testutil

import (
	"os"
)

// MakeInaccessibleToOwner makes path inaccessible to its owner.
func MakeInaccessibleToOwner(path string) error {
	return os.Chmod(path, 000)
}

// MakeAcessibleToOwner makes path ccessible to its owner.
func MakeAccessibleToOwner(path string) error {
	return os.Chmod(path, 0777)
}
