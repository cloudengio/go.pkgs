// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk_test

import (
	"context"
	"errors"
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
	sync.Mutex
	prefix   string
	fs       filewalk.FS
	linesMu  sync.Mutex
	lines    []string
	children map[string]file.InfoList
	state    map[string]int
	skip     string
	next     int
}

func (l *logger) appendLine(s string) {
	l.linesMu.Lock()
	defer l.linesMu.Unlock()
	l.lines = append(l.lines, s)
}

func (l *logger) Prefix(_ context.Context, state *int, prefix string, _ file.Info, err error) (bool, file.InfoList, error) {
	l.Lock()
	defer l.Unlock()
	if err != nil {
		l.appendLine(fmt.Sprintf("dir  : error: %v: %v\n", prefix, err))
		return true, nil, nil
	}
	prefix = strings.TrimPrefix(prefix, l.prefix)
	if len(l.skip) > 0 && prefix == l.skip {
		return true, nil, nil
	}
	*state = l.next
	l.state[prefix] = *state
	l.next++
	l.appendLine(fmt.Sprintf("%v* begin [%v]\n", prefix, *state))
	return false, l.children[prefix], nil
}

func (l *logger) Contents(ctx context.Context, state *int, prefix string, contents []filewalk.Entry, err error) (file.InfoList, error) {
	children := file.InfoList{}
	parent := strings.TrimPrefix(prefix, l.prefix)
	if err != nil {
		l.appendLine(fmt.Sprintf("%v: %v\n", parent,
			strings.Replace(err.Error(), l.prefix, "", 1)))
		return nil, nil
	}
	for _, de := range contents {
		link := ""
		info, err := l.fs.LStat(ctx, l.fs.Join(prefix, de.Name))
		if err != nil {
			return nil, err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			link = "@"
		}
		if de.IsDir() {
			children = append(children, info)
			continue
		}
		l.appendLine(fmt.Sprintf("%v%s: %v [%v]\n", l.fs.Join(parent, de.Name), link, info.Size(), *state))
	}
	return children, nil
}

func (l *logger) Done(_ context.Context, state *int, prefix string, _ error) error {
	prefix = strings.TrimPrefix(prefix, l.prefix)
	l.appendLine(fmt.Sprintf("%v* end [%v]\n", prefix, *state))
	return nil
}

func TestLocalWalk(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()
	sc := localfs.New()

	nl := func() *logger {
		return &logger{prefix: localTestTree,
			fs:       sc,
			children: map[string]file.InfoList{},
			state:    map[string]int{},
		}
	}
	lg := nl()
	wk := filewalk.New[int](sc, lg, filewalk.WithScanSize(1))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	lg = nl()
	wk = filewalk.New[int](sc, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedFull)

	lg = nl()
	wk = filewalk.New[int](sc, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
	lg.skip = strings.ReplaceAll("/b0/b0.1", "/", string(filepath.Separator))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedPartial1)

	lg = nl()
	wk = filewalk.New[int](sc, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
	lg.skip = strings.ReplaceAll("/b0", "/", string(filepath.Separator))
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedPartial2)

	lg = nl()
	wk = filewalk.New[int](sc, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))
	b01, err := sc.Stat(ctx, sc.Join(localTestTree, "b0", "b0.1"))
	if err != nil {
		t.Fatal(err)
	}
	// Replace /b0's children only with /b0.1 and not b0.0
	// Note: replacing / with filepath.Separator is required for windows.
	lg.children[strings.ReplaceAll("/b0", "/", string(filepath.Separator))] =
		file.InfoList{b01}
	testLocalWalk(ctx, t, localTestTree, wk, lg, expectedExistingChildren)
}

func testLocalWalk(ctx context.Context, t *testing.T, tmpDir string, wk *filewalk.Walker[int], lg *logger, expected string) {
	_, _, line, _ := runtime.Caller(1)
	err := wk.Walk(ctx, tmpDir)
	if err != nil {
		t.Errorf("line: %v: errors: %v", line, err)
	}
	sort.Strings(lg.lines)
	el := strings.Split(expected, "\n")
	for i, l := range lg.lines {
		state := ""
		if idx := strings.Index(l, "*"); idx >= 0 {
			p := l[:idx]
			state = fmt.Sprintf(" [%v]", lg.state[p])
		}
		if idx := strings.Index(l, ":"); idx > 0 {
			p := l[:idx]
			state = fmt.Sprintf(" [%v]", lg.state[filepath.Dir(p)])
		}
		if strings.Contains(l, "permission denied") {
			state = ""
		}
		if got, want := strings.TrimSpace(l), el[i]+state; got != want {
			t.Errorf("line: %v: got %v, want %v", line, got, want)
		}
	}
}

