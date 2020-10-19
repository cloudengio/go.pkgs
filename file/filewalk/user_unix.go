// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// +build darwin linux

package filewalk

import (
	"strconv"
	"syscall"
)

func getUserID(sys interface{}) string {
	si, ok := sys.(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return strconv.Itoa(int(si.Uid))
}
