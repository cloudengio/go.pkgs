// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloudeng.io/macos/keychainfs"
	"github.com/keybase/go-keychain"
)

var (
	account string
	service string
)

func usage() {
	name := filepath.Base(os.Args[0])
	fmt.Printf("%s <flags> [filename|-]\n", name)
	flag.PrintDefaults()
}

func usageAndExit() {
	usage()
	os.Exit(1)
}

func main() {
	flag.StringVar(&account, "account", keychainfs.DefaultAccount(), "keychain account that the note belongs to")
	flag.StringVar(&service, "service", "", "keychain service that the note belongs to")
	flag.Parse()

	args := flag.Args()

	if len(service) == 0 {
		fmt.Printf("-service must be specified\n")
		usageAndExit()

	}

	file := os.Stdin
	switch len(args) {
	case 0:
	case 1:
		if args[0] != "-" {
			var err error
			file, err = os.Open(args[0])
			if err != nil {
				fmt.Printf("error: %v\n", err)
				return
			}
		}
	default:
		usageAndExit()
	}
	data, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	item.SetDescription("secure note")
	item.SetData(data)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	err = keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		fmt.Printf("an item for service %q and account %q already exists\n", service, account)
		os.Exit(1)
	}
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("stored note for service: %q, account %q\n", service, account)
}
