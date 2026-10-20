// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package cmdyaml

import (
	"os"
	"path/filepath"
	"strings"
)

func withSystemConfigDir(path, appDir string) string {
	if home := os.Getenv("HOME"); len(home) > 0 {
		path = strings.Join([]string{path, filepath.Join(home, ".config", appDir)}, string(os.PathListSeparator))
	}
	return path
}
