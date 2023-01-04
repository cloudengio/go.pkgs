// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package goroutines

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	stackFileVolumeRE   = regexp.MustCompile(`^\s+([A-Za-z]+:[^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+)?)`)
	stackFileRE         = regexp.MustCompile(`^\s+([^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+))?$`)
	stackFileNoOffsetRE = regexp.MustCompile(`^\s+([^:]+):(\d+)$`)
)

func parseNoOffset(matches [][]byte) (file string, line int64, err error) {
	file = string(matches[1])
	line, err = strconv.ParseInt(string(matches[2]), 10, 64)
	return
}

func parseAll(matches [][]byte) (file string, line, offset int64, err error) {
	file = string(matches[1])
	line, err = strconv.ParseInt(string(matches[2]), 10, 64)
	if err != nil {
		return
	}
	offset, err = strconv.ParseInt(string(matches[3]), 16, 64)
	return
}

func parseFileLine(input []byte) (file string, line, offset int64, err error) {
	matches := stackFileVolumeRE.FindSubmatch(input)
	if len(matches) == 0 {
		matches = stackFileRE.FindSubmatch(input)
	}
	if len(matches) == 4 {
		file, line, offset, err = parseAll(matches)
		return
	}
	matches = stackFileNoOffsetRE.FindSubmatch(input)
	if len(matches) < 3 {
		err = fmt.Errorf("Could not parse file reference from %s", string(input))
		return
	}
	file, line, err = parseNoOffset(matches)
	return
}
