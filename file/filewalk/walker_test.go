// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/sync/synctestutil"
)

type logger struct {
	prefix   string
	linesMu  sync.Mutex
	lines    []string
	children map[string][]file.Info
	skip     string
}

func (l *logger) appendLine(s string) {
	l.linesMu.Lock()
	l.lines = append(l.lines, s)
	l.linesMu.Unlock()
}

func (l *logger) filesFunc(_ context.Context, prefix string, _ *file.Info, ch <-chan filewalk.Contents) ([]file.Info, error) {
	prefix = strings.TrimPrefix(prefix, l.prefix)
	children := make([]file.Info, 0, 10)
	for results := range ch {
		results.Path = strings.TrimPrefix(results.Path, l.prefix)
		if err := results.Err; err != nil {
			l.appendLine(fmt.Sprintf("%v: %v\n", results.Path,
				strings.Replace(results.Err.Error(), l.prefix, "", 1)))
			continue
		}
		for _, info := range results.Files {
			full := filepath.Join(prefix, info.Name())
			link := ""
			if info.IsLink() {
				link = "@"
			}
			l.appendLine(fmt.Sprintf("%v%s: %v\n", full, link, info.Size()))
		}
		children = append(children, results.Children...)
	}
	return children, nil
}

func (l *logger) dirsFunc(_ context.Context, prefix string, _ *file.Info, err error) (bool, []file.Info, error) {
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
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	sc := filewalk.LocalFilesystem(1)
	wk := filewalk.New(sc)
	nl := func() *logger {
		return &logger{prefix: localTestTree,
			children: map[string][]file.Info{},
		}
	}
	lg := nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.WithConcurrency(10))
	lg = nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.WithConcurrency(10))
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
	lg.children[strings.ReplaceAll("/b0", "/", string(filepath.Separator))] = []file.Info{b01}
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
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	sc := filewalk.LocalFilesystem(1)
	wk := filewalk.New(sc)

	err := wk.Walk(ctx,
		func(ctx context.Context, prefix string, info *file.Info, err error) (skip bool, children []file.Info, returnErr error) {
			return true, nil, fmt.Errorf("oops")
		},
		nil,
		localTestTree,
	)
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("missing or unexpected error: %v", err)
	}

	err = wk.Walk(ctx,
		func(ctx context.Context, prefix string, info *file.Info, err error) (skip bool, children []file.Info, returnErr error) {
			return false, nil, err
		},
		func(ctx context.Context, prefix string, info *file.Info, ch <-chan filewalk.Contents) ([]file.Info, error) {
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

func (is *infiniteScanner) List(_ context.Context, _ string, ch chan<- filewalk.Contents) {
	time.Sleep(time.Millisecond * 1000)
	ch <- filewalk.Contents{
		Path: "infinite",
		Children: []file.Info{
			*file.NewInfo("child", 0, fs.ModeDir, time.Now(), file.InfoOption{}),
		},
		Files: []file.Info{
			*file.NewInfo("file", 0, 0, time.Now(), file.InfoOption{}),
		},
	}
}

func (is *infiniteScanner) Stat(_ context.Context, _ string) (file.Info, error) {
	info, err := os.Lstat(localTestTree)
	if err != nil {
		return file.Info{}, err
	}
	return *file.NewInfo(
		info.Name(),
		info.Size(),
		info.Mode(),
		info.ModTime(),
		file.InfoOption{},
	), nil
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
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	is := &infiniteScanner{}
	wk := filewalk.New(is, filewalk.WithConcurrency(4))
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

type slowScanner struct {
	filewalk.Filesystem
}

func (s *slowScanner) List(ctx context.Context, prefix string, ch chan<- filewalk.Contents) {
	time.Sleep(time.Millisecond * 1500)
	s.Filesystem.List(ctx, prefix, ch)
}

func TestReporting(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	//	ctx, cancel := context.WithCancel(ctx)
	//	defer cancel()
	is := &slowScanner{filewalk.LocalFilesystem(1)}
	ch := make(chan filewalk.Status, 100)
	wk := filewalk.New(is, filewalk.WithConcurrency(2),
		filewalk.WithReporting(ch, time.Millisecond*100, time.Millisecond*250))
	lg := &logger{}
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		wk.Walk(ctx, lg.dirsFunc, lg.filesFunc, localTestTree)
		wg.Done()
	}()

	nSlow := 0
	nSync := int64(0)
	for r := range ch {
		if r.SlowPrefix != "" {
			nSlow++
			if r.ScanDuration < time.Millisecond*250 {
				t.Errorf("scan duration too short: %v", r.ScanDuration)
			}
		}
		nSync += r.SynchronousScans
	}

	if nSlow <= 40 {
		t.Errorf("not enough slow scans: %v", nSlow)
	}
	if nSync == 0 {
		t.Errorf("no synchronous scans: %v", nSync)
	}
}
