// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package checkpoint_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"cloudeng.io/file/checkpoint"
)

func readdir(t *testing.T, d string) []string {
	f, err := os.Open(d)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	return names
}

func TestCheckpoint(t *testing.T) {
	ctx := context.Background()
	tmpdir := t.TempDir()
	tmp1 := filepath.Join(tmpdir, "1")
	op, err := checkpoint.NewDirectoryOperation(tmp1)
	assert := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	id, err := op.Checkpoint(ctx, "-1-of-3", []byte("0"))
	assert()
	if got, want := id, "00000000-1-of-3.chk"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	id, err = op.Checkpoint(ctx, "-2-of-3", []byte("1"))
	assert()
	if got, want := id, "00000001-2-of-3.chk"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	expected := []string{"00000000-1-of-3.chk", "00000001-2-of-3.chk", "lock"}
	if got, want := readdir(t, tmp1), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	latest, err := op.Latest(ctx)
	assert()
	if got, want := latest, []byte("1"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// With no label.
	tmp2 := filepath.Join(tmpdir, "2")

	op, err = checkpoint.NewDirectoryOperation(tmp2)

	id, err = op.Checkpoint(ctx, "", []byte("0"))
	assert()
	if got, want := id, "00000000.chk"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	id, err = op.Checkpoint(ctx, "", []byte("1"))
	assert()
	if got, want := id, "00000001.chk"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	id, err = op.Checkpoint(ctx, "", []byte("2"))
	assert()
	if got, want := id, "00000002.chk"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	expected = []string{"00000000.chk", "00000001.chk", "00000002.chk", "lock"}
	if got, want := readdir(t, tmp2), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	latest, err = op.Latest(ctx)
	assert()
	if got, want := latest, []byte("2"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}
