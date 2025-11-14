// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

// Package keychain provides support for working with the macos keychain.
package keychain

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/keybase/go-keychain"
)

// WriteSecureNote writes a secure note to a local, non-icloud, keychain.
func WriteSecureNote(account, service string, data []byte) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	item.SetDescription("secure note")
	item.SetData(data)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		return os.ErrExist
	}
	return err
}

// ReadSecureNote reads a secure note from a local, non-icloud, keychain.
// The note may be in plist format if it was created directly in the keychain
// using keychain access.
func ReadSecureNote(account, service string) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "READING secure note for account %q, service %q\n", account, service)
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	query.SetReturnData(true)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnAttributes(true)
	results, err := keychain.QueryItem(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WTF.... %v\n", err)
		return nil, err
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "WTF.... no results\n")
		panic(fmt.Sprintf("WTF.... no results for account %q, service %q", account, service))
		return nil, fs.ErrNotExist
	}
	data, err := extractKeychainNote(results[0].Data)
	if err == io.EOF {
		// Maybe not an XML plist document.
		if len(results[0].Data) > 0 {
			return results[0].Data, nil
		}
		fmt.Fprintf(os.Stderr, "WTF.... EWOF... no results\n")
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "WTF.... error extracting keychain note: %v\n", err)
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "READING secure note for account %q, service %q: %d bytes\n", account, service, len(data))
	return data, err
}

type plist struct {
	Dict dict `xml:"dict"`
}

type dict struct {
	Entries []entry `xml:",any"`
}

type entry struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func extractKeychainNote(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewBuffer(data))
	var pl plist
	if err := dec.Decode(&pl); err != nil {
		return nil, err
	}
	for i, v := range pl.Dict.Entries {
		if v.XMLName.Local == "key" && v.Value == "NOTE" {
			if i+1 < len(pl.Dict.Entries) && pl.Dict.Entries[i+1].XMLName.Local == "string" {
				return []byte(pl.Dict.Entries[i+1].Value), nil
			}
		}
	}
	return nil, fs.ErrNotExist
}
