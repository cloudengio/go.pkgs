// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package s3fs implements fs.FS for AWS S3.
package file_test

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
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
	fi := file.NewInfo("ab", 32, 0700, now, file.InfoOption{
		User:    "user",
		Group:   "group",
		IsDir:   true,
		IsLink:  true,
		SysInfo: &sysinfo,
	})

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
		if got, want := nfi.IsLink(), true; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.Sys(), any(nil); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.User(), "user"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := nfi.Group(), "group"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func gobRoundTripList(t *testing.T, fi file.InfoList) file.InfoList {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(fi); err != nil {
		t.Fatal(err)
	}
	var nfi file.InfoList
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&nfi); err != nil {
		t.Fatal(err)
	}
	return nfi
}

func jsonRoundTripList(t *testing.T, fi file.InfoList) file.InfoList {
	buf, err := json.Marshal(fi)
	if err != nil {
		t.Fatal(err)
	}
	var nfi file.InfoList
	if err := json.Unmarshal(buf, &nfi); err != nil {
		t.Fatal(err)
	}
	return nfi
}

func TestEncodeDecodeList(t *testing.T) {

	sysinfo := struct{ name string }{"foo"}
	now := time.Now()
	var fl file.InfoList
	fl = fl.Append("0", 0, 0700, now, file.InfoOption{
		User:    "user0",
		Group:   "group0",
		IsDir:   true,
		IsLink:  true,
		SysInfo: &sysinfo,
	})
	fl = fl.Append("1", 1, 0700, now.Add(time.Minute), file.InfoOption{
		User:    "user1",
		Group:   "group1",
		IsDir:   true,
		IsLink:  true,
		SysInfo: &sysinfo,
	})

	if got, want := len(fl), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	type roundTripper func(*testing.T, file.InfoList) file.InfoList

	for _, fn := range []roundTripper{
		jsonRoundTripList, gobRoundTripList,
	} {
		nl := fn(t, fl)
		for i := 0; i <= 1; i++ {
			nfi := nl[i]
			id := fmt.Sprintf("%v", i)
			mt := now.Add(time.Minute * time.Duration(i))
			if got, want := nfi.Name(), id; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.Size(), int64(i); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.Mode(), fs.FileMode(0700); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.ModTime(), mt; !got.Equal(want) {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.IsDir(), true; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.IsLink(), true; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.Sys(), any(nil); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.User(), "user"+id; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := nfi.Group(), "group"+id; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}
