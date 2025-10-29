// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix || darwin

package os

import (
	"syscall"
)

// IsStopped checks whether the process with the specified pid has stopped.
func isStopped(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))
	return err != nil
}
