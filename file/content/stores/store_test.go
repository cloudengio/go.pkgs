// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"cloudeng.io/file/content"
	"cloudeng.io/file/content/stores"
	"cloudeng.io/file/localfs"
)

func mkdirall(t *testing.T, paths ...string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(paths...), 0700)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStore(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	fs := localfs.New()

	root := fs.Join(tmpDir, "store")
	mkdirall(t, root)
	mkdirall(t, root, "other")
	mkdirall(t, root, "l1")
	mkdirall(t, root, "l1", "l2")
	if _, err := fs.Stat(ctx, root); err != nil {
		t.Fatal(err)
	}

	store := stores.New(fs)

	if err := store.EraseExisting(ctx, root); err != nil {
		t.Fatal(err)
	}

	_, err := fs.Stat(ctx, root)
	if err == nil || !fs.IsNotExist(err) {
		t.Errorf("expected not to exist: %v", err)
	}

	obj := content.Object[string, int]{
		Value:    "test",
		Response: 1,
		Type:     content.Type("test"),
	}

	prefix := fs.Join("a", "b")
	if err := obj.Store(ctx, store, fs.Join(root, prefix), "c", content.JSONObjectEncoding, content.JSONObjectEncoding); err != nil {
		t.Fatal(err)
	}
	path := fs.Join(root, prefix, "c")

	read, written := store.Stats()
	if got, want := read, int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := written, int64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := fs.Stat(ctx, path); err != nil {
		t.Fatal(err)
	}

	var obj1 content.Object[string, int]
	ctype, err := obj1.Load(ctx, store, fs.Join(root, prefix), "c")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ctype, obj.Type; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := obj1, obj; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	read, written = store.Stats()
	if got, want := read, int64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := written, int64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestAsync(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	fs := localfs.New()

	root := fs.Join(tmpDir, "store")
	mkdirall(t, root)
	mkdirall(t, root, "l1", "l2")
	if _, err := fs.Stat(ctx, root); err != nil {
		t.Fatal(err)
	}

	store := stores.NewAsync(ctx, fs, 5)

	if err := store.EraseExisting(ctx, root); err != nil {
		t.Fatal(err)
	}

	_, err := fs.Stat(ctx, root)
	if err == nil || !fs.IsNotExist(err) {
		t.Errorf("expected not to exist: %v", err)
	}

	prefix := fs.Join(root, "l1", "l2")
	for i := 0; i < 100; i++ {
		obj := content.Object[string, int]{
			Value:    fmt.Sprintf("test-0%3v", i),
			Response: i,
			Type:     content.Type("test"),
		}
		err := obj.Store(ctx, store, prefix, fmt.Sprintf("c-%03v", i), content.JSONObjectEncoding, content.JSONObjectEncoding)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Finish(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		var obj content.Object[string, int]
		ctype, err := obj.Load(ctx, store, prefix, fmt.Sprintf("c-%03v", i))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := ctype, content.Type("test"); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := obj.Value, fmt.Sprintf("test-0%3v", i); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := obj.Response, i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}
