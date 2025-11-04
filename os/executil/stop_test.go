// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/os/executil"
)

func startStoppable(ctx context.Context, t *testing.T, hang bool, out *bytes.Buffer) *exec.Cmd {
	t.Helper()
	tmpDir := t.TempDir()
	tmpDir, _ = os.MkdirTemp(tmpDir, "stoppable")
	binary, err := executil.GoBuild(ctx, filepath.Join(tmpDir, "stoppable"), "./testdata/stoppable")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{}
	if hang {
		args = append(args, "-hang")
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	forscanning := &bytes.Buffer{}
	cmd.Stdout = io.MultiWriter(out, forscanning)
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(forscanning)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "pid: ") {
			break
		}
	}
	return cmd
}

func TestSignalAndWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping signal handling tests on windows")
	}
	ctx := context.Background()
	// Test graceful shutdown.
	t.Run("graceful", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := startStoppable(ctx, t, false, out)
		err := executil.SignalAndWait(ctx, time.Second*2, cmd, os.Interrupt)
		if err != nil {
			t.Fatalf("SignalAndWait failed: %v", err)
		}
		if !executil.IsStopped(cmd.Process.Pid) {
			t.Fatalf("process %d is not stopped", cmd.Process.Pid)
		}
	})

	// Timeout
	t.Run("timeout", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := startStoppable(ctx, t, true, out)
		time.Sleep(time.Second)
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()
		err := executil.SignalAndWait(ctx, 100*time.Millisecond, cmd, os.Interrupt)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected a timeout, got %v", err)
		}
	})

	// Test forced shutdown.
	t.Run("forced", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := startStoppable(ctx, t, true, out)
		err := executil.SignalAndWait(ctx, 100*time.Millisecond, cmd, os.Interrupt, syscall.SIGKILL)
		if err != nil {
			t.Fatalf("SignalAndWait failed: %v", err)
		}
		if !executil.IsStopped(cmd.Process.Pid) {
			t.Fatalf("process %d is not stopped", cmd.Process.Pid)
		}
	})
}

func TestIsStopped(t *testing.T) {
	// Test non-existent process.

	t.Run("non-existent", func(t *testing.T) {
		// Find a PID that is not running.
		pid := 65535
		for !executil.IsStopped(pid) {
			pid--
			if pid == 0 {
				t.Fatal("could not find a non-existent PID")
			}
		}
	})
}
