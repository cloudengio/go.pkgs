// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

func fileURIUnix(p string) *Match {
	if len(p) == 0 {
		return nil
	}
	host, rest, win := parseFileURI(p)
	if len(win) > 0 || len(rest) == 0 {
		return nil
	}
	return &Match{
		Scheme:    UnixFileSystem,
		Separator: '/',
		Host:      host,
		Path:      rest,
		Key:       rest,
		Local:     true,
	}
}

// UnixMatcher implements Matcher for unix filenames. It returns UnixFileSystem
// for its scheme result. It will match on file://[HOST]/[PATH].
func UnixMatcher(p string) *Match {
	if len(p) >= 7 && p[:7] == "file://" {
		return fileURIUnix(p[7:])
	}
	// Pretty much anything can be a unix filename, even a url.
	return &Match{
		Scheme:    UnixFileSystem,
		Separator: '/',
		Host:      "",
		Path:      p,
		Key:       p,
		Local:     true,
	}
}
