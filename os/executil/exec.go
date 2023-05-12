// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"context"
	"os/exec"
)

// Command is the same as exec.Command, but ensures that the path
// to the executable is in the form expected by the operating system.
func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Path = ExecName(cmd.Path)
	return cmd
}

// CommandContext is the same as exec.CommandContext, but ensures that the path
// to the executable is in the form expected by the operating system.
func CommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Path = ExecName(cmd.Path)
	return cmd
}
