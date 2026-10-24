// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !unix

package cmdconfig

// withXDGConfigDir is a no-op on systems that are not unix: os.UserConfigDir
// already reports the only user configuration directory they define.
func withXDGConfigDir(dirs []string, _ string) []string {
	return dirs
}
