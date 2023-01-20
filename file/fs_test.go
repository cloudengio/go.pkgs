// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package file_test

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"cloudeng.io/file"
)

func gobRoundTrip(t *testing.T, fi *file.Info) file.Info {
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
	return nfi
}

func jsonRoundTrip(t *testing.T, fi *file.Info) file.Info {
	buf, err := json.Marshal(fi)
	if err != nil {
		t.Fatal(err)
	}
	var nfi file.Info
	if err := json.Unmarshal(buf, &nfi); err != nil {
		t.Fatal(err)
	}
	return nfi
}

func TestEncodeDecode(t *testing.T) {

	sysinfo := struct{ name string }{"foo"}

	now := time.Now()
	fi := file.NewInfo("ab", 32, 0700, now, true, &sysinfo)

	type roundTripper func(*testing.T, *file.Info) file.Info

	for _, fn := range []roundTripper{
		jsonRoundTrip, gobRoundTrip,
	} {
		nfi := fn(t, fi)
		if got, want := nfi.Name(), "ab"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.Size(), int64(32); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.Mode(), fs.FileMode(0700); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.ModTime(), now; !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.IsDir(), true; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.Sys(), any(nil); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
