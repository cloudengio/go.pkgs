// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalktestutil_test

import (
	"context"
	"reflect"
	"testing"

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
