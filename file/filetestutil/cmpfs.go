// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"fmt"

	"cloudeng.io/errors"
	"cloudeng.io/file"
)

// CompareFS returns nil if the two instances of fs.FS contain exactly
// the same files and file contents.
func CompareFS(a, b file.FS) error {
	ca, cb := Contents(a), Contents(b)
	if got, want := len(ca), len(cb); got != want {
		return fmt.Errorf("got %v, want %v", got, want)
	}
	errs := errors.M{}
	for name, contents := range ca {
		if _, ok := cb[name]; !ok {
			return fmt.Errorf("%v was not downloaded", name)
		}
		if got, want := len(contents), len(cb[name]); got != want {
			return fmt.Errorf("%v: mismatched sizes: got %v, want %v", name, got, want)
		}
		if !bytes.Equal(contents, cb[name]) {
			n := 10
			if n > len(contents) {
				n = len(contents)
			}
			errs.Append(fmt.Errorf("mismatched contents (sha1) for %v (%v): %02x -- %02x", name, len(contents), contents[:n], cb[name]))
		}
	}
	return errs.Err()
}
