// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

var (
	errRun  = errors.New("run failed")
	errStop = errors.New("stop failed")
)

// TestVMStopAndRelease exercises VM.StopAndRelease's handling of the (runErr,
// stopErr) pair returned
// by the underlying instance: the VM is marked stopped only when stopErr is nil,
// and both errors are propagated to the caller unchanged.
func TestVMStopAndRelease(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name        string
		stopRunErr  error
		stopErr     error
		wantRunErr  error
		wantStopErr error
		wantStopped bool
		wantState   vms.State
	}{
		{"success", nil, nil, nil, nil, true, vms.StateStopped},
		{"run error only", errRun, nil, errRun, nil, true, vms.StateErrorUnknown},
		{"stop error only", nil, errStop, nil, errStop, false, vms.StateErrorUnknown},
		{"both errors", errRun, errStop, errRun, errStop, false, vms.StateErrorUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := vmstestutil.NewMock("vm-" + tc.name)
			m.SetState(vms.StateRunning)
			m.StopRunErr = tc.stopRunErr
			m.StopErr = tc.stopErr
			v := vmspool.NewTestVM(m)

			runErr, stopErr := v.StopAndRelease(ctx, time.Second)
			if !errors.Is(runErr, tc.wantRunErr) {
				t.Errorf("runErr: got %v, want %v", runErr, tc.wantRunErr)
			}
			if !errors.Is(stopErr, tc.wantStopErr) {
				t.Errorf("stopErr: got %v, want %v", stopErr, tc.wantStopErr)
			}
			if got := v.Stopped(); got != tc.wantStopped {
				t.Errorf("stopped: got %v, want %v", got, tc.wantStopped)
			}
			if got := m.State(ctx); got != tc.wantState {
				t.Errorf("mock state: got %v, want %v", got, tc.wantState)
			}
		})
	}
}

// TestPoolStopReplenishes verifies that StopAndRelease replenishes the pool, and
// that the Delete that follows it does not replenish a second time.
func TestPoolStopReplenishes(t *testing.T) {
	ctx := context.Background()
	statusCh := make(chan vmspool.Event, 64)

	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourRunning),
		vmspool.WithStatus(statusCh))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if runErr, stopErr := vm.StopAndRelease(ctx, time.Second); runErr != nil || stopErr != nil {
		t.Fatalf("StopAndRelease: got (%v, %v), want (nil, nil)", runErr, stopErr)
	}

	// The replacement is created by StopAndRelease, before Delete is called.
	waitForEvent(t, statusCh, vmspool.EventReplenished, 5*time.Second)
	if got, want := len(factory.Mocks()), 2; got != want {
		t.Errorf("mocks after StopAndRelease: got %d, want %d", got, want)
	}

	if err := vm.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Close waits for any replenishment goroutine Delete started, so the mock
	// count afterwards is a reliable test for a second replenishment.
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got, want := len(factory.Mocks()), 2; got != want {
		t.Errorf("mocks after Delete: got %d, want %d: Delete replenished a pool already replenished by StopAndRelease", got, want)
	}
	allDeleted(t, factory.Mocks())
}

// TestPoolStopIdempotentReplenish verifies that repeated StopAndRelease calls
// replenish the pool once only.
func TestPoolStopIdempotentReplenish(t *testing.T) {
	ctx := context.Background()
	statusCh := make(chan vmspool.Event, 64)

	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourRunning),
		vmspool.WithStatus(statusCh))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	for range 3 {
		if _, stopErr := vm.StopAndRelease(ctx, time.Second); stopErr != nil {
			t.Fatalf("StopAndRelease: %v", stopErr)
		}
	}
	waitForEvent(t, statusCh, vmspool.EventReplenished, 5*time.Second)
	if err := vm.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got, want := len(factory.Mocks()), 2; got != want {
		t.Errorf("mocks after 3 StopAndRelease calls: got %d, want %d", got, want)
	}
	allDeleted(t, factory.Mocks())
}

