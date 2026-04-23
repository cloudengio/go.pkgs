// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vms_test

import (
	"strings"
	"testing"

	"cloudeng.io/vms"
)

func TestTransitionHappyPaths(t *testing.T) {
	steps := []struct {
		from   vms.State
		action vms.Action
		want   vms.State
	}{
		// Clone lifecycle
		{vms.StateInitial, vms.ActionClone, vms.StateCloning},
		// Start from Stopped
		{vms.StateStopped, vms.ActionStart, vms.StateStarting},
		// Waiting no-ops
		{vms.StateStarting, vms.ActionNone, vms.StateStarting},
		{vms.StateStopping, vms.ActionNone, vms.StateStopping},
		{vms.StateSuspending, vms.ActionNone, vms.StateSuspending},
		{vms.StateDeleting, vms.ActionNone, vms.StateDeleting},
		// Stop
		{vms.StateRunning, vms.ActionStop, vms.StateStopping},
		// Suspend
		{vms.StateRunning, vms.ActionSuspend, vms.StateSuspending},
		// Start from Suspended
		{vms.StateSuspended, vms.ActionStart, vms.StateStarting},
		// Delete from Stopped, Suspended
		{vms.StateStopped, vms.ActionDelete, vms.StateDeleting},
		{vms.StateSuspended, vms.ActionDelete, vms.StateDeleting},
		// Clone from Deleted
		{vms.StateDeleted, vms.ActionClone, vms.StateCloning},
	}
	for _, tc := range steps {
		got, ok := tc.from.Transition(tc.action)
		if !ok {
			t.Errorf("Transition(%v, %v): unexpected failure", tc.from, tc.action)
			continue
		}
		if got != tc.want {
			t.Errorf("Transition(%v, %v): got %v, want %v", tc.from, tc.action, got, tc.want)
		}
	}
}

func TestTransitionInvalid(t *testing.T) {
	invalid := []struct {
		from   vms.State
		action vms.Action
	}{
		{vms.StateInitial, vms.ActionStart},
		{vms.StateInitial, vms.ActionStop},
		{vms.StateRunning, vms.ActionClone},
		{vms.StateRunning, vms.ActionDelete},
		{vms.StateDeleted, vms.ActionStart},
	}
	for _, tc := range invalid {
		got, ok := tc.from.Transition(tc.action)
		if ok {
			t.Errorf("Transition(%v, %v): expected failure, got next state %v", tc.from, tc.action, got)
		}
		if got != tc.from {
			t.Errorf("Transition(%v, %v): on failure state should be unchanged, got %v", tc.from, tc.action, got)
		}
	}
}

func TestAllowed(t *testing.T) {
	if !vms.StateRunning.Allowed(vms.ActionStop) {
		t.Error("Running+Stop should be allowed")
	}
	if vms.StateRunning.Allowed(vms.ActionDelete) {
		t.Error("Running+Delete should not be allowed")
	}
}

func TestValidActions(t *testing.T) {
	if len(vms.StateInitial.ValidActions()) == 0 {
		t.Error("Initial should have valid actions")
	}
	if len(vms.StateDeleted.ValidActions()) == 0 {
		t.Error("Deleted should have valid actions (Clone)")
	}
}

func TestPrintStates(t *testing.T) {
	var sb strings.Builder
	vms.PrintStates(&sb)
	out := sb.String()
	for _, want := range []string{"Initial", "Running", "Deleted", "Clone", "Stop", "Suspend"} {
		if !strings.Contains(out, want) {
			t.Errorf("PrintStates output missing %q", want)
		}
	}
	t.Log(out)
}
