// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package userid

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func runIDCommand(uid string) (string, error) {
	args := []string{}
	if len(uid) > 0 {
		args = append(args, uid)
	}
	cmd := exec.Command("id", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%v: %v", strings.Join(cmd.Args, " "), err)
	}
	return string(out), err
}

// GetCurrentUser returns the current user as determined by environment
// variables.
func GetCurrentUser() string {
	return os.Getenv("USER")
}

func usernameOnly(s string) string {
	return s
}
