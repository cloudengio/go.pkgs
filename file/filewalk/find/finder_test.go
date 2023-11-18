// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package find_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/find"
	"cloudeng.io/file/filewalk/internal"
	"cloudeng.io/file/filewalk/localfs"
	"cloudeng.io/file/matcher"
	"cloudeng.io/sync/synctestutil"
)

func newMatcher(t *testing.T, items ...matcher.Item) matcher.T {
	t.Helper()
	m, err := matcher.New(items...)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNeedsStat(t *testing.T) {
	p := newMatcher(t, matcher.Regexp(".go"))
	f := newMatcher(t, matcher.Regexp(".go"))

	if got, want := find.NeedsStat(p, f), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	p = newMatcher(t, matcher.Regexp(".go"), matcher.OR(), matcher.FileType("f"))
	if got, want := find.NeedsStat(p, f), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	f = newMatcher(t, matcher.Regexp(".go"), matcher.OR(), matcher.NewerThanParsed("2010-12-13"))
	if got, want := find.NeedsStat(p, f), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func findFiles(ctx context.Context, t *testing.T, testTree, start string, pm, fm matcher.T, prune, needStat, followlinks bool) ([]find.Found, []find.Found) {
	fs := localfs.New()
	errs := &errors.M{}
	ch := make(chan find.Found, 1000)
	handler := find.New(fs, ch,
		find.WithPrefixMatcher(pm),
		find.WithFileMatcher(fm),
		find.WithPrune(prune),
		find.WithStat(needStat),
		find.WithFollowSoftlinks(followlinks))
	go func() {
		defer close(ch)
		wk := filewalk.New(fs, handler)
		if err := wk.Walk(ctx, start); err != nil {
			errs.Append(err)
			return
		}
	}()
	found := []find.Found{}
	foundErrors := []find.Found{}
	for f := range ch {
		f.Prefix = strings.TrimPrefix(f.Prefix, testTree)
		if f.Err != nil {
			foundErrors = append(foundErrors, f)
			continue
		}
		found = append(found, f)
	}
	if err := errs.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].Prefix == found[j].Prefix {
			return found[i].Name < found[j].Name
		}
		return found[i].Prefix < found[j].Prefix
	})
	return found, foundErrors
}

func cmpFound(t *testing.T, found []find.Found, expected []find.Found) {
	if got, want := len(found), len(expected); got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range found {
		if got, want := found[i].Prefix, expected[i].Prefix; got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
		if got, want := found[i].Name, expected[i].Name; got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func zipf(a []string, b ...string) []find.Found {
	z := make([]find.Found, 0, len(a))
	for i := range a {
		z = append(z, find.Found{Prefix: a[i], Name: b[i]})
	}
	return z
}
func zips(a ...string) []string {
	return a
}

func TestPrefixMatch(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()

	localTestTree := internal.CreateTestTree()
	start := time.Now()

	// prefix match.
	var pm, fm matcher.T
	pm = newMatcher(t, matcher.Regexp("a0$"), matcher.OR(), matcher.Regexp("b0.1$"))
	found, foundErrors := findFiles(ctx, t, localTestTree, localTestTree, pm, fm, false, find.NeedsStat(pm, fm), false)
	cmpFound(t, found, zipf(zips("/a0", "/b0/b0.1"), "", ""))
	cmpFound(t, foundErrors, zipf(zips("/inaccessible-dir"), ""))
	if got, want := os.IsPermission(foundErrors[0].Err), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// with needstats and follow softlinks should get an error for a broken softlink
	fm = newMatcher(t, matcher.Regexp(".*"))
	_, foundErrors = findFiles(ctx, t, localTestTree, localTestTree, pm, fm, false, true, true)
	cmpFound(t, foundErrors, zipf(zips("", "/inaccessible-dir"), "la1", ""))
	if got, want := os.IsNotExist(foundErrors[0].Err), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := os.IsPermission(foundErrors[1].Err), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// file and prefix match.
	fm = newMatcher(t, matcher.Regexp("f2"))
	found, _ = findFiles(ctx, t, localTestTree, localTestTree, pm, fm, false, find.NeedsStat(pm, fm), false)
	cmpFound(t, found, zipf(
		zips("", "/a0", "/a0", "/a0/a0.0", "/a0/a0.1", "/b0/b0.0", "/b0/b0.1", "/b0/b0.1/b1.0"),
		"f2", "", "f2", "f2", "f2", "f2", "", "f2", "f2", "f2"))

	// pruned prefix match.
	found, _ = findFiles(ctx, t, localTestTree, localTestTree, pm, fm, true, find.NeedsStat(pm, fm), false)
	t.Log(found)
	cmpFound(t, found, zipf(
		zips("", "/a0", "/b0/b0.0", "/b0/b0.1"),
		"f2", "", "f2", ""))

	// find soft links, without and with follow soft links set.
	for _, followSoftLinks := range []bool{false, true} {
		pm = matcher.T{}
		fm = newMatcher(t, matcher.FileType("l"))
		found, _ = findFiles(ctx, t, localTestTree, localTestTree, pm, fm, false, find.NeedsStat(pm, fm), followSoftLinks)
		cmpFound(t, found, zipf(zips("", "", ""), "la0", "la1", "lf0"))
	}

	// find files newer than a time.
	subTree := filepath.Join(localTestTree, "b0", "b0.1", "b1.0")
	pm = matcher.T{}
	// nothing is newer than hour into the future
	fm = newMatcher(t, matcher.NewerThanParsed(start.Add(time.Hour).Format(time.RFC3339)))
	found, _ = findFiles(ctx, t, localTestTree, subTree, pm, fm, false, find.NeedsStat(pm, fm), false)
	t.Log(found)
	cmpFound(t, found, nil)

	// everything is newer than an hour ago.
	fm = newMatcher(t, matcher.NewerThanParsed(start.Add(-time.Hour).Format(time.RFC3339)))
	found, _ = findFiles(ctx, t, localTestTree, subTree, pm, fm, false, find.NeedsStat(pm, fm), false)
	cmpFound(t, found, zipf(zips("/b0/b0.1/b1.0", "/b0/b0.1/b1.0", "/b0/b0.1/b1.0"), "f0", "f1", "f2"))

	// modify a file and use NewerThanTime for a reliable fine-grained file
	// modification time comparison.
	file := filepath.Join(subTree, "f1")
	if err := os.WriteFile(file, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	fm = newMatcher(t, matcher.NewerThanTime(start))
	found, _ = findFiles(ctx, t, localTestTree, subTree, pm, fm, false, find.NeedsStat(pm, fm), false)
	cmpFound(t, found, zipf(zips("/b0/b0.1/b1.0"), "f1"))

}
