// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"errors"
	"regexp"
)

var ( // https://github.com/flytam/filenamify
	reControl = regexp.MustCompile("[\u0000-\u001f\u0080-\u009f]")

	// https://github.com/sindresorhus/filename-reserved-regex/blob/master/index.js
	reFilenameLinux   = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	reFilenameWindows = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)

	// one or more . followed by a slash, or three or more dots
	reRelativeComponets = regexp.MustCompile(`\.+/|\.\.\.+`)

	rules = []struct {
		matcher *regexp.Regexp
		err     string
	}{
		{reControl, "contains control characters"},
		{reRelativeComponets, "contains relative path components"},
		{reFilenameLinux, "contains unix reserved characters"},
		{reFilenameWindows, "contains windows reserved characters"},
	}
)

// SafePath checks if the given path is safe for use as a filename
// screening for control characters, windows device names, relative paths,
// paths (eg. a/b is not allowed) etc.
func SafePath(path string) error {
	for _, rule := range rules {
		if rule.matcher.MatchString(path) {
			return errors.New(rule.err)
		}
	}
	return nil
}
