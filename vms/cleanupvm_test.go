// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vms_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmstestutil"
)

func TestCleanupVMInitialOrDeleted(t *testing.T) {
	ctx := context.Background()
	for _, state := range []vms.State{vms.StateInitial, vms.StateDeleted} {
		m := vmstestutil.NewMock()
		m.SetState(state)
		if err := vms.CleanupVM(ctx, m); err != nil {
			t.Errorf("state %v: expected no error, got %v", state, err)
		}
		// Should not have been changed.
		if m.State(ctx) != state {
			t.Errorf("state %v changed to %v", state, m.State(ctx))
		}
	}
}

func TestCleanupVMRunningSuccess(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock()
	m.SetState(vms.StateRunning)
	if err := vms.CleanupVM(ctx, m); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if got := m.State(ctx); got != vms.StateDeleted {
		t.Errorf("expected state Deleted, got %v", got)
	}
}

func TestCleanupVMRunningStopFailure(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock()
	m.SetState(vms.StateRunning)
	m.StopErr = errors.New("stop failed")
	err := vms.CleanupVM(ctx, m)
	if err == nil || !strings.Contains(err.Error(), "cleanup: failed to stop VM") {
		t.Errorf("expected stop error, got %v", err)
	}
}

func TestCleanupVMRunningStopWrongState(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock()
	m.SetState(vms.StateRunning)
	badState := vms.StateErrorUnknown
	m.StopState = &badState
	err := vms.CleanupVM(ctx, m)
	if err == nil || !strings.Contains(err.Error(), "cleanup: expected VM to be stopped after stopping") {
		t.Errorf("expected wrong state error, got %v", err)
	}
}

func TestCleanupVMStoppedSuspendedErrorUnknown(t *testing.T) {
	ctx := context.Background()
	for _, state := range []vms.State{vms.StateStopped, vms.StateSuspended, vms.StateErrorUnknown} {
		m := vmstestutil.NewMock()
		m.SetState(state)
		if err := vms.CleanupVM(ctx, m); err != nil {
			t.Errorf("state %v: expected no error, got %v", state, err)
		}
		if got := m.State(ctx); got != vms.StateDeleted {
			t.Errorf("state %v: expected state Deleted, got %v", state, got)
		}
	}
}

func TestCleanupVMStoppedDeleteFailure(t *testing.T) {
	ctx := context.Background()
	m := vmstestutil.NewMock()
	m.SetState(vms.StateStopped)
	m.DeleteErr = errors.New("delete failed")
	err := vms.CleanupVM(ctx, m)
	if err == nil || !strings.Contains(err.Error(), "cleanup: failed to delete VM") {
		t.Errorf("expected delete error, got %v", err)
	}
}

func TestCleanupVMUnexpectedState(t *testing.T) {
	ctx := context.Background()
	for _, state := range []vms.State{vms.StateCloning, vms.StateStarting, vms.StateStopping, vms.StateSuspending, vms.StateDeleting} {
		m := vmstestutil.NewMock()
		m.SetState(state)
		err := vms.CleanupVM(ctx, m)
		if err == nil || !strings.Contains(err.Error(), "cleanup: unexpected VM state") {
			t.Errorf("state %v: expected unexpected state error, got %v", state, err)
		}
	}
}
