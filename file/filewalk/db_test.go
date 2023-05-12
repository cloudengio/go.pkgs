// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

func TestCodec(t *testing.T) {
	now := time.Now().Round(0)
	pi := filewalk.PrefixInfo{
		ModTime:   now,
		Size:      33,
		UserID:    "500",
		Mode:      0555,
		DiskUsage: 999,
		Err:       "some err",
	}
	child := *file.NewInfo(
		"file1",
		3444,
		0666,
		now,
		file.InfoOption{User: "600"},
	)
	pi.Files = []file.Info{child, child}
	pi.Children = []file.Info{child, child}
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(pi); err != nil {
		t.Fatal(err)
	}

	dec := gob.NewDecoder(bytes.NewBuffer(buf.Bytes()))
	var npi filewalk.PrefixInfo
	if err := dec.Decode(&npi); err != nil {
		t.Fatal(err)
	}
	if got, want := pi, npi; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
