// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs_test

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/s3fs"
)

func readdir(ctx context.Context, t *testing.T, fs *s3fs.T, prefix string) []string {
	t.Helper()
	sc := fs.LevelScanner(prefix)
	names := []string{}
	for sc.Scan(ctx, 100) {
		contents := sc.Contents()
		for _, c := range contents {
			if c.IsDir() {
				continue
			}
			names = append(names, c.Name)
			fmt.Printf("ADD %v\n", c.Name)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	return names
}

func TestCheckpoint(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()

	tmpdir := "s3://bucket-checkpoint/a"
	cpdir1 := fs.Join(tmpdir, "checkpoint")
	op := s3fs.NewCheckpointOperation(fs)
	var err error
	assert := func() {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	err = op.Init(ctx, cpdir1)
	assert()

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
	expected := []string{"00000000-1-of-3.chk", "00000001-2-of-3.chk"}
	if got, want := readdir(ctx, t, fs, cpdir1), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	latest, err := op.Latest(ctx)
	assert()
	if got, want := latest, []byte("1"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	err = op.Clear(ctx)
	assert()

	// With no label.
	op = s3fs.NewCheckpointOperation(fs)
	cpdir2 := fs.Join(tmpdir, "2")
	err = op.Init(ctx, cpdir2)
	assert()

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
	expected = []string{"00000000.chk", "00000001.chk", "00000002.chk"}
	if got, want := readdir(ctx, t, fs, cpdir2), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	latest, err = op.Latest(ctx)
	assert()
	if got, want := latest, []byte("2"); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	err = op.Clear(ctx)
	assert()

}

func TestCompact(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()
	tmpdir := "s3://bucket-checkpoint/b"
	cpdir := fs.Join(tmpdir, "checkpoint")
	op := s3fs.NewCheckpointOperation(fs)
	err := op.Init(ctx, cpdir)
	assert := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	expected := []string{}
	for i := 0; i < 5; i++ {
		_, err = op.Checkpoint(ctx, "", []byte(fmt.Sprintf("%02v", i)))
		assert()
		expected = append(expected, fmt.Sprintf("%08v.chk", i))
	}
	if got, want := readdir(ctx, t, fs, cpdir), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	err = op.Compact(ctx, "-label")
	assert()
	expected = append([]string{}, "00000000-label.chk")
	if got, want := readdir(ctx, t, fs, cpdir), expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	latest, err := op.Latest(ctx)
	assert()
	if got, want := latest, []byte(fmt.Sprintf("%02v", 4)); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestInvalidFilenames(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()
	tmpdir := "s3://bucket-checkpoint/c"
	cpdir := fs.Join(tmpdir, "checkpoint")
	op := s3fs.NewCheckpointOperation(fs)

	err := op.Init(ctx, cpdir)
	assert := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	expected := []string{}
	for i := 0; i < 5; i++ {
		_, err = op.Checkpoint(ctx, "-something-", []byte(fmt.Sprintf("%02v", i)))
		assert()
		expected = append(expected, fmt.Sprintf("%08v-something-.chk", i))
	}

	err = fs.Put(ctx, fs.Join(cpdir, "invalid.chk"), 0644, []byte("invalid"))
	assert()

	withInvalid := append([]string{}, expected...)
	withInvalid = append(withInvalid, "invalid.chk")

	if got, want := readdir(ctx, t, fs, cpdir), withInvalid; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	data, err := op.Latest(ctx)
	assert()
	if got, want := data, []byte(fmt.Sprintf("%02v", 4)); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	err = op.Compact(ctx, "")
	assert()

	data, err = op.Latest(ctx)
	assert()
	if got, want := data, []byte(fmt.Sprintf("%02v", 4)); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	err = op.Complete(ctx)
	assert()

}
