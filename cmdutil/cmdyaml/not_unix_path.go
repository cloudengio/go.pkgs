// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !unix

package cmdyaml

func withSystemConfigDir(path, appDir string) string {
	return path
}
