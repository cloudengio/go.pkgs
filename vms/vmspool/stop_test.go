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

// TestVMStop exercises VM.Stop's handling of the (runErr, stopErr) pair returned
// by the underlying instance: the VM is marked stopped only when stopErr is nil,
// and both errors are propagated to the caller unchanged.
func TestVMStop(t *testing.T) {
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

			runErr, stopErr := v.Stop(ctx, time.Second)
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

// TestVMStopInvalid verifies that Stop reports a stopErr, and leaves runErr nil,
// when the VM has no underlying instance.
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
			runErr, stopErr := tc.vm.Stop(ctx, time.Second)
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

	runErr, stopErr := v.Stop(ctx, time.Second)
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

// TestVMStopAlreadyStopped documents that VM.Stop defers the already-stopped
// policy to the underlying instance: the mock treats it as a no-op success, so
// Stop returns (nil, nil) and marks the VM stopped.
func TestVMStopAlreadyStopped(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock("stopped")
	m.SetState(vms.StateStopped)
	v := vmspool.NewTestVM(m)

	runErr, stopErr := v.Stop(ctx, time.Second)
	if runErr != nil || stopErr != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", runErr, stopErr)
	}
	if !v.Stopped() {
		t.Error("stopped: got false, want true")
	}
}
