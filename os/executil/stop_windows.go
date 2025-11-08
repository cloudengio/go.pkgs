// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package executil

import (
	"os"
	"os/exec"
	"syscall"
)

func isStopped(pid int) bool {
	// PROCESS_QUERY_INFORMATION is enough to check for existence.
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return true
	}
	syscall.CloseHandle(h)
	return false
}

func signal(*exec.Cmd, os.Signal) error {
	return nil
}
