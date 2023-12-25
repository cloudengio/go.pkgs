// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file"
)

func TestXAttr(t *testing.T) {
	tmpdir := t.TempDir()

	ctx := context.Background()
	fs := file.LocalFS()
	name := fs.Join(tmpdir, "testfile")
	if err := os.WriteFile(name, make([]byte, 4096), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := fs.Stat(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	xattr, err := fs.XAttr(ctx, name, info)
	if err != nil {
		t.Fatal(err)
	}
	if xattr.Device == 0 || xattr.FileID == 0 {
		t.Fatalf("got %v, want non-zero", xattr)
	}

	if got, want := xattr.Blocks, int64(8); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uid, gid := os.Getuid(), os.Getgid()
	if got, want := xattr.UID, int64(uid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.GID, int64(gid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.Hardlinks, uint64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if xattr.UID == -1 && len(xattr.User) == 0 {
		t.Errorf("got %v, want non-empty User", xattr)
	}

	if xattr.UID == -1 && len(xattr.Group) == 0 {
		t.Errorf("got %v, want non-empty Group", xattr)
	}

}

func TestSetXAttr(t *testing.T) {
	x := file.XAttr{
		UID:       1,
		GID:       2,
		User:      "user",
		Group:     "group",
		Device:    3,
		FileID:    4,
		Blocks:    5,
		Hardlinks: 6,
	}
	now := time.Now()
	fi := file.NewInfo("test", 8, 0, now, x)

	if got, want := fi.Sys(), x; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
