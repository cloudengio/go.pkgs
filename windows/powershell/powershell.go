// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package powershell

import (
	"bytes"
	"os/exec"
)

// T represents an instance of a Windows PowerShell
type T struct {
	ps string
}

// New creates a new PowerShell instance.
func New() *T {
	ps, _ := exec.LookPath("powershell.exe")
	return &T{ps}
}

// Run executes the supplied commands using PowerShell.
func (p *T) Run(args ...string) (stdOut string, stdErr string, err error) {
	args = append([]string{"-NoProfile", "-NonInteractive"}, args...)
	cmd := exec.Command(p.ps, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	stdOut, stdErr = stdout.String(), stderr.String()
	return
}
