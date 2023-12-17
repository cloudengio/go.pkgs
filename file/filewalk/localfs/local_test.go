// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs_test

import (
	"context"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/internal"
	"cloudeng.io/file/filewalk/localfs"
)

var localTestTree string

func TestMain(m *testing.M) {
	localTestTree = internal.CreateTestTree()
	code := m.Run()
	os.RemoveAll(localTestTree)
	os.Exit(code)
}

func scan(sc filewalk.FS, dir string) (dirNames, fileNames []string, errors []error, info map[string]file.Info) {
	ctx := context.Background()
	info = map[string]file.Info{}
	ds := sc.LevelScanner(dir)
	for ds.Scan(ctx, 1) {
		entries := ds.Contents()
		for _, entry := range entries {
			fi, err := sc.Lstat(ctx, sc.Join(dir, entry.Name))
			if err != nil {
				errors = append(errors, err)
				continue
			}
			info[entry.Name] = fi
			if entry.IsDir() {
				dirNames = append(dirNames, entry.Name)
			} else {
				fileNames = append(fileNames, entry.Name)
			}
		}
	}
	if err := ds.Err(); err != nil {
		errors = append(errors, err)
	}
	sort.Strings(dirNames)
	sort.Strings(fileNames)
	return
}

func TestLocalFilesystem(t *testing.T) {
	sc := localfs.New()

	dirs, files, errors, info := scan(sc, localTestTree)

	expectedDirNames := []string{"a0", "b0", "inaccessible-dir"}
	expectedFileNames := []string{"f0", "f1", "f2", "la0", "la1", "lf0"}

	if got, want := dirs, expectedDirNames; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := files, expectedFileNames; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(errors), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, d := range expectedDirNames {
		i := info[d]
		if got, want := i.IsDir(), true; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
		if got, want := i.Mode()&fs.ModeSymlink == fs.ModeSymlink, false; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
	}

	for _, f := range expectedFileNames {
		i := info[f]
		if got, want := i.IsDir(), false; got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
		if !strings.HasPrefix(f, "l") {
			if got, want := i.Size(), int64(3); got != want {
				t.Errorf("%v: got %v, want %v", f, got, want)
			}
		}
		if got, want := i.Mode()&fs.ModeSymlink == fs.ModeSymlink, strings.HasPrefix(f, "l"); got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
	}

	_, _, errors, _ = scan(sc, sc.Join(localTestTree, "inaccessible-dir"))

	if got, want := len(errors), 1; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got, want := os.IsPermission(errors[0]), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestXAttr(t *testing.T) {
	tmpdir := t.TempDir()

	ctx := context.Background()
	fs := localfs.New()
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
	if uid == -1 {
		// on windows uid, gid are zero for now.
		uid, gid = 0, 0
	}
	if got, want := xattr.UID, uint64(uid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.GID, uint64(gid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.Hardlinks, uint64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestSetXAttr(t *testing.T) {
	x := filewalk.XAttr{
		UID:       1,
		GID:       2,
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
