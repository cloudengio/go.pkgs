// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"strings"
)

func isWindowsDrive(p string) (string, bool) {
	if len(p) == 0 {
		return "", false
	}
	drive := p[0]
	if drive >= 'A' && drive <= 'Z' || drive >= 'a' && drive <= 'z' {
		if len(p) >= 2 && p[1] == ':' {
			return string(drive), true
		}
	}
	return "", false
}

func fileURIWindows(p string) *Match {
	if len(p) == 0 {
		return nil
	}
	host, rest, drive := parseFileURI(p)
	if len(drive) == 0 || len(rest) == 0 {
		return nil
	}
	return &Match{
		Scheme:    WindowsFileSystem,
		Separator: '/',
		Host:      host,
		Volume:    drive,
		Path:      rest,
		Key:       rest[2:],
		Local:     true,
	}
}

// WindowsMatcher implements Matcher for Windows filenames. It returns
// WindowsFileSystem for its scheme result.
func WindowsMatcher(p string) *Match {
	if len(p) == 0 {
		return nil
	}
	m := &Match{
		Scheme:    WindowsFileSystem,
		Host:      "",
		Separator: '\\',
		Local:     true,
	}

	if len(p) >= 7 && p[:7] == "file://" {
		return fileURIWindows(p[7:])
	}

	// extended length names
	p = strings.TrimPrefix(p, `\\?`)
	if drive, ok := isWindowsDrive(p); ok {
		m.Volume = drive
		m.Path = p
		m.Key = p[2:]
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
	m.Key = m.Path
	return m
}
