// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"bytes"
	"context"
	"io"
	"time"

	"cloudeng.io/vms"
)

// InstanceTestConfig configures the test suite for an implementation of vms.Instance.
type InstanceTestConfig struct {
	// Constructor creates a new uninitialized vms.Instance for each test.
	Constructor func() vms.Instance

	// Timeout caps individual operations. Defaults to 30 s.
	Timeout time.Duration

	// ExecCmd is a command that should succeed inside a running VM. If empty,
	// the Exec subtest is skipped.
	ExecCmd    string
	ExecArgs   []string
	ExecStdout string // Expected output from the exec.
	ExecStderr string // Expected stderr output from the exec.
}

func (c InstanceTestConfig) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

// TestInstanceLifecycle runs the full vms.Instance lifecycle test suite.
func TestInstanceLifecycle(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()
	TestInstanceCloneStartStopDelete(t, cfg)
	TestInstanceSuspendResume(t, cfg)
	TestInstanceExec(t, cfg)
	TestInstanceDeleteFromSuspended(t, cfg)
}

// TestInstanceCloneStartStopDelete verifies the standard Clone -> Start -> Stop -> Delete state transitions.
func TestInstanceCloneStartStopDelete(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor()
	if got := inst.State(ctx); got != vms.StateInitial {
		t.Errorf("expected state %s, got %s", vms.StateInitial, got)
	}

	t.Cleanup(func() {
		_ = vms.CleanupVM(context.Background(), inst, cfg.timeout())
	})

	if err := inst.Clone(ctx); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if got := inst.State(ctx); got != vms.StateStopped {
		t.Errorf("expected state %s, got %s", vms.StateStopped, got)
	}

	var stdout, stderr bytes.Buffer
	if err := inst.Start(ctx, &stdout, &stderr); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got := inst.State(ctx); got != vms.StateRunning {
		t.Errorf("expected state %s, got %s", vms.StateRunning, got)
	}

	// Test Properties when Running
	if _, err := inst.Properties(ctx); err != nil {
		t.Errorf("Properties: %v", err)
	}

	runErr, stopErr := inst.Stop(ctx, cfg.timeout())
	if runErr != nil || stopErr != nil {
		t.Fatalf("Stop: runErr=%v, stopErr=%v", runErr, stopErr)
	}
	if got := inst.State(ctx); got != vms.StateStopped {
		t.Errorf("expected state %s, got %s", vms.StateStopped, got)
	}

	if err := inst.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := inst.State(ctx); got != vms.StateDeleted {
		t.Errorf("expected state %s, got %s", vms.StateDeleted, got)
	}
}

// TestInstanceSuspendResume verifies the Suspend and Resume (Start) transitions for suspendable VMs.
func TestInstanceSuspendResume(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor()
	t.Cleanup(func() {
		_ = vms.CleanupVM(context.Background(), inst, cfg.timeout())
	})

	if !inst.Suspendable() {
		return
	}

	if err := inst.Clone(ctx); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := inst.Suspend(ctx); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if got := inst.State(ctx); got != vms.StateSuspended {
		t.Errorf("expected state %s, got %s", vms.StateSuspended, got)
	}

	// Start from Suspended state
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		t.Fatalf("Resume (Start): %v", err)
	}
	if got := inst.State(ctx); got != vms.StateRunning {
		t.Errorf("expected state %s, got %s", vms.StateRunning, got)
	}
}

// TestInstanceExec verifies that a command can be executed inside a running VM.
func TestInstanceExec(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()
	if cfg.ExecCmd == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor()
	t.Cleanup(func() {
		_ = vms.CleanupVM(context.Background(), inst, cfg.timeout())
	})

	if err := inst.Clone(ctx); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := inst.Exec(ctx, &stdout, &stderr, cfg.ExecCmd, cfg.ExecArgs...); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if got, want := stdout.String(), cfg.ExecStdout; got != want {
		t.Errorf("Exec stdout: got %q, want %q", got, want)
	}
	if got, want := stderr.String(), cfg.ExecStderr; got != want {
		t.Errorf("Exec stderr: got %q, want %q", got, want)
	}
}

// TestInstanceDeleteFromSuspended verifies that an instance can be deleted directly from the Suspended state.
func TestInstanceDeleteFromSuspended(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor()
	t.Cleanup(func() {
		_ = vms.CleanupVM(context.Background(), inst, cfg.timeout())
	})

	if !inst.Suspendable() {
		return
	}
	if err := inst.Clone(ctx); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if err := inst.Start(ctx, io.Discard, io.Discard); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := inst.Suspend(ctx); err != nil {
		t.Fatalf("Suspend: %v", err)
	}
	if err := inst.Delete(ctx); err != nil {
		t.Fatalf("Delete from Suspended: %v", err)
	}
	if got := inst.State(ctx); got != vms.StateDeleted {
		t.Errorf("expected state %s, got %s", vms.StateDeleted, got)
	}
}
