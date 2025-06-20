// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"slices"

	"cloudeng.io/macos/keychainfs"
	"cloudeng.io/security/keys/keychain"
)

func createKeychainEntry(ctx context.Context, account, key string, data []byte) {
	fs := keychainfs.NewSecureNoteFS(
		keychainfs.WithAccount(account))
	if err := fs.WriteFileCtx(ctx, key, data, 0600); err != nil {
		if !errors.Is(err, os.ErrExist) {
			panic(err)
		}
	}

	rd, err := fs.ReadFileCtx(ctx, key)
	if err != nil {
		panic(err)
	}
	if got, want := rd, data; !slices.Equal(got, want) {
		panic(fmt.Errorf("expected data %q, got %q", want, got))
	}
}

func main() {
	ctx := context.Background()
	if err := keychain.WithExternalPlugin(ctx); err != nil {
		panic(fmt.Errorf("failed to initialize keychain plugin: %w", err))
	}

	data := base64.StdEncoding.EncodeToString([]byte("test_data"))

	createKeychainEntry(ctx, "", "test-key", []byte(data))
	d, err := keychain.GetKey(ctx, "", "test-key")
	if err != nil {
		panic(err)
	}
	if got, want := d, data; !slices.Equal(got, []byte(want)) {
		panic(fmt.Errorf("expected data %q, got %q", want, got))
	}

}
