// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"cloudeng.io/os/executil"
)

// selfPID is the pid AcquireRunLock records for this process.
func selfPID() string { return strconv.Itoa(os.Getpid()) }

func TestAcquireRunLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock: %v", err)
	}
	defer unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	if got, want := strings.TrimSpace(string(data)), selfPID(); got != want {
		t.Errorf("recorded pid: got %q, want %q", got, want)
	}
}

// TestAcquireRunLockCreatesDir verifies that the directories leading to the lock
// are created, so callers need not pre-create the per-user config directory.
func TestAcquireRunLockCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock: %v", err)
	}
	defer unlock()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("lock file was not created: %v", err)
	}
}

// TestAcquireRunLockAlreadyHeld verifies that a second acquisition is refused
// with ErrAlreadyRunning, and that the error names the holding pid and the lock
// path so the user can find the other instance.
func TestAcquireRunLockAlreadyHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("first AcquireRunLock: %v", err)
	}
	defer unlock()

	second, err := executil.AcquireRunLock(path)
	if err == nil {
		t.Fatal("second AcquireRunLock: got nil error, want ErrAlreadyRunning")
	}
	if second != nil {
		t.Error("second AcquireRunLock returned an unlock function alongside the error")
	}
	if !errors.Is(err, executil.ErrAlreadyRunning) {
		t.Errorf("got %v, want it to wrap ErrAlreadyRunning", err)
	}
	if !strings.Contains(err.Error(), selfPID()) {
		t.Errorf("error %q does not name the holding pid %v", err, selfPID())
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the lock path %v", err, path)
	}
}

// TestAcquireRunLockReleased verifies that unlock frees the lock for the next
// caller, and that the next caller overwrites the previous pid rather than
// appending to it.
func TestAcquireRunLockReleased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("first AcquireRunLock: %v", err)
	}
	unlock()

	again, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock after unlock: %v", err)
	}
	defer again()

	if got, want := readPID(t, path), selfPID(); got != want {
		t.Errorf("recorded pid: got %q, want %q", got, want)
	}
}

// TestAcquireRunLockTruncatesStalePID verifies that a longer pid left by a
// previous holder is truncated away rather than leaving trailing digits behind.
func TestAcquireRunLockTruncatesStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")
	stale := strings.Repeat("9", len(selfPID())+8)
	if err := os.WriteFile(path, []byte(stale+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock: %v", err)
	}
	defer unlock()

	if got, want := readPID(t, path), selfPID(); got != want {
		t.Errorf("recorded pid: got %q, want %q: stale content was not truncated", got, want)
	}
}

// TestAcquireRunLockEmptyLockFile verifies that a lock file with no pid, as left
// by an interrupted write, still yields a usable error naming the path.
func TestAcquireRunLockEmptyLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock: %v", err)
	}
	defer unlock()

	// Blank the pid behind the holder's back.
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = executil.AcquireRunLock(path)
	if !errors.Is(err, executil.ErrAlreadyRunning) {
		t.Fatalf("got %v, want it to wrap ErrAlreadyRunning", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the lock path %v", err, path)
	}
}

// TestAcquireRunLockUncreatableDir verifies that a path whose parent cannot be
// created is reported rather than panicking or silently succeeding.
func TestAcquireRunLockUncreatableDir(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blocker, "sub", "run.lock")

	unlock, err := executil.AcquireRunLock(path)
	if err == nil {
		unlock()
		t.Fatal("got nil error, want a failure creating the lock directory")
	}
	if errors.Is(err, executil.ErrAlreadyRunning) {
		t.Errorf("got %v, want a directory error rather than ErrAlreadyRunning", err)
	}
}

// TestAcquireRunLockCrossProcess verifies the lock across processes, which is
// what it exists for: the in-process tests above exercise the same file lock,
// but not the case it is actually defending against. The child re-executes this
// test binary and reports its outcome through its exit code.
func TestAcquireRunLockCrossProcess(t *testing.T) {
	if path := os.Getenv("EXECUTIL_TEST_RUNLOCK_PATH"); path != "" {
		unlock, err := executil.AcquireRunLock(path)
		switch {
		case err == nil:
			unlock()
			os.Exit(0)
		case errors.Is(err, executil.ErrAlreadyRunning):
			os.Exit(3)
		default:
			os.Exit(4)
		}
	}

	path := filepath.Join(t.TempDir(), "run.lock")
	child := func() error {
		cmd := exec.Command(os.Args[0], "-test.run=TestAcquireRunLockCrossProcess") //nolint:gosec // re-executing this test binary
		cmd.Env = append(os.Environ(), "EXECUTIL_TEST_RUNLOCK_PATH="+path)
		return cmd.Run()
	}

	unlock, err := executil.AcquireRunLock(path)
	if err != nil {
		t.Fatalf("AcquireRunLock: %v", err)
	}

	// While this process holds the lock the child must be refused.
	err = child()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("child while locked: got %v, want it to exit non-zero", err)
	}
	if got, want := exitErr.ExitCode(), 3; got != want {
		t.Errorf("child while locked: exit code %d, want %d (ErrAlreadyRunning)", got, want)
	}

	// Once released the child acquires it, and releases it again on exit.
	unlock()
	if err := child(); err != nil {
		t.Fatalf("child after unlock: got %v, want it to acquire the lock", err)
	}
	if got, want := readPID(t, path), selfPID(); got == want {
		t.Errorf("recorded pid %v is this process, want the child's", got)
	}
}

func TestUserConfigDirPath(t *testing.T) {
	got, err := executil.UserConfigDirPath(filepath.Join("some-app", "run.lock"))
	if err != nil {
		t.Fatalf("UserConfigDirPath: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("got %q, want an absolute path", got)
	}
	if want := filepath.Join("some-app", "run.lock"); !strings.HasSuffix(got, want) {
		t.Errorf("got %q, want it to end with %q", got, want)
	}
}

func readPID(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	return strings.TrimSpace(string(data))
}
