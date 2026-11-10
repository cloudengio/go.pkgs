// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"testing"

	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// TestEventKindString guards against an EventKind being added, or inserted
// between existing ones, without a corresponding name.
func TestEventKindString(t *testing.T) {
	for e := vmspool.EventAcquireWaiting; e <= vmspool.EventRefreshFailed; e++ {
		if got := e.String(); got == "" || got == "Unknown" {
			t.Errorf("EventKind(%d): missing name: got %q", int(e), got)
		}
	}
	for _, e := range []vmspool.EventKind{-1, vmspool.EventRefreshFailed + 1} {
		if got, want := e.String(), "Unknown"; got != want {
			t.Errorf("EventKind(%d): got %v, want %v", int(e), got, want)
		}
	}
}

// TestEventAttribution covers what every event says about where it came from:
// the name of the pool, so that a consumer of a channel shared by several
// pools can tell them apart, and the ID of the VM, so that the events of one
// VM can be followed without parsing the text of an error. An event about the
// pool itself names no VM.
func TestEventAttribution(t *testing.T) {
	ctx := context.Background()
	statusCh := make(chan vmspool.Event, 100)
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, testPoolName,
		vmspool.WithSize(1),
		vmspool.WithCreateBackoff(fastCreateBackoff()),
		vmspool.WithStatus(statusCh))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	acquired := vm.ID()
	if err := vm.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Close has waited for everything that emits, so nothing is left to send
	// and the events can be read as a closed sequence.
	close(statusCh)

	// The VM that was acquired is the subject of each of these, from the
	// moment it left the pool to the moment the caller was done with it.
	aboutAcquiredVM := map[vmspool.EventKind]bool{
		vmspool.EventVMDequeued: true,
		vmspool.EventAcquired:   true,
		vmspool.EventRelease:    true,
		vmspool.EventReleased:   true,
	}
	// These are about the pool: its capacity, or a VM that does not exist yet.
	aboutPool := map[vmspool.EventKind]bool{
		vmspool.EventStartPoolFull:    true,
		vmspool.EventVMCreateStarted:  true,
		vmspool.EventAcquireWaiting:   true,
		vmspool.EventReplenishStarted: true,
		vmspool.EventReplenished:      true,
	}
	seen := map[vmspool.EventKind]int{}
	for e := range statusCh {
		seen[e.Kind]++
		if e.Pool != testPoolName {
			t.Errorf("%v: pool %q, want %q", e.Kind, e.Pool, testPoolName)
		}
		switch {
		case aboutAcquiredVM[e.Kind]:
			if e.VM != acquired {
				t.Errorf("%v: vm %q, want the acquired VM %q", e.Kind, e.VM, acquired)
			}
		case aboutPool[e.Kind]:
			if e.VM != "" {
				t.Errorf("%v: names vm %q, want none: it is about the pool", e.Kind, e.VM)
			}
		case e.Kind == vmspool.EventVMCreated:
			// Each creation names the VM it created, which is the acquired one
			// for the first and its replacement for the second.
			if e.VM == "" {
				t.Errorf("%v: names no vm", e.Kind)
			}
		}
	}
	for _, kind := range []vmspool.EventKind{
		vmspool.EventVMCreated, vmspool.EventVMDequeued, vmspool.EventAcquired,
		vmspool.EventRelease, vmspool.EventReleased,
	} {
		if seen[kind] == 0 {
			t.Errorf("no %v event was emitted", kind)
		}
	}
}
