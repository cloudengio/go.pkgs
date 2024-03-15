// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/localfs"
	"cloudeng.io/sync/synctestutil"
)

var (
	allFiles = `f0
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

	allB0Files = `b0/b0.0/f0
b0/b0.0/f1
b0/b0.0/f2
b0/b0.1/b1.0/f0
b0/b0.1/b1.0/f1
b0/b0.1/b1.0/f2`
)

func TestSimple(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	sc := localfs.New()

	found := []filewalk.Entry{}
	var mu sync.Mutex
	err := filewalk.ContentsOnly(ctx, sc, localTestTree,
		func(_ context.Context, prefix string, contents []filewalk.Entry, err error) error {
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
				c.Name = sc.Join(prefix, c.Name)
				mu.Lock()
				found = append(found, c)
				mu.Unlock()
			}
			return nil
		})
	if err != nil {
		t.Errorf("%v", err)
	}
	compareNames(t, sc, found, allFiles)
}

func compareNames(t *testing.T, sc filewalk.FS, found []filewalk.Entry, expected string) {
	fnames := []string{}
	for _, f := range found {
		fnames = append(fnames, f.Name)
	}
	sort.Strings(fnames)
	names := strings.Split(expected, "\n")
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

func TestSkipDir(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	sc := localfs.New()
	found := []filewalk.Entry{}
	var mu sync.Mutex

	err := filewalk.ContentsOnly(ctx, sc, localTestTree,
		func(_ context.Context, prefix string, contents []filewalk.Entry, err error) error {
			if err != nil {
				return nil
			}
			if prefix == sc.Join(localTestTree, "a0") {
				return filewalk.SkipDir
			}
			for _, c := range contents {
				c.Name = sc.Join(prefix, c.Name)
				mu.Lock()
				found = append(found, c)
				mu.Unlock()
			}
			return nil
		}, filewalk.WithScanSize(1))
	if err != nil {
		t.Errorf("%v", err)
	}

	for _, f := range found {
		fmt.Printf("found: %v\n", f.Name)
	}
	compareNames(t, sc, found, `f0
f1
f2
la0
la1
lf0
`+allB0Files)

}

func TestSkipAll(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	sc := localfs.New()
	found := []filewalk.Entry{}
	var mu sync.Mutex

	err := filewalk.ContentsOnly(ctx, sc, localTestTree,
		func(_ context.Context, prefix string, contents []filewalk.Entry, err error) error {
			if err != nil {
				return nil
			}
			for _, c := range contents {
				if c.Name == "la0" {
					return filewalk.SkipAll
				}
				c.Name = sc.Join(prefix, c.Name)
				mu.Lock()
				found = append(found, c)
				mu.Unlock()
			}
			return nil
		}, filewalk.WithScanSize(1))
	if err != nil {
		t.Errorf("%v", err)
	}
	anyOf := map[string]struct{}{}
	for _, n := range []string{"f0", "f1", "f2", "la0", "la1", "lf0"} {
		anyOf[sc.Join(localTestTree, n)] = struct{}{}
	}
	for _, f := range found {
		_, ok := anyOf[f.Name]
		if !ok {
			t.Errorf("unexpected: %v", f.Name)
		}
	}
	t.Fail()
}
