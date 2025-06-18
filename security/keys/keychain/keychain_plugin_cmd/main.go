// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"os"

	"cloudeng.io/security/keys/keychain/plugins"
	"golang.org/x/term"
)

func main() {
	if term.IsTerminal(int(os.Stdin.Fd())) || term.IsTerminal(int(os.Stdout.Fd())) {
		msg := plugins.Response{
			Error: "stdin or stdout cannot be a terminal",
		}
		if err := json.NewEncoder(os.Stdout).Encode(msg); err != nil {
			os.Stderr.WriteString("plugin failed: " + err.Error() + "\n")
		}
		if err := json.NewEncoder(os.Stderr).Encode(msg); err != nil {
			os.Stderr.WriteString("plugin failed: " + err.Error() + "\n")
		}
		return
	}
	defer os.Stdout.Sync()
	err := plugins.Plugin(os.Stdin, os.Stdout)
	if err == nil {
		return
	}
	msg := plugins.Response{
		Error: err.Error(),
	}
	if err := json.NewEncoder(os.Stdout).Encode(msg); err != nil {
		os.Stderr.WriteString("plugin failed: " + err.Error() + "\n")
	}
}
