// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package userid

import (
	"strings"
)

// ParseWindowsUser returns the domain and user component of a windows
// username (domain\user).
func ParseWindowsUser(u string) (domain, user string) {
	idx := strings.LastIndex(u, `\`)
	if idx < 0 {
		user = u
		return
	}
	domain = u[:idx]
	user = u[idx+1:]
	return
}
