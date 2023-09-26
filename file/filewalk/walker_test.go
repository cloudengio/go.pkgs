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
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/internal"
	"cloudeng.io/file/filewalk/localfs"
	"cloudeng.io/sync/synctestutil"
)

var localTestTree string

func TestMain(m *testing.M) {
	localTestTree = internal.CreateTestTree()
	code := m.Run()
	os.RemoveAll(localTestTree)
	os.Exit(code)
}

type logger struct {
	prefix   string
	fs       filewalk.FS
	linesMu  sync.Mutex
	lines    []string
	children map[string]file.InfoList
	skip     string
}

func (l *logger) appendLine(s string) {
	l.linesMu.Lock()
	defer l.linesMu.Unlock()
	l.lines = append(l.lines, s)
}

/*
func (l *logger) handleContents(ctx context.Context, prefix, parent string, unchanged bool, contents filewalk.Contents) (file.InfoList, error) {
	children := file.InfoList{}
	if err := contents.Err; err != nil {
		l.appendLine(fmt.Sprintf("%v: %v\n", parent,
			strings.Replace(contents.Err.Error(), l.prefix, "", 1)))
		return nil, nil
	}
	for _, de := range contents.Entries {
		link := ""
		info, err := l.fs.LStat(ctx, l.fs.Join(prefix, de.Name))
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			link = "@"
		}
		if de.IsDir() {
			children = children.AppendInfo(info)
			continue
		}
		l.appendLine(fmt.Sprintf("%v%s: %v\n", l.fs.Join(parent, de.Name), link, info.Size()))
	}
	return children, nil
}
*/

func (l *logger) entriesFunc(ctx context.Context, prefix string, unchanged bool, _ file.Info, contents filewalk.Contents, ch <-chan filewalk.Contents) (file.InfoList, error) {
	children := file.InfoList{}
	parent := strings.TrimPrefix(prefix, l.prefix)
	if err := contents.Err; err != nil {
		l.appendLine(fmt.Sprintf("%v: %v\n", parent,
			strings.Replace(contents.Err.Error(), l.prefix, "", 1)))
		return nil, nil
	}
	for _, de := range contents.Entries {
		link := ""
		info, err := l.fs.LStat(ctx, l.fs.Join(prefix, de.Name))
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			link = "@"
		}
		if de.IsDir() {
			children = children.AppendInfo(info)
			continue
		}
		l.appendLine(fmt.Sprintf("%v%s: %v\n", l.fs.Join(parent, de.Name), link, info.Size()))
	}
	return children, nil
}

/*
	var children file.InfoList
	parent := strings.TrimPrefix(prefix, l.prefix)
	if ch == nil {
		return l.handleContents(ctx, prefix, parent, unchanged, contents)
	}
	for contents := range ch {
		c, err := l.handleContents(ctx, prefix, parent, unchanged, contents)
		if err != nil {
			return nil, err
		}
		children = append(children, c...)
	}
	return children, nil
}*/

func (l *logger) dirsFunc(_ context.Context, prefix string, _ file.Info, err error) (bool, bool, file.InfoList, error) {
	if err != nil {
		l.appendLine(fmt.Sprintf("dir  : error: %v: %v\n", prefix, err))
		return true, false, nil, nil
	}
	prefix = strings.TrimPrefix(prefix, l.prefix)
	if len(l.skip) > 0 && prefix == l.skip {
		return true, false, nil, nil
	}
	l.appendLine(fmt.Sprintf("%v*\n", prefix))
	return false, false, l.children[prefix], nil
}

func TestLocalWalk(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	sc := localfs.New()
	wk := filewalk.New(sc, filewalk.WithScanSize(1))
	nl := func() *logger {
		return &logger{prefix: localTestTree,
			fs:       sc,
			children: map[string]file.InfoList{},
		}
	}
	lg := nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	return

	wk = filewalk.New(sc, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
	lg = nl()
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
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
	lg.children[strings.ReplaceAll("/b0", "/", string(filepath.Separator))] = file.InfoList{b01}
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedExistingChildren)
}