var expectedFull = `* begin
* end
/a0* begin
/a0* end
/a0/a0.0* begin
/a0/a0.0* end
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1* begin
/a0/a0.1* end
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0* begin
/b0* end
/b0/b0.0* begin
/b0/b0.0* end
/b0/b0.0/f0: 3
/b0/b0.0/f1: 3
/b0/b0.0/f2: 3
/b0/b0.1* begin
/b0/b0.1* end
/b0/b0.1/b1.0* begin
/b0/b0.1/b1.0* end
/b0/b0.1/b1.0/f0: 3
/b0/b0.1/b1.0/f1: 3
/b0/b0.1/b1.0/f2: 3
/inaccessible-dir* begin
/inaccessible-dir* end
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

// No b0.1 sub dir.
var expectedPartial1 = `* begin
* end
/a0* begin
/a0* end
/a0/a0.0* begin
/a0/a0.0* end
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1* begin
/a0/a0.1* end
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0* begin
/b0* end
/b0/b0.0* begin
/b0/b0.0* end
/b0/b0.0/f0: 3
/b0/b0.0/f1: 3
/b0/b0.0/f2: 3
/inaccessible-dir* begin
/inaccessible-dir* end
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

// No b0 sub dir.
var expectedPartial2 = `* begin
* end
/a0* begin
/a0* end
/a0/a0.0* begin
/a0/a0.0* end
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1* begin
/a0/a0.1* end
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/inaccessible-dir* begin
/inaccessible-dir* end
/inaccessible-dir: open /inaccessible-dir: permission denied
f0: 3
f1: 3
f2: 3
la0@: 2
la1@: 7
lf0@: 5
`

