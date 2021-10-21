// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package userid

import (
	"fmt"
	"os"
	"strings"

	"cloudeng.io/os/windows/powershell"
)

func runIDCommand(uid string) (string, error) {
	var script string
	if len(uid) > 0 {
		script = fmt.Sprintf(`(glu | where sid -eq %s).Name`, uid)
	} else {
		script = `$env:username
(glu | where name -eq $env:username).sid.value`
	}
	ps := powershell.New()
	stdout, _, err := ps.Run(script)
	if err != nil {
		return "", err
	}
	if len(uid) > 0 {
		return fmt.Sprintf("uid=%s(%s)", uid, strings.TrimSpace(stdout)), nil
	}
	parts := strings.Split(stdout, "\n")
	if len(parts) == 2 {
		return fmt.Sprintf("uid=%s(%s)", parts[1], parts[0]), nil
	}
	return "", fmt.Errorf("failed to parse power shell output: %v", stdout)
}

// GetCurrentUser returns the current user as determined by environment
// variables. On windows, it does not include the domain name.
func GetCurrentUser() string {
	return os.Getenv("USERNAME")
}

// return just the username without the domain name component if it's present.
func usernameOnly(s string) string {
	_, u := ParseWindowsUser(s)
	return u
}
