// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin || linux
// +build darwin linux

package filewalk

import (
	"strconv"
	"syscall"
)

func getUserAndGroupID(sys interface{}) (string, string) {
	si, ok := sys.(*syscall.Stat_t)
	if !ok {
		return "", ""
	}
	return strconv.Itoa(int(si.Uid)), strconv.Itoa(int(si.Gid))
}