var expectedExistingChildren = `* begin
* end
/a0* begin
/a0* end
/a0/a0.0* begin
/a0/a0.0* end
/a0/a0.0/f0: 3
/a0/a0.0/f1: 3
/a0/a0.0/f2: 3
/a0/a0.1* begin
/a0/a0.1* end
/a0/a0.1/f0: 3
/a0/a0.1/f1: 3
/a0/a0.1/f2: 3
/a0/f0: 3
/a0/f1: 3
/a0/f2: 3
/a0/inaccessible-file: 3
/b0* begin
/b0* end
/b0/b0.1* begin
/b0/b0.1* end
/b0/b0.1/b1.0* begin
/b0/b0.1/b1.0* end
/b0/b0.1/b1.0/f0: 3
/b0/b0.1/b1.0/f1: 3
/b0/b0.1/b1.0/f2: 3
/inaccessible-dir* begin
/inaccessible-dir* end
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

type errorScanner struct {
	prefixErr     error
	contentsError error
	doneError     error
}

func (e *errorScanner) Prefix(_ context.Context, _ *int, _ string, _ file.Info, _ error) (bool, file.InfoList, error) {
	return false, nil, e.prefixErr
}

func (e *errorScanner) Contents(_ context.Context, _ *int, _ string, _ []filewalk.Entry, _ error) (file.InfoList, error) {
	return nil, e.contentsError
}

func (e *errorScanner) Done(_ context.Context, _ *int, _ string, err error) error {
	if err != nil {
		return err
	}
	return e.doneError
}

func TestFunctionErrors(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()
	sc := localfs.New()

	wk := filewalk.New[int](sc, &errorScanner{prefixErr: errors.New("oops")}, filewalk.WithScanSize(1))
	err := wk.Walk(ctx, localTestTree)
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Errorf("missing or unexpected error: %v", err)
	}

	wk = filewalk.New[int](sc, &errorScanner{contentsError: errors.New("oh no")}, filewalk.WithScanSize(1))
	err = wk.Walk(ctx, localTestTree)
	if err == nil || strings.Count(err.Error(), "oh no") != 1 {
		t.Errorf("missing or unexpected error: %v", err)
	}

	wk = filewalk.New[int](sc, &errorScanner{doneError: errors.New("one more")}, filewalk.WithScanSize(1))
	err = wk.Walk(ctx, localTestTree)
	if err == nil || strings.Count(err.Error(), "one more") != 1 {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

type infiniteScanner struct {
	filewalk.FS
	ctx       context.Context
	scanDelay time.Duration
	scanCh    chan struct{}
}

func (is *infiniteScanner) close(ch chan struct{}) {
	if ch != nil {
		close(ch)
	}
}

func (is *infiniteScanner) Scan(ctx context.Context, _ int) bool {
	is.close(is.scanCh)
	select {
	case <-time.After(is.scanDelay):
		return true
	case <-ctx.Done():
		return false
	}
}

func (is *infiniteScanner) Err() error {
	return nil
}

func (is *infiniteScanner) Contents() []filewalk.Entry {
	return nil
}

func (is *infiniteScanner) LevelScanner(_ string) filewalk.LevelScanner {
	return &infiniteScanner{
		scanDelay: is.scanDelay,
		scanCh:    is.scanCh,
	}
}

func TestCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	readyCh := make(chan struct{})
	is := &infiniteScanner{
		FS:        localfs.New(),
		scanDelay: time.Hour,
		scanCh:    readyCh,
	}
	lg := &logger{
		prefix:   localTestTree,
		fs:       is,
		children: map[string]file.InfoList{},
		state:    map[string]int{},
	}

	wk := filewalk.New[int](is, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(10))

	ch := make(chan error)
	go func() {
		ch <- wk.Walk(ctx, localTestTree)
	}()

	<-readyCh

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

func (is *slowScanner) Contents() []filewalk.Entry {
	return is.sc.Contents()
}

func TestReportingSlowScanner(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()
	is := &slowFS{localfs.New()}
	ch := make(chan filewalk.Status, 100)
	lg := &logger{
		fs:    is,
		state: map[string]int{},
	}
	wk := filewalk.New[int](is, lg, filewalk.WithScanSize(1), filewalk.WithConcurrency(2),
		filewalk.WithReporting(ch, time.Millisecond*100, time.Millisecond*250))

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_ = wk.Walk(ctx, localTestTree)
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
	wg.Wait()
}

type dbState bool

type dbScanner struct {
	sync.Mutex
	fs        filewalk.FS
	db        map[string]file.Info
	unchanged map[string]bool
	lines     []string
}

func (d *dbScanner) Contents(ctx context.Context, state *bool, prefix string, contents []filewalk.Entry, _ error) (file.InfoList, error) {
	d.Lock()
	defer d.Unlock()
	var children file.InfoList
	for _, de := range contents {
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
		children = append(children, fi)
	}
	return children, nil
}

func (d *dbScanner) Prefix(_ context.Context, state *bool, prefix string, fi file.Info, err error) (bool, file.InfoList, error) {
	d.Lock()
	defer d.Unlock()
	if err != nil {
		return true, nil, nil
	}
	existing, ok := d.db[prefix]
	if !ok {
		d.db[prefix] = fi
		return false, nil, nil
	}
	*state = fi.ModTime() == existing.ModTime() &&
		fi.Mode() == existing.Mode()
	return false, nil, nil
}

func (d *dbScanner) Done(_ context.Context, _ *bool, _ string, _ error) error {
	return nil
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
	defer synctestutil.AssertNoGoroutines(t)()
	ctx := context.Background()
	sc := localfs.New()
	dbl := &dbScanner{
		fs:        sc,
		db:        map[string]file.Info{},
		unchanged: map[string]bool{}}
	wk := filewalk.New[bool](sc, dbl, filewalk.WithScanSize(1), filewalk.WithConcurrency(2))

	// Use a separate copy of the test tree that can be modified without
	// affecting other tests.
	localTestTree := internal.CreateTestTree()
	defer os.RemoveAll(localTestTree)

	err := wk.Walk(ctx, localTestTree)
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

	err = wk.Walk(ctx, localTestTree)
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

type dummyError struct{}

func (d dummyError) Error() string {
	return "dummy error"
}

func TestError(t *testing.T) {
	err := &filewalk.Error{"/a/b/c", context.Canceled}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled")
	}

	if u := err.Unwrap(); u != context.Canceled {
		t.Errorf("expected context.Canceled")
	}

	var expected filewalk.Error
	if !errors.As(err, &expected) {
		t.Errorf("expected filewalk.Error")
	}

	if got, want := expected.Path, "/a/b/c"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := expected.Err, context.Canceled; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var other dummyError
	if errors.As(err, &other) {
		t.Errorf("expected filewalk.Error")
	}
}
