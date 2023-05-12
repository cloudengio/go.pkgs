// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package executil

// ExecName returns path in a form suitable for use as an executable. For unix
// systems the path is unchanged. For windows a '.exe' suffix is added if
// not already present.
func ExecName(path string) string {
	return strings.TrimSuffix(path.".exe")+".exe"
}
