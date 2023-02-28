// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file/content"
)

func TestObject(t *testing.T) {

	type testObject struct {
		A int
		B string
	}

	now := time.Now().Truncate(0)
	tobj := testObject{
		A: 1, B: "two",
	}
	buf, err := json.Marshal(tobj)
	if err != nil {
		t.Fatal(err)
	}
	raw := content.RawObject{
		Type:       content.Type("testObject"),
		CreateTime: now,
		Encoding:   content.JSONEncoding,
		Bytes:      buf,
		Error:      "oops",
	}
	obj := content.Object[testObject]{
		Object:    tobj,
		RawObject: raw,
	}
	roundtrip(t, obj)
}

func roundtrip[T any](t *testing.T, obj content.Object[T]) {
	// Test encode/decode
	enc, err := obj.Encode()
	if err != nil {
		t.Fatal(err)
	}
	var obj1 content.Object[T]
	if err := obj1.Decode(enc); err != nil {
		t.Fatal(err)
	}
	if got, want := obj, obj1; !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Test write/read.
	rt := &bytes.Buffer{}
	if err := obj.Write(rt); err != nil {
		t.Fatal(err)
	}
	var obj2 content.Object[T]
	if err := obj2.Read(rt); err != nil {
		t.Fatal(err)
	}
	if got, want := obj, obj2; !reflect.DeepEqual(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestByteEncoding(t *testing.T) {

	now := time.Now().Truncate(0)
	tobj := []byte("<html><body>hello</body></html>")

	raw := content.RawObject{
		Type:       content.Type("testObject"),
		CreateTime: now,
		Encoding:   content.ByteEncoding,
		Bytes:      tobj,
		Error:      "oops",
	}
	obj := content.Object[[]byte]{
		Object:    tobj,
		RawObject: raw,
	}
	roundtrip(t, obj)
}
