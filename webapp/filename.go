// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"fmt"
	"regexp"
)

var (
	reControl           = regexp.MustCompile("[\u0000-\u001f\u0080-\u009f]")
	reRelative          = regexp.MustCompile(`^\.+`)
	reRelativeComponets = regexp.MustCompile(`\.+`)
	reFilenameLinux     = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	reFilenameWindows   = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)

	rules = []struct {
		re  *regexp.Regexp
		err string
	}{
		{reControl, "contains control characters"},
		{reRelative, "relative path"},
		{reRelativeComponets, "contains relative path components"},
		{reFilenameLinux, "contains unix reserved characters"},
		{reFilenameWindows, "contains windows reserved characters"},
	}
)

func SafePath(path string) error {
	for _, rule := range rules {
		if rule.re.MatchString(path) {
			return fmt.Errorf(rule.err)
		}
	}
	return nil
}
