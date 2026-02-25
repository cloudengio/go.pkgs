// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filetestutil

import (
	"bytes"
	"fmt"
	"reflect"

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
			n := min(10, len(contents))
			errs.Append(fmt.Errorf("mismatched contents (sha1) for %v (%v): %02x -- %02x", name, len(contents), contents[:n], cb[name]))
		}
	}
	return errs.Err()
}

func CompareFileInfo(a, b file.InfoList) error {
	if got, want := len(a), len(b); got != want {
		return fmt.Errorf("len: got %v, want %v", got, want)
	}
	for i := range a {
		if got, want := a[i].Name(), b[i].Name(); got != want {
			return fmt.Errorf("name: got %v, want %v", got, want)
		}
		if got, want := a[i].Size(), b[i].Size(); got != want {
			return fmt.Errorf("size: got %v, want %v", got, want)
		}
		if got, want := a[i].Mode(), b[i].Mode(); got != want {
			return fmt.Errorf("mode: got %v, want %v", got, want)
		}
		if got, want := a[i].ModTime(), b[i].ModTime(); !got.Equal(want) {
			return fmt.Errorf("modTime: got %v, want %v", got, want)
		}
		if got, want := a[i].IsDir(), b[i].IsDir(); got != want {
			return fmt.Errorf("isDir: got %v, want %v", got, want)
		}
		if got, want := reflect.TypeOf(a[i].Sys()), reflect.TypeOf(b[i].Sys()); got != want {
			return fmt.Errorf("sys: got %v, want %v", got, want)
		}
	}
	return nil
}
