// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalktestutil_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk/filewalktestutil"
)

func newFS(t *testing.T, opts ...filewalktestutil.Option) *filewalktestutil.MockFS {
	t.Helper()
	fs, err := filewalktestutil.NewMockFS("root", opts...)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

const simpleSpec = `
name: root
uid: 12
entries:
  - file:
      name: f0
	  uid: 2
  - file:
      name: f1
  - dir:
	  name: d0
	  entries:
	    - file:
			name: f3`

func TestYAML(t *testing.T) {
	ctx := context.Background()

	mfs := newFS(t, filewalktestutil.WithYAMLConfig(simpleSpec))

	f, err := mfs.Stat(ctx, "root/f0")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f.Name(), "f0"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	f, err = mfs.Stat(ctx, "root/d0")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f.Name(), "d0"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.IsDir(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	_, err = mfs.Stat(ctx, "rox")
	if got, want := mfs.IsNotExist(err), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	_, err = mfs.Stat(ctx, "root/d0/f6")
	if got, want := mfs.IsNotExist(err), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := mfs.String(), `root
 f0
 f1
 d0
  f3
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

const scanSpec = `
name: root
entries:
  - file:
	  name: f0
  - file:
	  name: f1
  - dir:
	  name: d0
`

func TestScan(t *testing.T) {
	ctx := context.Background()
	mfs := newFS(t, filewalktestutil.WithYAMLConfig(scanSpec))

	for _, tc := range []struct {
		root string
		want []string
	}{
		{"root", []string{"f0", "f1", "d0"}},
		{"root/d0", []string{}},
	} {
		sc := mfs.LevelScanner(tc.root)
		found := []string{}
		for sc.Scan(ctx, 2) {
			for _, e := range sc.Contents() {
				found = append(found, e.Name)
			}
		}
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		if got, want := found, tc.want; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %v, want %v", tc.root, got, want)
		}
	}
}

const withDetailsSpec = `
name: root
size: 100
device: 30
file_id: 40
mode: 0700
uid: 10
gid: 1
time: "2021-10-10T03:03:03-07:00"
entries:
  - file:
	  name: f0
	  size: 2
	  device: 20
	  file_id: 30
	  mode: 0644
	  time: "2021-10-10T03:03:03-07:00"
	  uid: 20
	  gid: 2
`

func TestXAttr(t *testing.T) {
	ctx := context.Background()

	when, err := time.Parse(time.RFC3339, "2021-10-10T03:03:03-07:00")
	if err != nil {
		t.Fatal(err)
	}

	mfs := newFS(t, filewalktestutil.WithYAMLConfig(withDetailsSpec))

	f, err := mfs.Stat(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f.IsDir(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.Size(), int64(100); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.Mode().Perm(), os.FileMode(0700); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.ModTime(), when; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	xattr, err := mfs.XAttr(ctx, "root", f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := xattr, (file.XAttr{UID: 10, GID: 1, Device: 30, FileID: 40}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	f, err = mfs.Stat(ctx, "root/f0")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := f.Mode().IsRegular(), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.Size(), int64(2); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.Mode().Perm(), os.FileMode(0644); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := f.ModTime(), when; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	xattr, err = mfs.XAttr(ctx, "root/f0", f)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := xattr, (file.XAttr{UID: 20, GID: 2, Device: 20, FileID: 30}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
