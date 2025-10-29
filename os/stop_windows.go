// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package os

import (
	"os"
)

func isStopped(pid int) bool {
	_, err := os.FindProcess(pid)
	return err != nil
}
