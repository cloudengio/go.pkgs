// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	fi := file.NewInfo(name, int64(len(contents)), 0600, time.Now().Truncate(0), nil)
	dl := []download.Result{{
		Contents: contents,
		Name:     name,
		FileInfo: &fi,
		Retries:  2,
		Err:      fmt.Errorf("oops"),
	}}
	objs := download.AsObjects(dl)
	roundtrip(t, objs[0], content.GOBObjectEncoding, content.GOBObjectEncoding)
	roundtrip(t, objs[0], content.JSONObjectEncoding, content.GOBObjectEncoding)
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

	now := time.Now().UTC().Truncate(0)
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
	roundtrip(t, obj, content.GOBObjectEncoding, content.GOBObjectEncoding)
	roundtrip(t, obj, content.JSONObjectEncoding, content.JSONObjectEncoding)
	roundtrip(t, obj, content.JSONObjectEncoding, content.GOBObjectEncoding)
}

func roundtrip[V, R any](t *testing.T, obj content.Object[V, R], valueEncoding, responseEncoding content.ObjectEncoding) {
	_, _, line, _ := runtime.Caller(1)
	loc := fmt.Sprintf("line: %v", line)
	// Test encode/decode
	data, err := obj.Encode(valueEncoding, responseEncoding)
	if err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	var obj1 content.Object[V, R]
	if err := obj1.Decode(data); err != nil {
		t.Fatalf("%s: %v", loc, err)
	}
	if got, want := obj, obj1; !reflect.DeepEqual(got, want) {
		t.Errorf("%v: got: %#v, want: %#v", loc, got, want)
	}
}

func roundTripFile[V, R any](ctx context.Context, t *testing.T, store *content.Store[V, R], obj content.Object[V, R], prefix, name string, ctype content.Type) {
	if err := store.Store(ctx, prefix, name, obj); err != nil {
		t.Fatal(err)
	}
	ctype1, obj1, err := store.Load(ctx, prefix, name)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctype1, ctype; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := obj1, obj; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestObjectEncoding(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	ctype := content.Type("bar")
	obj := content.Object[int, string]{
		Value:    3,
		Response: "anything",
		Type:     ctype,
	}
	fs := file.LocalFS()
	store := content.NewStore[int, string](fs, tmpDir, content.GOBObjectEncoding, content.GOBObjectEncoding)

	roundTripFile(ctx, t, store, obj, "a", "obj1.obj", ctype)
	roundTripFile(ctx, t, store, obj, "a", "obj2.obj", ctype)

}
