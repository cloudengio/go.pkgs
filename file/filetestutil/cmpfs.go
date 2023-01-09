// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
)

// CompareFS returns nil if the two instances of fs.FS contain exactly
// the same files and file contents.
func CompareFS(a, b fs.FS) error {
	ca, cb := Contents(a), Contents(b)
	if got, want := len(ca), len(cb); got != want {
		return fmt.Errorf("got %v, want %v", got, want)
	}
	for name, contents := range ca {
		if _, ok := cb[name]; !ok {
			return fmt.Errorf("%v was not downloaded", name)
		}
		if got, want := len(contents), len(cb[name]); got != want {
			return fmt.Errorf("%v: mismatched sizes: got %v, want %v", name, got, want)
		}
		sumA := sha1.Sum(contents)
		sumB := sha1.Sum(cb[name])
		if got, want := hex.EncodeToString(sumA[:]), hex.EncodeToString(sumB[:]); got != want {
			return fmt.Errorf("%v: mismatched contents (sha1): got %v, want %v", name, got, want)
		}
	}
	return nil
}
