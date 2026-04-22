// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cloudeng.io/os/executil"
)

var sleepHelper string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "sleep-helper")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	sleepHelper = filepath.Join(tmpDir, "sleep")
	sleepHelper, err = executil.GoBuild(context.Background(), sleepHelper, "./testdata/sleep")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build sleep helper: %v\n", err)
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func TestAsyncWaitSuccessfulCommand(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	aw := executil.NewAsyncWait(cmd)
	if err := aw.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	done, err := aw.WaitDone()
	if !done {
		t.Error("expected done to be true after Wait()")
	}
	if err != nil {
		t.Errorf("unexpected error from WaitDone: %v", err)
	}
}

func TestAsyncWaitFailingCommand(t *testing.T) {
	cmd := exec.Command("false")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	aw := executil.NewAsyncWait(cmd)
	if err := aw.Wait(); err == nil {
		t.Error("expected non-nil error for failing command")
	}
	done, err := aw.WaitDone()
	if !done {
		t.Error("expected done to be true after Wait()")
	}
	if err == nil {
		t.Error("expected non-nil error from WaitDone after failure")
	}
}

func TestAsyncWaitDoneBeforeWait(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	aw := executil.NewAsyncWait(cmd)
	// Poll until done without calling Wait.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		done, err := aw.WaitDone()
		if done {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("timed out waiting for command to finish")
}

func TestAsyncWaitNotDoneWhileRunning(t *testing.T) {
	cmd := exec.Command(sleepHelper, "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	aw := executil.NewAsyncWait(cmd)
	done, err := aw.WaitDone()
	if done {
		t.Error("expected done to be false for long-running command")
	}
	if err != nil {
		t.Errorf("unexpected error from WaitDone while running: %v", err)
	}
	_ = cmd.Process.Kill()
}

func TestAsyncWaitMultipleWaiters(t *testing.T) {
	cmd := exec.Command(sleepHelper, "0.05")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	aw := executil.NewAsyncWait(cmd)

	errs := make(chan error, 3)
	for range 3 {
		go func() { errs <- aw.Wait() }()
	}
	for range 3 {
		if err := <-errs; err != nil {
			t.Errorf("unexpected error from concurrent Wait: %v", err)
		}
	}
}

func TestAsyncWaitWaitIdempotent(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	aw := executil.NewAsyncWait(cmd)

	for range 3 {
		if err := aw.Wait(); err != nil {
			t.Errorf("unexpected error on repeated Wait: %v", err)
		}
	}
}
