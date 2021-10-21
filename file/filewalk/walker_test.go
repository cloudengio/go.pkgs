// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file/filewalk"
)

type logger struct {
	prefix   string
	linesMu  sync.Mutex
	lines    []string
	children map[string][]filewalk.Info
	skip     string
}

func (l *logger) appendLine(s string) {
	l.linesMu.Lock()
	l.lines = append(l.lines, s)
	l.linesMu.Unlock()
}

func (l *logger) filesFunc(ctx context.Context, prefix string, info *filewalk.Info, ch <-chan filewalk.Contents) ([]filewalk.Info, error) {
	prefix = strings.TrimPrefix(prefix, l.prefix)
	children := make([]filewalk.Info, 0, 10)
	for results := range ch {
		results.Path = strings.TrimPrefix(results.Path, l.prefix)
		if err := results.Err; err != nil {
			l.appendLine(fmt.Sprintf("%v: %v\n", results.Path,
				strings.Replace(results.Err.Error(), l.prefix, "", 1)))
			continue
		}
		for _, info := range results.Files {
			full := filepath.Join(prefix, info.Name)
			link := ""
			if info.IsLink() {
				link = "@"
			}
			l.appendLine(fmt.Sprintf("%v%s: %v\n", full, link, info.Size))
		}
		children = append(children, results.Children...)
	}
	return children, nil
}

func (l *logger) dirsFunc(ctx context.Context, prefix string, info *filewalk.Info, err error) (bool, []filewalk.Info, error) {
	if err != nil {
		l.appendLine(fmt.Sprintf("dir  : error: %v: %v\n", prefix, err))
		return true, nil, nil
	}
	prefix = strings.TrimPrefix(prefix, l.prefix)
	if len(l.skip) > 0 && prefix == l.skip {
		return true, nil, nil
	}
	l.appendLine(fmt.Sprintf("%v*\n", prefix))
	return false, l.children[prefix], nil
}

func TestLocalWalk(t *testing.T) {
	ctx := context.Background()
	sc := filewalk.LocalFilesystem(1)
	wk := filewalk.New(sc)
	nl := func() *logger {
		return &logger{prefix: localTestTree,
			children: map[string][]filewalk.Info{},
		}
	}
	lg := nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.Concurrency(10))
	lg = nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.Concurrency(10))
	lg = nl()
	lg.skip = strings.ReplaceAll("/b0/b0.1", "/", string(filepath.Separator))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedPartial1)

	lg = nl()
	lg.skip = strings.ReplaceAll("/b0", "/", string(filepath.Separator))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedPartial2)

	lg = nl()
	b01, err := sc.Stat(ctx, sc.Join(localTestTree, "b0", "b0.1"))
	if err != nil {
		t.Fatal(err)
	}
	lg.children[strings.ReplaceAll("/b0", "/", string(filepath.Separator))] = []filewalk.Info{b01}
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedExistingChildren)
}

func testLocalWalk(ctx context.Context, t *testing.T, tmpDir string, wk *filewalk.Walker, lg *logger, expected string) {
	_, _, line, _ := runtime.Caller(1)
	err := wk.Walk(ctx, lg.dirsFunc, lg.filesFunc, tmpDir)
	if err != nil {
		t.Errorf("line: %v: errors: %v", line, err)
	}
	sort.Strings(lg.lines)
	if got, want := strings.Join(lg.lines, ""), expected; got != want {
		t.Errorf("line: %v: got %v, want %v", line, got, want)
	}
}

var expectedFull = `*
/a0*
/a0/a0.0*
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1*
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0*
/b0/b0.0*
/b0/b0.0/f0: 3
/b0/b0.0/f1: 3
/b0/b0.0/f2: 3
/b0/b0.1*
/b0/b0.1/b1.0*
/b0/b0.1/b1.0/f0: 3
/b0/b0.1/b1.0/f1: 3
/b0/b0.1/b1.0/f2: 3
/inaccessible-dir*
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

// No b0.1 sub dir.
var expectedPartial1 = `*
/a0*
/a0/a0.0*
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1*
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0*
/b0/b0.0*
/b0/b0.0/f0: 3
/b0/b0.0/f1: 3
/b0/b0.0/f2: 3
/inaccessible-dir*
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

// No b0 sub dir.
var expectedPartial2 = `*
/a0*
/a0/a0.0*
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1*
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/inaccessible-dir*
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

var expectedExistingChildren = `*
/a0*
/a0/a0.0*
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1*
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0*
/b0/b0.1*
/b0/b0.1/b1.0*
/b0/b0.1/b1.0/f0: 3
/b0/b0.1/b1.0/f1: 3
/b0/b0.1/b1.0/f2: 3
/inaccessible-dir*
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

func init() {
	if filepath.Separator != '/' {
		for _, p := range []*string{&expectedFull, &expectedPartial1, &expectedPartial2, &expectedExistingChildren} {
			*p = strings.ReplaceAll(*p, "/", string(filepath.Separator))
			*p = strings.ReplaceAll(*p, "permission denied", "Access is denied.")
		}
	}
}

func TestFunctionErrors(t *testing.T) {
	ctx := context.Background()
	sc := filewalk.LocalFilesystem(1)
	wk := filewalk.New(sc)

	err := wk.Walk(ctx,
		func(ctx context.Context, prefix string, info *filewalk.Info, err error) (skip bool, children []filewalk.Info, returnErr error) {
			return true, nil, fmt.Errorf("oops")
		},
		nil,
		localTestTree,
	)
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("missing or unexpected error: %v", err)
	}

	err = wk.Walk(ctx,
		func(ctx context.Context, prefix string, info *filewalk.Info, err error) (skip bool, children []filewalk.Info, returnErr error) {
			return false, nil, err
		},
		func(ctx context.Context, prefix string, info *filewalk.Info, ch <-chan filewalk.Contents) ([]filewalk.Info, error) {
			for c := range ch {
				_ = c
			}
			return nil, fmt.Errorf("oh no")
		},
		localTestTree,
	)
	if err == nil || strings.Count(err.Error(), "oh no") != 1 {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

type infiniteScanner struct{}

func (is *infiniteScanner) List(ctx context.Context, path string, ch chan<- filewalk.Contents) {
	time.Sleep(time.Millisecond * 1000)
	ch <- filewalk.Contents{
		Path: "infinite",
		Children: []filewalk.Info{
			{
				Name:    "child",
				ModTime: time.Now(),
				Mode:    filewalk.ModePrefix,
			},
		},
		Files: []filewalk.Info{
			{
				Name:    "file",
				ModTime: time.Now(),
			},
		},
	}
}

func (is *infiniteScanner) Stat(ctx context.Context, path string) (filewalk.Info, error) {
	info, err := os.Lstat(localTestTree)
	if err != nil {
		return filewalk.Info{}, err
	}
	return filewalk.Info{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    filewalk.FileMode(info.Mode()),
	}, nil
}

func (is *infiniteScanner) Join(components ...string) string {
	return filepath.Join(components...)
}

func (is *infiniteScanner) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (is *infiniteScanner) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func TestCancel(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	is := &infiniteScanner{}
	wk := filewalk.New(is, filewalk.Concurrency(4))
	lg := &logger{}

	ch := make(chan error)
	go func() {
		ch <- wk.Walk(ctx, lg.dirsFunc, lg.filesFunc, "anywhere")
	}()
	go cancel()
	select {
	case err := <-ch:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("missing or wrong error: %v", err)
		}
	case <-time.After(time.Second * 30):
		t.Fatalf("timed out")
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("missing or wrong error: %v", err)
		}
	default:
		t.Fatalf("context was not canceld")
	}
}
