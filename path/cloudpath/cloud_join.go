// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"strings"
)

// Join will join the supplied components using the supplied separator
// behaviour appropriate for cloud storage paths that do not elide multiple
// contiguous separators. It behaves as follows:
//   - empty components are ignored.
//   - trailing instances of sep are preserved.
//   - separators are added only when not already present as a trailing
//     character in the previous component and leading character in the
//     next component.
//   - a leading separator is ignored/removed if the previous component
//     ended with a separator and the next component starts with a separator.
func Join(sep byte, components []string) string {
	size := 0
	for _, c := range components {
		size += len(c)
	}
	if size == 0 {
		return ""
	}
	joined := make([]byte, 0, size+len(components)-1)
	for _, c := range components {
		if len(c) == 0 {
			continue
		}
		if lj := len(joined); lj > 0 {
			ts := joined[lj-1] == sep
			ls := c[0] == sep
			if !ts && !ls {
				joined = append(joined, sep)
			}
			if ls && ts {
				c = c[1:]
			}
		}
		joined = append(joined, c...)
	}
	return string(joined)
}

func lastIdx(scheme string, sep byte, path string) int {
	idx := strings.LastIndexByte(path, sep)
	if idx < 0 {
		return -1
	}
	if idx <= len(scheme) && strings.HasPrefix(path, scheme) {
		return -2
	}
	return idx
}

// Base is like path.Base but for cloud storage paths which may include
// a scheme (eg. s3://). It does not support URI host names, parameters etc.
// In particular:
//   - the scheme parameter should include the trailing :// or be the
//     empty string.
//   - a trailing seperator means that the path is a prefix with
//     an empty base and hence Base returns "".
//   - the returned basename never includes the supplied scheme.
func Base(scheme string, seperator byte, path string) string {
	p := strings.TrimPrefix(path, scheme)
	idx := strings.LastIndexByte(p, seperator)
	if idx < 0 {
		return path
	}
	return p[idx+1:]
}

// Prefix is like path.Dir but for cloud storage paths which may include
// a scheme (eg. s3:///). It does not support URI host names, parameters etc.
// In particular:
//   - the scheme parameter should include the trailing :// or be the
//     empty string.
//   - the returned prefix never includes the supplied scheme.
//   - the returned prefix never includes a trailing seperator.
func Prefix(scheme string, seperator byte, path string) string {
	p := strings.TrimPrefix(path, scheme)
	if len(p) > 0 && p[len(p)-1] == seperator {
		p = p[:len(p)-1] // already a prefix
		return p
	}
	if idx := strings.LastIndexByte(p, seperator); idx >= 0 {
		return p[:idx]
	}
	return ""
}
