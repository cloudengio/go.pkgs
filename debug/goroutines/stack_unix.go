// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !windows

package goroutines

import (
	"fmt"
	"regexp"
	"strconv"
)

var stackFileRE = regexp.MustCompile(`^\s+([^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+))?$`)

func parseFileLine(input []byte) (file string, line, offset int64, err error) {
	matches := stackFileRE.FindSubmatch(input)
	if len(matches) < 4 {
		err = fmt.Errorf("could not parse file reference from %q", string(input))
		return
	}
	file = string(matches[1])
	line, err = strconv.ParseInt(string(matches[2]), 10, 64)
	if err != nil {
		return
	}
	if len(matches[3]) > 0 {
		offset, err = strconv.ParseInt(string(matches[3]), 16, 64)
		if err != nil {
			return
		}
	}
	return
}
