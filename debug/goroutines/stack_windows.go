// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package goroutines

import "regexp"

var stackFileRE = regexp.MustCompile(`^\s+[A-Za-z]+:([^:]+):(\d+)(?: \+0x([0-9A-Fa-f]+))?$`)