// TestPoolStagedStoppedVMReplenishesOnDelete verifies that a VM staged stopped
// or suspended in the pool, and hence started by Acquire, is treated as running:
// it is Delete, not the staging state, that replenishes the pool.
func TestPoolStagedStoppedVMReplenishesOnDelete(t *testing.T) {
	ctx := context.Background()
	for _, behaviour := range []vmspool.StagingBehaviour{
		vmspool.StagingBehaviourStopped,
		vmspool.StagingBehaviourSuspended,
	} {
		t.Run(behaviour.String(), func(t *testing.T) {
			statusCh := make(chan vmspool.Event, 64)
			factory := vmstestutil.NewMockFactory(true)
			p := vmspool.New(factory,
				vmspool.WithSize(1),
				vmspool.WithStagingBehaviour(behaviour),
				vmspool.WithStatus(statusCh))
			if err := p.Start(ctx); err != nil {
				t.Fatalf("Start: %v", err)
			}
			vm, err := p.Acquire(ctx)
			if err != nil {
				t.Fatalf("Acquire: %v", err)
			}
			if err := vm.Delete(ctx); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			waitForEvent(t, statusCh, vmspool.EventReplenished, 5*time.Second)
			if err := p.Close(ctx); err != nil {
				t.Fatalf("Close: %v", err)
			}
			if got, want := len(factory.Mocks()), 2; got != want {
				t.Errorf("mocks after Release: got %d, want %d", got, want)
			}
		})
	}
}

func allDeleted(t *testing.T, mocks []*vmstestutil.Mock) {
	t.Helper()
	for i, m := range mocks {
		if got := m.State(context.Background()); got != vms.StateDeleted {
			t.Errorf("mock[%d]: state = %s, want %s", i, got, vms.StateDeleted)
		}
	}
}

// TestVMStopInvalid verifies that StopAndRelease reports a stopErr, and leaves
// runErr nil, when the VM has no underlying instance.
func TestVMStopInvalid(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		vm   *vmspool.VM
	}{
		{"nil inst", &vmspool.VM{}},
		{"nil Instance", vmspool.NewTestVM(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runErr, stopErr := tc.vm.StopAndRelease(ctx, time.Second)
			if runErr != nil {
				t.Errorf("runErr: got %v, want nil", runErr)
			}
			if stopErr == nil {
				t.Fatal("stopErr: got nil, want an error")
			}
			if !strings.Contains(stopErr.Error(), "invalid VM instance") {
				t.Errorf("stopErr: got %q, want it to mention invalid VM instance", stopErr)
			}
		})
	}
}

// TestVMStopNotRunning verifies that a stop error from the instance (here, an
// attempt to stop a VM that is not running) is propagated and the VM is not
// marked stopped.
func TestVMStopNotRunning(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock("suspended")
	m.SetState(vms.StateSuspended)
	v := vmspool.NewTestVM(m)

	runErr, stopErr := v.StopAndRelease(ctx, time.Second)
	if runErr != nil {
		t.Errorf("runErr: got %v, want nil", runErr)
	}
	if stopErr == nil {
		t.Fatal("stopErr: got nil, want an error for stopping a non-running VM")
	}
	if v.Stopped() {
		t.Error("stopped: got true, want false when stop failed")
	}
}

// TestVMStopAlreadyStopped documents that VM.StopAndRelease defers the
// already-stopped policy to the underlying instance: the mock treats it as a
// no-op success, so it returns (nil, nil) and marks the VM stopped.
func TestVMStopAlreadyStopped(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock("stopped")
	m.SetState(vms.StateStopped)
	v := vmspool.NewTestVM(m)

	runErr, stopErr := v.StopAndRelease(ctx, time.Second)
	if runErr != nil || stopErr != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", runErr, stopErr)
	}
	if !v.Stopped() {
		t.Error("stopped: got false, want true")
	}
}
