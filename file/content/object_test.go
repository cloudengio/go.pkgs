// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/download"
)

func TestCrawledObject(t *testing.T) {
	contents := []byte("hello world")
	name := "hello.html"
	fi := file.NewInfo(name, int64(len(contents)), 0600, time.Now().Truncate(0), file.InfoOption{})
	dl := []download.Result{{
		Contents: contents,
		Name:     name,
		FileInfo: fi,
		Retries:  2,
		Err:      fmt.Errorf("oops"),
	}}
	objs := download.AsObjects(dl)
	roundtrip(t, objs[0])
}

func TestAPIObject(t *testing.T) {
	type testValue struct {
		A int
		B string
	}

	type testResponse struct {
		Type       content.Type
		CreateTime time.Time
		Bytes      []byte
	}

	now := time.Now().Truncate(0)
	val := testValue{
		A: 1, B: "two",
	}
	buf, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}
	resp := testResponse{
		Type:       content.Type("testObject"),
		CreateTime: now,
		Bytes:      buf,
	}
	obj := content.Object[testValue, testResponse]{
		Value:    val,
		Response: resp,
	}
	roundtrip(t, obj)
}

func roundtrip[V, R any](t *testing.T, obj content.Object[V, R]) {
	_, _, line, _ := runtime.Caller(1)
	loc := fmt.Sprintf("line: %v", line)
	// Test encode/decode
	enc, err := obj.Encode()
	if err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	var obj1 content.Object[V, R]
	if err := obj1.Decode(enc); err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	if got, want := obj, obj1; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got: %v, want: %v", loc, got, want)
	}

	// Test write/read.
	rt := &bytes.Buffer{}
	if err := obj.Write(rt); err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	var obj2 content.Object[V, R]
	if err := obj2.Read(rt); err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	if got, want := obj, obj2; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got: %v, want: %v", loc, got, want)
	}
}

func TestBinaryEncoding(t *testing.T) {
	tmpDir := t.TempDir()
	ctype := content.Type("bar")
	obj := content.Object[int, string]{
		Value:    3,
		Response: "anything",
		Type:     ctype,
	}
	path := filepath.Join(tmpDir, "obj")

	data, _ := obj.Encode()

	if err := content.WriteBinary(path, ctype, data); err != nil {
		t.Fatal(err)
	}
	ctype1, data1, err := content.ReadBinary(path)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := ctype1, ctype; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := data1, data; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	var obj1 content.Object[int, string]
	if err := obj1.Decode(data1); err != nil {
		t.Fatal(err)
	}

	if got, want := obj, obj1; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
