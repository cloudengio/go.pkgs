// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloudeng.io/os/executil"
	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
)

// InstanceTestConfig configures the test suite for an implementation of vms.Instance.
type InstanceTestConfig struct {
	// Constructor creates a new uninitialized vms.Instance for each test.
	Constructor vmspool.Constructor

	// Timeout caps individual operations. Defaults to 30 s.
	Timeout time.Duration

	// ExecCmd is a command that should succeed inside a running VM. If empty,
	// the Exec subtest is skipped.
	ExecCmd    string
	ExecArgs   []string
	ExecStdout string // Expected output from the exec.
	ExecStderr string // Expected stderr output from the exec.

	// RequireUnderlyingState is an optional helper for tests that need to verify
	// the underlying state of the instance, e.g. by querying a cloud provider API.
	// The function is expected to wait for the instance to reach a stable state
	RequireUnderlyingState func(
		ctx context.Context, inst vms.Instance, msg string, final vms.State, intermediate ...vms.State) error
}

func (c InstanceTestConfig) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30 * time.Second
}

// TestInstanceCloneStartStopDelete verifies the standard Clone -> Start -> Stop -> Delete state transitions.
func TestInstanceCloneStartStopDelete(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor.New()
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
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor.New()
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
	if cfg.ExecCmd == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor.New()
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
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor.New()
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

func TestInstanceStateErrors(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	t.Helper()

	inst := cfg.Constructor.New()
	runErr, stopErr := inst.Stop(context.Background(), cfg.timeout())
	if runErr == nil && stopErr == nil {
		t.Fatalf("instance should not be running at start of test")
	}
	err := inst.Delete(context.Background())
	if err == nil {
		t.Fatalf("instance should not be deleted at start of test")
	}
	if err := vms.CleanupVM(context.Background(), inst, cfg.timeout()); err != nil {
		t.Fatalf("cleanup before test: %v", err)
	}
}

func logStep(t TestingT, format string, args ...any) func() {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	t.Logf("→ %s", msg)
	start := time.Now()
	doneLog := func() {
		t.Logf("  ✓ %s (%.1fs)", msg, time.Since(start).Seconds())
	}
	return doneLog
}

func newWriter(format string, args ...any) io.Writer { //cicd:ignore
	msg := fmt.Sprintf(format, args...)
	return executil.NewLabelingWriter(os.Stderr, []byte(fmt.Sprintf("→→ %s", msg)), '\n')
}

// TestInstanceLifecycle is a detailed lifecycle test that walks a VM through
// its state machine.
// Initial → Clone → Stopped →
// Start → Running → Stop → Stopped → Stop (idempotent) →
// Start → Running → [Suspend → Suspended → Suspend (idempotent) → Start → Running →]
// Stop → Stopped → Delete → Deleted
func TestInstanceLifecycle(t TestingT, cfg InstanceTestConfig) { //cicd:astest
	if cfg.RequireUnderlyingState == nil {
		t.Fatalf("RequireUnderlyingState function must be provided for lifecycle test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout())
	defer cancel()

	inst := cfg.Constructor.New()

	requireState := func(ctx context.Context, inst vms.Instance, msg string, final vms.State, intermediate ...vms.State) {
		t.Helper()
		err := cfg.RequireUnderlyingState(ctx, inst, msg, final, intermediate...)
		if err != nil {
			t.Fatalf("state check: %v", err)
		}
	}

	requireState(ctx, inst, "initial", vms.StateInitial, vms.StateInitial)
	props, err := inst.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties in Initial state: %v", err)
	}
	done := logStep(t, "clone %s → %s", props.CloneInfo, inst.ID())
	checkErr := func(action string, err error) {
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}

	err = inst.Clone(ctx)
	checkErr("Clone", err)
	done()
	requireState(ctx, inst, "clone", vms.StateStopped, vms.StateCloning, vms.StateInitial)

	done = logStep(t, "run")
	err = inst.Start(ctx, newWriter("run-stdout-stopped"), newWriter("run-stderr-stopped"))
	checkErr("Run", err)
	done()
	requireState(ctx, inst, "run",
		vms.StateRunning,
		vms.StateStopped, vms.StateStarting)

	done = logStep(t, "stop")
	runErr, stopErr := inst.Stop(ctx, time.Minute)
	checkErr("Stop", runErr)
	checkErr("Stop", stopErr)
	done()
	requireState(ctx, inst, "stop",
		vms.StateStopped,
		vms.StateRunning, vms.StateStopping)

	done = logStep(t, "stop again (idempotency)")
	runErr, stopErr = inst.Stop(ctx, time.Minute)
	checkErr("Stop (idempotent)", runErr)
	checkErr("Stop (idempotent)", stopErr)
	done()
	requireState(ctx, inst, "stop idempotent", vms.StateStopped)

	time.Sleep(time.Second)
	done = logStep(t, "run again from stopped")
	err = inst.Start(ctx, newWriter("run-stdout"), newWriter("run-stderr"))
	checkErr("Start (second)", err)
	done()
	requireState(ctx, inst, "run again from stopped",
		vms.StateRunning,
		vms.StateRunning, vms.StateStopped)

	if inst.Suspendable() {
		done = logStep(t, "suspend")
		err = inst.Suspend(ctx)
		checkErr("Suspend", err)
		done()
		requireState(ctx, inst, "suspend",
			vms.StateSuspended,
			vms.StateRunning, vms.StateSuspending)

		done = logStep(t, "suspend again (idempotency)")
		err = inst.Suspend(ctx)
		checkErr("Suspend (idempotent)", err)
		done()
		requireState(ctx, inst, "suspend idempotent", vms.StateSuspended)

		done = logStep(t, "run again from suspended")
		err = inst.Start(ctx, newWriter("run-stdout-suspended"), newWriter("run-stderr-suspended"))
		checkErr("Start (from suspended)", err)
		done()
		requireState(ctx, inst, "run again from suspended",
			vms.StateRunning,
			vms.StateSuspended, vms.StateStarting)
	}

	done = logStep(t, "stop before delete")
	runErr, stopErr = inst.Stop(ctx, time.Minute)
	checkErr("Stop (before delete)", runErr)
	checkErr("Stop (before delete)", stopErr)
	done()
	requireState(ctx, inst, "stop before delete",
		vms.StateStopped,
		vms.StateRunning, vms.StateStopping)

	done = logStep(t, "delete")
	err = inst.Delete(ctx)
	checkErr("Delete", err)
	done()
	requireState(ctx, inst, "delete",
		vms.StateDeleted,
		vms.StateDeleting)
}
