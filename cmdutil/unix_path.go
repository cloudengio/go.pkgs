// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package cmdutil

import (
	"os"
	"path/filepath"
)

// withXDGConfigDir appends $HOME/.config/appDir, the XDG Base Directory
// default for user configuration, to dirs. It is appended on every unix
// system, including macOS where os.UserConfigDir reports
// $HOME/Library/Application Support instead and so does not cover it.
func withXDGConfigDir(dirs []string, appDir string) []string {
	if home := os.Getenv("HOME"); len(home) > 0 {
		return append(dirs, filepath.Join(home, ".config", appDir))
	}
	return dirs
}
