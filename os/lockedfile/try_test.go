// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build unix

package lockedfile_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cloudeng.io/os/lockedfile"
	"cloudeng.io/os/lockedfile/internal/testenv"
)

func TestTryEditSameProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.lock")

	f1, ok, err := lockedfile.TryEdit(path)
	if err != nil || !ok {
		t.Fatalf("first TryEdit: ok=%v err=%v", ok, err)
	}

	// A second attempt while held must return ok=false with no error.
	if f2, ok, err := lockedfile.TryEdit(path); err != nil || ok {
		if f2 != nil {
			_ = f2.Close()
		}
		t.Fatalf("second TryEdit: ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	// After release, a new lock succeeds.
	if err := f1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	f3, ok, err := lockedfile.TryEdit(path)
	if err != nil || !ok {
		t.Fatalf("TryEdit after close: ok=%v err=%v", ok, err)
	}
	_ = f3.Close()
}

func TestMutexTryLockSameProcess(t *testing.T) {
	mu := lockedfile.MutexAt(filepath.Join(t.TempDir(), "m.lock"))
	unlock, ok, err := mu.TryLock()
	if err != nil || !ok {
		t.Fatalf("first TryLock: ok=%v err=%v", ok, err)
	}
	if u2, ok, err := mu.TryLock(); err != nil || ok {
		if u2 != nil {
			u2()
		}
		t.Fatalf("second TryLock: ok=%v err=%v, want ok=false", ok, err)
	}
	unlock()
	unlock2, ok, err := mu.TryLock()
	if err != nil || !ok {
		t.Fatalf("TryLock after unlock: ok=%v err=%v", ok, err)
	}
	unlock2()
}

// TestMutexTryLockNonBlocking verifies TryLock returns promptly (does not block
// on the internal mutex) when the Mutex is already held by a blocking Lock in
// the same process.
func TestMutexTryLockNonBlocking(t *testing.T) {
	mu := lockedfile.MutexAt(filepath.Join(t.TempDir(), "n.lock"))
	unlock, err := mu.Lock()
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	defer unlock()

	type result struct {
		ok  bool
		err error
	}
	done := make(chan result, 1)
	go func() {
		u, ok, err := mu.TryLock()
		if u != nil {
			u()
		}
		done <- result{ok, err}
	}()

	select {
	case r := <-done:
		if r.err != nil || r.ok {
			t.Fatalf("TryLock while held: ok=%v err=%v, want ok=false err=nil", r.ok, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TryLock blocked while the mutex was held")
	}
}

// lockHelperEnv, when set, turns the test binary into a helper that acquires the
// named lock, signals readiness by creating <lock>.ready, and holds it until its
// stdin is closed.
const lockHelperEnv = "LOCKEDFILE_TRY_HELPER"

// TestTryLockHelper is the subprocess entry point for TestTryLockCrossProcess.
// It is a no-op unless invoked with lockHelperEnv set.
func TestTryLockHelper(t *testing.T) {
	path := os.Getenv(lockHelperEnv)
	if path == "" {
		t.Skip("helper: not invoked as a subprocess")
	}
	f, ok, err := lockedfile.TryEdit(path)
	if err != nil || !ok {
		t.Fatalf("helper TryEdit: ok=%v err=%v", ok, err)
	}

	if err := os.WriteFile(path+".ready", nil, 0o644); err != nil { //nolint:gosec // G306: Expect WriteFile permissions to be 0600 or less
		t.Fatalf("helper ready: %v", err)
	}
	buf := make([]byte, 1)
	_, _ = os.Stdin.Read(buf) // block until the parent closes stdin.
	_ = f.Close()
}

func TestTryLockCrossProcess(t *testing.T) {
	testenv.MustHaveExec(t)
	path := filepath.Join(t.TempDir(), "cross.lock")

	cmd := exec.Command(os.Args[0], "-test.run=^TestTryLockHelper$", "-test.timeout=60s") //nolint:gosec // os.Args[0] is the test binary.
	cmd.Env = append(os.Environ(), lockHelperEnv+"="+path)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stdin.Close(); _ = cmd.Wait() }()

	ready := path + ".ready"
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child never acquired the lock")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// The parent must not be able to acquire the lock the child holds.
	if f, ok, err := lockedfile.TryEdit(path); err != nil || ok {
		if f != nil {
			_ = f.Close()
		}
		t.Fatalf("parent TryEdit: ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	// Release the child, then the lock must be acquirable.
	_ = stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child exited with error: %v", err)
	}
	f, ok, err := lockedfile.TryEdit(path)
	if err != nil || !ok {
		t.Fatalf("TryEdit after child exit: ok=%v err=%v", ok, err)
	}
	_ = f.Close()
}
