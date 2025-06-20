// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package plugins

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"cloudeng.io/macos/keychainfs"
)

// Plugin is the entry point for the macOS keychain plugin. It reads a
// Request from in and writes a Response to out.
func Plugin(in io.Reader, out io.Writer) error {
	var req Request
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode input: %w", err)
	}
	account := req.Account
	if account == "" {
		account = keychainfs.DefaultAccount()
	}
	key := req.Keyname
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	fs := keychainfs.NewSecureNoteFS(keychainfs.WithAccount(account))
	if req.WriteKey {
		if err := writeSecureNote(fs, account, key, req.Contents, out); err != nil {
			return fmt.Errorf("keychain plugin write: account %q, key %q: %w", account, key, err)
		}
		return nil
	}
	err := readSecureNote(fs, account, key, out)
	if err != nil {
		return fmt.Errorf("keychain plugin read: account %q, key %q: %w", account, key, err)
	}
	return nil
}

func readSecureNote(fs *keychainfs.SecureNoteFS, account, key string, out io.Writer) error {
	data, err := fs.ReadFile(key)
	if err != nil {
		return fmt.Errorf("failed to read secure note account: %w", err)
	}
	resp := Response{
		Account:  account,
		Keyname:  key,
		Contents: base64.StdEncoding.EncodeToString(data),
	}
	if err := json.NewEncoder(out).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

func writeSecureNote(fs *keychainfs.SecureNoteFS, account, key, contents string, out io.Writer) error {
	if len(contents) == 0 {
		return fmt.Errorf("contents cannot be empty when writing a secure note")
	}
	data, err := base64.StdEncoding.DecodeString(contents)
	if err != nil {
		return fmt.Errorf("failed to decode contents: %w", err)
	}
	if err := fs.WriteFile(key, data, 0); err != nil {
		return fmt.Errorf("failed to write secure note: %w", err)
	}
	resp := Response{
		Account: account,
		Keyname: key,
	}
	if err := json.NewEncoder(out).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}
