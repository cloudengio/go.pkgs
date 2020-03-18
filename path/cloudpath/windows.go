// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"strings"
)

func isWindowsDrive(p string) (string, bool) {
	drive := p[0]
	if drive >= 'A' && drive <= 'Z' || drive >= 'a' && drive <= 'z' {
		if len(p) >= 2 && p[1] == ':' {
			return string(drive), true
		}
	}
	return "", false
}

// WindowsMatcher implements Matcher for Windows filenames. It returns
// WindowsFileSystem for its scheme result.
func WindowsMatcher(p string) *Match {
	if len(p) == 0 {
		return nil
	}
	m := &Match{
		Scheme:    WindowsFileSystem,
		Host:      "localhost",
		Separator: '\\',
		Local:     true,
	}

	// Handle file:// uris.
	if u, err := url.Parse(p); err == nil && u.Scheme == "file" {
		wp := strings.TrimPrefix(u.Path, "/")
		if drive, ok := isWindowsDrive(wp); ok && len(wp) > 2 && wp[2] == '/' {
			m.Volume = drive
			m.Separator = '/'
			m.Path = wp
			return m
		}
		return nil
	}

	// extended length names
	p = strings.TrimPrefix(p, `\\?`)
	if drive, ok := isWindowsDrive(p); ok {
		m.Volume = drive
		m.Path = p
		return m
	}
	if len(p) < 2 || strings.Index(p, `\`) < -1 {
		// no backslashes so there's no way to tell.
		return nil
	}
	if !strings.HasPrefix(p, `\\`) {
		return nil
	}
	// UNC format: \\server\share\path
	parts := strings.Split(strings.TrimSuffix(p[2:], `\`), `\`)
	switch len(parts) {
	default:
		m.Path = `\` + strings.Join(parts[2:], `\`)
		fallthrough
	case 2:
		m.Volume = parts[1]
		fallthrough
	case 1:
		m.Host = parts[0]
	}
	return m
}
