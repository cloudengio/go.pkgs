// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"context"
	"fmt"
	"os/exec"
)

// GoBuild builds the specified Go binary using the provided context and
// arguments. It returns the path to the built binary or an error if the build
// fails.
func GoBuild(ctx context.Context, binary string, args ...string) (string, error) {
	gobin, goargs, err := GoBuildArgs(binary, args...)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, gobin, goargs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", out, err)
	}
	return binary, nil
}

// GoBuildArgs returns the Go executable path, the arguments for a Go build
// command that outputs the specified binary, or an error if the Go executable
// could not be found.
// GoBuildArgs("mybinary", "a", "b") might return "/usr/local/go/bin/go", ["build", "-o", "mybinary", "a", "b"], nil.
func GoBuildArgs(binary string, args ...string) (string, []string, error) {
	gobin, err := exec.LookPath("go")
	if err != nil {
		return "", nil, fmt.Errorf("could not find 'go' executable: %v", err)
	}
	return gobin, append([]string{"build", "-o", binary}, args...), nil
}

// GoInstallArgs returns the Go executable path and the arguments for
// a Go install command, or an error if the Go executable could not be found.
// GoInstall("a", "b") might return "/usr/local/go/bin/go", ["install", "a", "b"], nil.
func GoInstallArgs(args ...string) (string, []string, error) {
	gobin, err := exec.LookPath("go")
	if err != nil {
		return "", nil, fmt.Errorf("could not find 'go' executable: %v", err)
	}
	return gobin, append([]string{"install"}, args...), nil
}
