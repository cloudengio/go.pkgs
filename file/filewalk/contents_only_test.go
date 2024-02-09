// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/localfs"
	"cloudeng.io/sync/synctestutil"
)

var allFiles = `f0
f1
f2
la0
la1
lf0
a0/f0
a0/f1
a0/f2
a0/inaccessible-file
a0/a0.0/f0
a0/a0.0/f1
a0/a0.0/f2
a0/a0.1/f0
a0/a0.1/f1
a0/a0.1/f2
b0/b0.0/f0
b0/b0.0/f1
b0/b0.0/f2
b0/b0.1/b1.0/f0
b0/b0.1/b1.0/f1
b0/b0.1/b1.0/f2`

func TestSimple(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	sc := localfs.New()

	found := []filewalk.Entry{}
	err := filewalk.ContentsOnly(ctx, sc, localTestTree,
		func(ctx context.Context, prefix string, contents []filewalk.Entry, err error) error {
			if err != nil {
				if sc.IsPermissionError(err) {
					return nil
				}
				return err
			}
			for _, c := range contents {
				if c.IsDir() {
					t.Fatal(c)
				}
				nc := c
				nc.Name = sc.Join(prefix, c.Name)
				found = append(found, nc)
			}
			return nil
		})
	if err != nil {
		t.Errorf("%v", err)
	}
	fnames := []string{}
	for _, f := range found {
		fnames = append(fnames, f.Name)
	}
	sort.Strings(fnames)
	names := strings.Split(allFiles, "\n")
	for i, n := range names {
		names[i] = sc.Join(localTestTree, n)
	}
	sort.Strings(names)
	if got, want := len(fnames), len(names); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range names {
		if got, want := fnames[i], names[i]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
