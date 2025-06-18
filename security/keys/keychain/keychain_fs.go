// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychain

import (
	"context"
)

// KeyChainReadFS defines an interface for reading files from a keychain via
// an internal or external plugin.
type KeyChainReadFS interface {
	ReadFileCtx(ctx context.Context, name string) ([]byte, error)
}

// KeyChainWriteFS defines an interface for writing files to a keychain via
// an internal or external plugin.
type KeyChainWriteFS interface {
	WriteFileCtx(ctx context.Context, name string, data []byte) error
}

// PluginFS combines both reading and writing capabilities for a keychain
// via an internal or external plugin.
type PluginFS struct {
	account string
}

func (p *PluginFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	return GetKey(ctx, p.account, name)
}

func (p *PluginFS) WriteFileCtx(ctx context.Context, name string, data []byte) error {
	return SetKey(ctx, p.account, name, data)
}

// NewPluginFS creates a new PluginFS instance with the specified binary and account.
// If the binary is specified, and the process is run via 'go run' then the
// external plugin will be used, otherwise the internal plugin is used.
func NewPluginFS(binary, account string) *PluginFS {
	return &PluginFS{
		account: account,
	}
}
