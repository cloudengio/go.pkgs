// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package executil

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
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

func signal(cmd *exec.Cmd, sig os.Signal) error {
	if cmd.Process == nil {
		return nil
	}
	if sig == os.Kill {
		return cmd.Process.Kill()
	}
	event := uint32(windows.CTRL_C_EVENT)
	if sig != os.Interrupt {
		event = uint32(windows.CTRL_BREAK_EVENT)
	}
	pid := uint32(cmd.Process.Pid)
	return windows.GenerateConsoleCtrlEvent(event, pid)
}
