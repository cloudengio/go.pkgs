// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package file_test

import (
	"bytes"
	"encoding/gob"
	"io/fs"
	"testing"
	"time"

	"cloudeng.io/file"
)

func TestGob(t *testing.T) {

	sysinfo := struct{ name string }{"foo"}

	now := time.Now().Round(0) // strip the monotonic clock value.
	fi := file.NewInfo("ab", 32, 0700, now, true, &sysinfo)

	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(fi); err != nil {
		t.Fatal(err)
	}

	var nfi file.Info
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&nfi); err != nil {
		t.Fatal(err)
	}

	if got, want := nfi.Name(), "ab"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nfi.Size(), int64(32); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nfi.Mode(), fs.FileMode(0700); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nfi.ModTime(), now; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nfi.IsDir(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nfi.Sys(), any(nil); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
