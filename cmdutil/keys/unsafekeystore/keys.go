// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package unsafekeystore is intended to document to use
// of plaintext, local filesystems being used to store keys.
package unsafekeystore

import (
	"cloudeng.io/file"
	"cloudeng.io/file/localfs"
)

// New returns a new instance of an unsafekeystore that reads keys from
// a plaintext, local file.
func New() file.ReadFileFS {
	return localfs.New()
}
