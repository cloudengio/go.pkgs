// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// mockFor returns the mock with the given id, which is how a test tells the VM
// it acquired from the pool apart from the rest of the pool's VMs.
func mockFor(t *testing.T, factory *vmstestutil.MockFactory, id string) *vmstestutil.Mock {
	t.Helper()
	for _, m := range factory.Mocks() {
		if m.ID() == id {
			return m
		}
	}
	t.Fatalf("no mock with id %q amongst the %v the factory created", id, len(factory.Mocks()))
	return nil
}

// TestPoolRefresh covers replacing the VMs that are available in a pool, which
// is how a pool picks up a new base image without being restarted: the VMs it
// was holding are deleted and the pool fills itself again with new ones.
func TestPoolRefresh(t *testing.T) {
	ctx := context.Background()
	const size = 3
	factory := vmstestutil.NewMockFactory(true)
	p := newPool(t, size, vmspool.StagingBehaviourSuspended, factory)

	replaced := factory.Mocks()
	if got, want := len(replaced), size; got != want {
		t.Fatalf("mocks before the refresh: got %v, want %v", got, want)
	}

	if got, want := p.Refresh(ctx), size; got != want {
		t.Fatalf("Refresh: got %v, want %v", got, want)
	}

	// The pool is full again once every replacement has been created, which
	// cannot happen until the VM it replaces has been deleted.
	requireSettled(t, p, size)
	allInState(t, replaced, vms.StateDeleted)

	mocks := factory.Mocks()
	if got, want := len(mocks), size*2; got != want {
		t.Fatalf("mocks after the refresh: got %v, want %v", got, want)
	}
	// The pool is holding the new VMs, not the ones it replaced.
	if got, want := countInState(mocks[size:], vms.StateDeleted), 0; got != want {
		t.Errorf("%v of the %v replacement VMs have been deleted", got, size)
	}
}

// TestPoolRefreshLeavesAcquiredVMs covers a refresh of a pool that has handed
// a VM out. That VM is in use by work that did not ask for a different image,
// so it is left alone and is not counted as replaced.
func TestPoolRefreshLeavesAcquiredVMs(t *testing.T) {
	ctx := context.Background()
	const size = 3
	factory := vmstestutil.NewMockFactory(true)
	p := newPool(t, size, vmspool.StagingBehaviourSuspended, factory)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	held := mockFor(t, factory, vm.ID())

	if got, want := p.Refresh(ctx), size-1; got != want {
		t.Fatalf("Refresh: got %v, want %v", got, want)
	}
	// The acquired VM still counts as acquired, and the pool refills around it.
	waitForStats(t, p, vmspool.Stats{Available: size - 1, Acquired: 1})
	if got := held.State(ctx); got == vms.StateDeleted {
		t.Errorf("the acquired VM was deleted by the refresh")
	}

	// It is the caller that finishes with it, as it would have been without
	// the refresh, and the pool fills the slot it gives up.
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
	requireSettled(t, p, size)
	if got, want := held.State(ctx), vms.StateDeleted; got != want {
		t.Errorf("state: got %v, want %v", got, want)
	}
}

// TestPoolRefreshNothingAvailable covers the refreshes that replace nothing: a
// pool whose VMs are all in use, and one that has been closed. Neither is an
// error, since a refresh only ever replaces what the pool is holding.
func TestPoolRefreshNothingAvailable(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	p := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if got := p.Refresh(ctx); got != 0 {
		t.Errorf("Refresh: got %v, want 0", got)
	}
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
	requireSettled(t, p, 1)

	// A closed pool has nothing available and hands nothing out, so a refresh
	// of one is a no-op rather than an attempt to acquire from it.
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := p.Refresh(ctx); got != 0 {
		t.Errorf("Refresh of a closed pool: got %v, want 0", got)
	}
}

// TestPoolRefreshDeleteFailure covers a VM that a refresh takes out of the pool
// but cannot delete. The pool is replenished for it either way, so the pool
// keeps its size, and the failure is reported through the status channel rather
// than returned, since the deletions outlive the call that asked for them.
func TestPoolRefreshDeleteFailure(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	statusCh := make(chan vmspool.Event, 100)
	p := vmspool.New(factory, testPoolName,
		vmspool.WithSize(1),
		vmspool.WithCreateBackoff(fastCreateBackoff()),
		vmspool.WithStatus(statusCh))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	requireSettled(t, p, 1)

	undeletable := factory.Mocks()[0]
	undeletable.DeleteErr = errors.New("this VM cannot be deleted")

	if got, want := p.Refresh(ctx), 1; got != want {
		t.Fatalf("Refresh: got %v, want %v", got, want)
	}
	waitForEvent(t, statusCh, vmspool.EventRefreshFailed, 5*time.Second)
	if got, want := undeletable.State(ctx), vms.StateDeleted; got == want {
		t.Errorf("state: got %v, want anything but %v", got, want)
	}
	// The pool is full again regardless: the slot was given up when the VM was
	// taken out of circulation, whatever became of the VM itself.
	requireSettled(t, p, 1)

	// The VM that could not be deleted is no longer the pool's to delete, so
	// closing it leaves the pool holding nothing.
	if err := p.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}
	requireEmpty(t, p)
}
