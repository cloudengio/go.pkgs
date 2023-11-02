// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package asyncstat_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/asyncstat"
	"cloudeng.io/file/filewalk/internal"
	"cloudeng.io/file/filewalk/localfs"
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
		entries = append(entries, filewalk.Entry{Name: d.Name(), Type: d.Type()})
		var info os.FileInfo
		var err error
		if stat {
			info, err = os.Stat(filepath.Join(dir, d.Name()))
		} else {
			info, err = os.Lstat(filepath.Join(dir, d.Name()))
		}
		if err != nil {
			if stat {
				continue
			}
			t.Fatal(err)
		}
		infos = append(infos, file.NewInfoFromFileInfo(info))
	}
	return entries, infos
}

func verifyEntries(t *testing.T, children, all file.InfoList, wantAll file.InfoList) {
	_, _, line, _ := runtime.Caller(1)
	if got, want := all, wantAll; !reflect.DeepEqual(got, want) {
		t.Errorf("line %v, got %v, want %v", line, got, want)
	}
	wantNDirs := 0
	for _, i := range wantAll {
		if i.IsDir() {
			wantNDirs++
		}
	}
	wantNFiles := len(wantAll) - wantNDirs

	if got, want := len(children), wantNDirs; got != want {
		t.Errorf("line %v, got %v, want %v", line, got, want)
	}

	dirs := map[string]bool{}
	for _, c := range children {
		if got, want := c.IsDir(), true; got != want {
			t.Errorf("line %v, got %v, want %v", line, got, want)
		}
		dirs[c.Name()] = true
	}

	if got, want := len(all), wantNFiles+wantNDirs; got != want {
		t.Errorf("line %v, got %v, want %v", line, got, want)
	}
	ndirs, nfiles := 0, 0
	for _, e := range all {
		if got, want := dirs[e.Name()], e.IsDir(); got != want {
			t.Errorf("line %v, got %v, want %v", line, got, want)
		}
		if e.IsDir() {
			ndirs++
		} else {
			nfiles++
		}
	}
	if got, want := ndirs, wantNDirs; got != want {
		t.Errorf("line %v, got %v, want %v", line, got, want)
	}
	if got, want := nfiles, wantNFiles; got != want {
		t.Errorf("line %v, got %v, want %v", line, got, want)
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
	lt.took += time.Since(t)
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
		is := asyncstat.NewIssuer(fs,
			asyncstat.WithAsyncThreshold(threshold),
		)

		entries, infos := entriesFromDir(t, localTestTree, false)
		children, all, err := is.Process(ctx, localTestTree, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, children, all, infos)

		nl := fs.Join(localTestTree, "a0")
		entries, infos = entriesFromDir(t, nl, false)
		children, all, err = is.Process(ctx, nl, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, children, all, infos)

		// Stat
		latency := &latencyTracker{when: time.Now()}
		var errors []string
		is = asyncstat.NewIssuer(fs,
			asyncstat.WithAsyncThreshold(threshold),
			asyncstat.WithStat(),
			asyncstat.WithLatencyTracker(latency),
			asyncstat.WithErrorLogger(func(ctx context.Context, filename string, err error) {
				errors = append(errors, fmt.Sprintf("%v: %v", filename, err))
			}),
		)

		entries, infos = entriesFromDir(t, localTestTree, true)
		children, all, err = is.Process(ctx, localTestTree, entries)
		if err != nil {
			t.Fatalf("%v: %v", mode, err)
		}
		verifyEntries(t, children, all, infos)
		ep := fs.Join(localTestTree, "la1")
		if got, want := errors, []string{ep + ": stat " + ep + ": no such file or directory"}; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %v, want %v", mode, got, want)
		}

		if got, want := latency.finished, len(entries); got != want {
			t.Errorf("%v: got %v, want %v", mode, got, want)
		}

		if latency.took == 0 {
			t.Errorf("%v: got %v, want non-zero", mode, latency.took)
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
	is := asyncstat.NewIssuer(fs,
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
	verifyEntries(t, children, all, infos)
	t.Logf("is.Process ran in %v, total stat time was %v", took, latency.took)

	if took >= (latency.took / 10) {
		t.Fatalf("is.Process took (%v) longer than would be expected if async ops were issued (%v)", took, latency.took/10)
	}
}
