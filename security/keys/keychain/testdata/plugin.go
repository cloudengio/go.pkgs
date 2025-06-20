// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"cloudeng.io/security/keys/keychain/plugins"
	"golang.org/x/term"
)

func main() {
	if term.IsTerminal(int(os.Stdin.Fd())) || term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Printf("stdin or stdout cannot be a terminal\n")
		os.Exit(1)
	}
	var req plugins.Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		os.Exit(1)
	}
	resp := plugins.Response{
		Account:  req.Account,
		Keyname:  req.Keyname,
		Contents: base64.StdEncoding.EncodeToString([]byte(req.Keyname)),
	}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		os.Stderr.WriteString("Failed to encode response: " + err.Error() + "\n")
	}
	os.Stdout.Sync()
}
