// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"flag"
	"fmt"

	"cloudeng.io/macos/keychainfs"
)

var (
	account string
)

func main() {
	flag.StringVar(&account, "account", keychainfs.DefaultAccount(), "keychain ccount that the note belongs to")

	flag.Parse()

	for _, arg := range flag.Args() {
		fs := keychainfs.New(keychainfs.WithAccount(account))
		data, err := fs.ReadFile(arg)
		if err != nil {
			fmt.Printf("error: secure note %v: %v\n", arg, err)
			return
		}
		fmt.Print(string(data))
	}
}