func testLocalWalk(ctx context.Context, t *testing.T, tmpDir string, wk *filewalk.Walker, lg *logger, expected string) {
	_, _, line, _ := runtime.Caller(1)
	err := wk.Walk(ctx, lg.dirsFunc, lg.entriesFunc, tmpDir)
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
	sc := localfs.New()
	wk := filewalk.New(sc, filewalk.WithScanSize(1))
	err := wk.Walk(ctx,
		func(ctx context.Context, prefix string, info file.Info, err error) (skip, unchanged bool, children file.InfoList, returnErr error) {
			return true, false, nil, fmt.Errorf("oops")
		},
		nil,
		localTestTree,
	)
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("missing or unexpected error: %v", err)
	}

	err = wk.Walk(ctx,
		func(ctx context.Context, prefix string, info file.Info, err error) (skip, unchanged bool, children file.InfoList, returnErr error) {
			return false, false, nil, err
		},
		func(ctx context.Context, prefix string, unchanged bool, info file.Info, contents filewalk.Contents, ch <-chan filewalk.Contents) (file.InfoList, error) {
			//for c := range ch {
			//	_ = c
			//}
			return nil, fmt.Errorf("oh no")
		},
		localTestTree,
	)
	if err == nil || strings.Count(err.Error(), "oh no") != 1 {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

type infiniteScanner struct{}

func (is *infiniteScanner) Scan(context.Context, int) bool {
	time.Sleep(time.Hour)
	return true
}

func (is *infiniteScanner) Err() error {
	return nil
}

func (is *infiniteScanner) Contents() filewalk.Contents {
	return filewalk.Contents{}
}

func (is *infiniteScanner) Stat(context.Context, string) (file.Info, error) {
	info, err := os.Stat(localTestTree)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfo(info.Name(), info.Size(), info.Mode(), info.ModTime(), nil), nil
}

func (is *infiniteScanner) LStat(context.Context, string) (file.Info, error) {
	info, err := os.Lstat(localTestTree)
	if err != nil {
		return file.Info{}, err
	}
	return file.NewInfo(info.Name(), info.Size(), info.Mode(), info.ModTime(), nil), nil
}

func (is *infiniteScanner) LevelScanner(path string) filewalk.LevelScanner {
	return &infiniteScanner{}
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
	wk := filewalk.New(is, filewalk.WithScanSize(1), filewalk.WithConcurrency(4))
	lg := &logger{fs: is}

	ch := make(chan error)
	go func() {
		ch <- wk.Walk(ctx, lg.dirsFunc, lg.entriesFunc, "anywhere")
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

type slowFS struct {
	filewalk.FS
}

func (sfs slowFS) LevelScanner(path string) filewalk.LevelScanner {
	return &slowScanner{
		sc: sfs.FS.LevelScanner(path),
	}
}

type slowScanner struct {
	sc filewalk.LevelScanner
}

func (is *slowScanner) Scan(ctx context.Context, n int) bool {
	time.Sleep(time.Millisecond * 200)
	return is.sc.Scan(ctx, n)
}

func (is *slowScanner) Err() error {
	return is.sc.Err()
}

func (is *slowScanner) Contents() filewalk.Contents {
	return is.sc.Contents()
}

func TestReporting(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	is := &slowFS{localfs.New()}
	ch := make(chan filewalk.Status, 100)
	wk := filewalk.New(is, filewalk.WithScanSize(1), filewalk.WithConcurrency(2),
		filewalk.WithReporting(ch, time.Millisecond*100, time.Millisecond*250))
	lg := &logger{fs: is}
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_ = wk.Walk(ctx, lg.dirsFunc, lg.entriesFunc, localTestTree)
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

type dbScanner struct {
	sync.Mutex
	fs        filewalk.FS
	db        map[string]file.Info
	unchanged map[string]bool
	lines     []string
}

func (d *dbScanner) entriesFunc(ctx context.Context, prefix string, unchanged bool, fi file.Info, contents filewalk.Contents, ch <-chan filewalk.Contents) (file.InfoList, error) {
	d.Lock()
	defer d.Unlock()
	var children file.InfoList
	for _, de := range contents.Entries {
		path := d.fs.Join(prefix, de.Name)
		fi, err := d.fs.LStat(ctx, path)
		if err != nil {
			return nil, err
		}
		if _, ok := d.db[path]; !ok {
			d.db[path] = fi
		} else {
			existing := d.db[path]
			if fi.ModTime() == existing.ModTime() &&
				fi.Mode() == existing.Mode() {
				d.unchanged[path] = true
			}
		}
		if !de.IsDir() {
			d.lines = append(d.lines, path)
			continue
		}
		d.lines = append(d.lines, path+"/")
		children = children.AppendInfo(fi)
	}

	return children, nil
}

func (d *dbScanner) dirsFunc(_ context.Context, prefix string, fi file.Info, err error) (bool, bool, file.InfoList, error) {
	d.Lock()
	defer d.Unlock()
	if err != nil {
		return true, false, nil, nil
	}
	existing, ok := d.db[prefix]
	if !ok {
		d.db[prefix] = fi
		return false, false, nil, nil
	}
	unchanged := fi.ModTime() == existing.ModTime() &&
		fi.Mode() == existing.Mode()
	return false, unchanged, nil, nil
}

func (d *dbScanner) dirsAndFiles() (dirs, files, unchanged []string) {
	for _, line := range d.lines {
		if line[len(line)-1] == '/' {
			dirs = append(dirs, line)
		} else {
			files = append(files, line)
		}
	}
	for k, v := range d.unchanged {
		if v {
			unchanged = append(unchanged, k)
		}
	}
	d.lines = nil
	d.unchanged = map[string]bool{}
	return
}

func TestUnchanged(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)
	ctx := context.Background()
	sc := localfs.New()
	wk := filewalk.New(sc, filewalk.WithScanSize(1), filewalk.WithConcurrency(2))

	dbl := &dbScanner{
		fs:        sc,
		db:        map[string]file.Info{},
		unchanged: map[string]bool{}}

	// Use a separate copy of the test tree that can be modified without
	// affecting other tests.
	localTestTree := internal.CreateTestTree()
	defer os.RemoveAll(localTestTree)

	err := wk.Walk(ctx, dbl.dirsFunc, dbl.entriesFunc, localTestTree)
	if err != nil {
		t.Fatal(err)
	}
	dirs, files, unchanged := dbl.dirsAndFiles()

	if got, want := len(dirs), 8; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(files), 22; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(unchanged), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	err = wk.Walk(ctx, dbl.dirsFunc, dbl.entriesFunc, localTestTree)
	if err != nil {
		t.Fatal(err)
	}

	ndirs, nfiles, nunchanged := dbl.dirsAndFiles()

	sort.Strings(dirs)
	sort.Strings(ndirs)
	if got, want := ndirs, dirs; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(nfiles), 22; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(nunchanged), len(dirs)+len(files); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
