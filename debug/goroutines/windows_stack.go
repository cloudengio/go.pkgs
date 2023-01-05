// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	windowsStackFileVolumeRE   = regexp.MustCompile(`^\s+([A-Za-z]+:[^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+)?)`)
	windowsStackFileRE         = regexp.MustCompile(`^\s+([^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+))?$`)
	windowsStackFileNoOffsetRE = regexp.MustCompile(`^\s+([A-Za-z]+:[^:]+):(\d+)$`)
)

func windowsParseNoOffset(matches [][]byte) (file string, line int64, err error) {
	file = string(matches[1])
	line, err = strconv.ParseInt(string(matches[2]), 10, 64)
	return
}

func windowsParseAll(matches [][]byte) (file string, line, offset int64, err error) {
	file = string(matches[1])
	line, err = strconv.ParseInt(string(matches[2]), 10, 64)
	if err != nil {
		return
	}
	offset, err = strconv.ParseInt(string(matches[3]), 16, 64)
	return
}

func windowsParseFileLine(input []byte) (file string, line, offset int64, err error) {
	matches := windowsStackFileVolumeRE.FindSubmatch(input)
	if len(matches) == 0 {
		matches = windowsStackFileRE.FindSubmatch(input)
	}
	if len(matches) == 4 {
		file, line, offset, err = windowsParseAll(matches)
		return
	}
	matches = windowsStackFileNoOffsetRE.FindSubmatch(input)
	if len(matches) < 3 {
		err = fmt.Errorf("could not parse file reference from %q", string(input))
		return
	}
	file, line, err = windowsParseNoOffset(matches)
	return
}
