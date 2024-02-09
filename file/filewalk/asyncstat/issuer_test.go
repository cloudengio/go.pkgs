// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package asyncstat_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/asyncstat"
	"cloudeng.io/file/internal"
	"cloudeng.io/file/localfs"
)

var localTestTree string

func TestMain(m *testing.M) {
	localTestTree = internal.CreateTestTree()
	code := m.Run()
	if code == 0 {
		os.RemoveAll(localTestTree)
	} else {
		fmt.Printf("test tree left at: %v\n", localTestTree)
	}
	os.Exit(code)
}

func entriesFromDir(t *testing.T, dir string, stat bool) ([]filewalk.Entry, []file.Info) {
	t.Helper()
	de, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries := make([]filewalk.Entry, 0, len(de))
	infos := make([]file.Info, 0, len(de))
	for _, d := range de {
		filename := filepath.Join(dir, d.Name())
		entries = append(entries, filewalk.Entry{Name: d.Name(), Type: d.Type()})
		var info os.FileInfo
		var err error
		if stat {
			info, err = os.Stat(filename)
		} else {
			info, err = os.Lstat(filename)
		}

		if err != nil {
			if stat {
				continue
			}
			t.Fatal(err)
		}
		finfo := file.NewInfoFromFileInfo(info)
		if info.Mode()&os.ModeSymlink != 0 {
			// symlinks are annoyinglyg different on windows so we use
			// readlink for all platforms.
			buf, err := os.Readlink(filename)
			if err != nil {
				t.Fatal(err)
			}
			finfo = file.NewInfo(
				info.Name(),
				int64(len(buf)),
				info.Mode(),
				info.ModTime(),
				info.Sys())
		}
		infos = append(infos, finfo)
	}
	return entries, infos
}

func verifyEntries(t *testing.T, mode string, children, all file.InfoList, wantAll file.InfoList) {
	_, _, line, _ := runtime.Caller(1)
	if err := filetestutil.CompareFileInfo(all, wantAll); err != nil {
		t.Errorf("line %v: mode %v, %v", line, mode, err)
	}
	wantNDirs := 0
	for _, i := range wantAll {
		if i.IsDir() {
			wantNDirs++
		}
	}
	wantNFiles := len(wantAll) - wantNDirs

	if got, want := len(children), wantNDirs; got != want {
		t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
	}

	dirs := map[string]bool{}
	for _, c := range children {
		if got, want := c.IsDir(), true; got != want {
			t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
		}
		dirs[c.Name()] = true
	}

	if got, want := len(all), wantNFiles+wantNDirs; got != want {
		t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
	}
	ndirs, nfiles := 0, 0
	for _, e := range all {
		if got, want := dirs[e.Name()], e.IsDir(); got != want {
			t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
		}
		if e.IsDir() {
			ndirs++
		} else {
			nfiles++
		}
	}
	if got, want := ndirs, wantNDirs; got != want {
		t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
	}
	if got, want := nfiles, wantNFiles; got != want {
		t.Errorf("line %v: mode %v, got %v, want %v", mode, line, got, want)
	}
}

type latencyTracker struct {
	sync.Mutex
	started, finished int
	when              time.Time
	took              time.Duration
}

func (lt *latencyTracker) Before() time.Time {
	lt.Lock()
	defer lt.Unlock()
	lt.started++
	return lt.when
}

func (lt *latencyTracker) After(t time.Time) {
	lt.Lock()
	defer lt.Unlock()
	lt.finished++
	s := time.Since(t)
	lt.took += s
}

func TestIssue(t *testing.T) {
	ctx := context.Background()
	fs := localfs.New()

	for threshold := range []int{0, 1000} {
		mode := "sync"
		if threshold == 0 {
			mode = "async"
		}
		// Lstat
		is := asyncstat.New(fs,
			asyncstat.WithAsyncThreshold(threshold),
		)

		entries, infos := entriesFromDir(t, localTestTree, false)
		children, all, err := is.Process(ctx, localTestTree, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, mode, children, all, infos)

		nl := fs.Join(localTestTree, "a0")
		entries, infos = entriesFromDir(t, nl, false)
		children, all, err = is.Process(ctx, nl, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, mode, children, all, infos)

		// Stat
		latency := &latencyTracker{when: time.Now()}
		statErrors := map[string]error{}
		is = asyncstat.New(fs,
			asyncstat.WithAsyncThreshold(threshold),
			asyncstat.WithStat(),
			asyncstat.WithLatencyTracker(latency),
			asyncstat.WithErrorLogger(func(_ context.Context, filename string, err error) {
				statErrors[filename] = err
			}),
		)

		entries, infos = entriesFromDir(t, localTestTree, true)
		children, all, err = is.Process(ctx, localTestTree, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, mode, children, all, infos)
		ep := fs.Join(localTestTree, "la1")
		if got, want := len(statErrors), 1; got != want {
			t.Errorf("%v: got %v, want %v", mode, got, want)
		}
		if got, want := errors.Is(statErrors[ep], os.ErrNotExist), true; got != want {
			t.Errorf("%v: got %v, want %v", mode, got, want)
		}

		if got, want := latency.finished, len(entries); got != want {
			t.Errorf("%v: got %v, want %v", mode, got, want)
		}
	}
}

func TestASyncIssue(t *testing.T) {
	ctx := context.Background()
	fs := localfs.New()
	tmpdir := t.TempDir()

	for i := 0; i < 1000; i++ {
		buf := make([]byte, i+1)
		if err := os.WriteFile(fs.Join(tmpdir, fmt.Sprintf("file%v", i)), buf, 0600); err != nil {
			t.Fatal(err)
		}
		if i%10 == 0 {
			if err := os.Mkdir(fs.Join(tmpdir, fmt.Sprintf("dir%v", i)), 0700); err != nil {
				t.Fatal(err)
			}
		}
	}

	latency := &latencyTracker{when: time.Now()}
	is := asyncstat.New(fs,
		asyncstat.WithAsyncThreshold(0),
		asyncstat.WithLatencyTracker(latency),
	)

	entries, infos := entriesFromDir(t, tmpdir, false)

	start := time.Now()
	children, all, err := is.Process(ctx, tmpdir, entries)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	verifyEntries(t, "async", children, all, infos)
	t.Logf("is.Process ran in %v, total stat time was %v", took, latency.took)

	if took >= (latency.took / 10) {
		t.Fatalf("is.Process took (%v) longer than would be expected if async ops were issued (%v)", took, latency.took/10)
	}
}
