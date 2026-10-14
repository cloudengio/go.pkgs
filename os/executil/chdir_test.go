// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/os/executil"
)

// realDir returns a new temporary directory, with symbolic links resolved so
// that it can be compared with the result of os.Getwd.
func realDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// mustRestore calls restore and checks that it succeeds.
func mustRestore(t *testing.T, restore func() error) {
	t.Helper()
	if err := restore(); err != nil {
		t.Fatalf("restore: %v", err)
	}
}

func getwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	wd, err = filepath.EvalSymlinks(wd)
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func TestChdirLocked(t *testing.T) {
	start, target := realDir(t), realDir(t)
	t.Chdir(start) // restored when the test completes

	restore, err := executil.ChdirLocked(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := getwd(t); got != target {
		t.Errorf("got %v, want %v", got, target)
	}
	mustRestore(t, restore)
	if got := getwd(t); got != start {
		t.Errorf("after restore: got %v, want %v", got, start)
	}
}

func TestChdirLockedRelative(t *testing.T) {
	start := realDir(t)
	sub := filepath.Join(start, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(start)

	restore, err := executil.ChdirLocked("sub")
	if err != nil {
		t.Fatal(err)
	}
	if got := getwd(t); got != sub {
		t.Errorf("got %v, want %v", got, sub)
	}
	mustRestore(t, restore)
	if got := getwd(t); got != start {
		t.Errorf("after restore: got %v, want %v", got, start)
	}
}

// TestChdirLockedError checks that a directory that cannot be changed to is an
// error, that the working directory is left as it was, and that the lock is not
// held, which a later call, that would otherwise block, shows.
func TestChdirLockedError(t *testing.T) {
	start := realDir(t)
	t.Chdir(start)

	file := filepath.Join(start, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{
		"missing": filepath.Join(start, "does", "not", "exist"),
		"file":    file,
	} {
		restore, err := executil.ChdirLocked(dir)
		if err == nil || restore != nil {
			if restore != nil {
				_ = restore()
			}
			t.Fatalf("%s: got %v, %v, want an error and no restore function", name, err, restore != nil)
		}
		if got := getwd(t); got != start {
			t.Errorf("%s: the working directory is %v, want %v", name, got, start)
		}
	}

	other := realDir(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		restore, err := executil.ChdirLocked(other)
		if err != nil {
			t.Error(err)
			return
		}
		if err := restore(); err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock was held after an error")
	}
	if got := getwd(t); got != start {
		t.Errorf("the working directory is %v, want %v", got, start)
	}
}

// TestChdirLockedBlocks checks that a second call waits for the first to be
// restored, and then proceeds.
func TestChdirLockedBlocks(t *testing.T) {
	start, first, second := realDir(t), realDir(t), realDir(t)
	t.Chdir(start)

	restoreFirst, err := executil.ChdirLocked(first)
	if err != nil {
		t.Fatal(err)
	}
	var entered atomic.Bool
	finished := make(chan string, 1)
	go func() {
		restore, err := executil.ChdirLocked(second)
		if err != nil {
			t.Error(err)
			finished <- ""
			return
		}
		entered.Store(true)
		wd := getwdUnlocked()
		if err := restore(); err != nil {
			t.Error(err)
		}
		finished <- wd
	}()

	time.Sleep(100 * time.Millisecond)
	if entered.Load() {
		t.Fatal("the second call did not wait")
	}
	if got := getwd(t); got != first {
		t.Errorf("the working directory is %v, want %v", got, first)
	}
	mustRestore(t, restoreFirst)
	select {
	case wd := <-finished:
		if wd != second {
			t.Errorf("the second call had %v, want %v", wd, second)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the second call did not proceed")
	}
	if got := getwd(t); got != start {
		t.Errorf("after both: got %v, want %v", got, start)
	}
}

// getwdUnlocked is getwd for a goroutine other than the test's, and returns
// the working directory with symbolic links resolved.
func getwdUnlocked() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(wd); err == nil {
		return resolved
	}
	return wd
}

// holdAndCheck repeatedly holds the lock for dir, checking that it is the
// working directory while it is, and records the most that hold it at once.
func holdAndCheck(dir string, iterations int, holders, maxHolders *atomic.Int32, errs chan<- string) {
	for range iterations {
		restore, err := executil.ChdirLocked(dir)
		if err != nil {
			errs <- err.Error()
			return
		}
		n := holders.Add(1)
		for {
			m := maxHolders.Load()
			if n <= m || maxHolders.CompareAndSwap(m, n) {
				break
			}
		}
		if got := getwdUnlocked(); got != dir {
			errs <- "got " + got + ", want " + dir
		}
		time.Sleep(50 * time.Microsecond)
		if got := getwdUnlocked(); got != dir {
			errs <- "after sleeping, got " + got + ", want " + dir
		}
		holders.Add(-1)
		if err := restore(); err != nil {
			errs <- err.Error()
		}
	}
}

// TestChdirLockedConcurrent checks that, while it is held, each caller sees its
// own directory, and that no two callers hold it at once.
func TestChdirLockedConcurrent(t *testing.T) {
	start := realDir(t)
	t.Chdir(start)

	const goroutines, iterations = 8, 40
	dirs := make([]string, goroutines)
	for i := range dirs {
		dirs[i] = realDir(t)
	}
	var holders, maxHolders atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan string, goroutines*iterations*3)
	for _, dir := range dirs {
		wg.Go(func() { holdAndCheck(dir, iterations, &holders, &maxHolders, errs) })
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if got := maxHolders.Load(); got != 1 {
		t.Errorf("%d callers held the lock at once", got)
	}
	if got := getwd(t); got != start {
		t.Errorf("after all: got %v, want %v", got, start)
	}
}

// TestChdirLockedRestoreFails checks that the error is returned, and the lock
// is released, if the original directory cannot be returned to, as when it has
// been removed.
func TestChdirLockedRestoreFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a working directory cannot be removed on windows")
	}
	scratch := realDir(t)
	original := filepath.Join(scratch, "original")
	if err := os.Mkdir(original, 0o700); err != nil {
		t.Fatal(err)
	}
	target := realDir(t)
	t.Chdir(scratch)
	if err := os.Chdir(original); err != nil {
		t.Fatal(err)
	}

	restore, err := executil.ChdirLocked(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	// Cannot return to original, which is reported, and the lock is not held.
	err = restore()
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, want an error that the directory does not exist", err)
	}
	if got := getwd(t); got != target {
		t.Errorf("the working directory is %v, want %v, since it could not be changed", got, target)
	}
	// And a second call has no effect, and no error.
	if err := restore(); err != nil {
		t.Errorf("a second restore returned %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		restore, err := executil.ChdirLocked(target)
		if err != nil {
			t.Error(err)
			return
		}
		if err := restore(); err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock was held after a restore that failed")
	}
}

// TestChdirLockedRestoreTwice checks that restoring more than once is harmless:
// it does not panic, and does not release the lock, or change the working
// directory, for a caller that has acquired it since.
func TestChdirLockedRestoreTwice(t *testing.T) {
	start, first, second := realDir(t), realDir(t), realDir(t)
	t.Chdir(start)

	restoreFirst, err := executil.ChdirLocked(first)
	if err != nil {
		t.Fatal(err)
	}
	mustRestore(t, restoreFirst)
	mustRestore(t, restoreFirst) // nothing is held, and this must not panic or fail
	if got := getwd(t); got != start {
		t.Errorf("got %v, want %v", got, start)
	}

	restoreSecond, err := executil.ChdirLocked(second)
	if err != nil {
		t.Fatal(err)
	}
	mustRestore(t, restoreFirst) // the second holds the lock, which this must not release
	if got := getwd(t); got != second {
		t.Errorf("a second restore changed the working directory to %v, want %v", got, second)
	}
	var entered atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		restore, err := executil.ChdirLocked(first)
		if err != nil {
			t.Error(err)
			return
		}
		entered.Store(true)
		if err := restore(); err != nil {
			t.Error(err)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	if entered.Load() {
		t.Fatal("a caller acquired the lock while it was held, after a second restore")
	}
	mustRestore(t, restoreSecond)
	mustRestore(t, restoreSecond)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the lock was not released")
	}
	if got := getwd(t); got != start {
		t.Errorf("got %v, want %v", got, start)
	}
}
