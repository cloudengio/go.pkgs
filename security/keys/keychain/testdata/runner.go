// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"

	"cloudeng.io/security/keys/keychain"
)

func main() {
	ctx := context.Background()
	if err := keychain.WithExternalPlugin(ctx); err != nil {
		panic(err)
	}
}
