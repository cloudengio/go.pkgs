// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"cloudeng.io/sync/synctestutil"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// countersWithin polls the tracker until the named pool's count of kind has
// reached want, which is not immediate: a pool reports its events
// asynchronously and the tracker counts them on a goroutine of its own.
func countersWithin(t *testing.T, tr *vmspool.StatusTracker, pool string, kind vmspool.EventKind, want int) map[vmspool.EventKind]int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		counters, _ := tr.Counters(pool)
		if counters[kind] >= want {
			return counters
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for pool %v: %v counted %v times, want %v: %v", pool, kind, counters[kind], want, counters)
			return counters
		}
		time.Sleep(time.Millisecond)
	}
}

// TestStatusTrackerCounters covers the counting itself: the events of a pool
// are counted by kind, the time of the last of them is kept, and what a caller
// is given is a copy rather than the accounting itself.
func TestStatusTrackerCounters(t *testing.T) {
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 10)
	defer tr.Close()
	ch := make(chan vmspool.Event, 10)
	tr.Watch(ctx, ch)

	for _, e := range []vmspool.Event{
		{Pool: "one", Kind: vmspool.EventVMCreated, Time: time.Now()},
		{Pool: "one", Kind: vmspool.EventVMCreated, Time: time.Now()},
		// A failure a pool reports asynchronously, which is counted like any
		// other event and logged as the only report it will ever get.
		{Pool: "one", Kind: vmspool.EventReplenishFailed, VM: "vm-1",
			Err: errors.New("the replacement could not be created"), Time: time.Now()},
		{Pool: "one", Kind: vmspool.EventAcquired, Time: time.Now()},
	} {
		ch <- e
	}
	// Everything sent through one channel is counted in the order it was sent,
	// so the last event having been counted means that all of them have been.
	counters := countersWithin(t, tr, "one", vmspool.EventAcquired, 1)
	if got, want := counters[vmspool.EventVMCreated], 2; got != want {
		t.Errorf("%v: got %v, want %v", vmspool.EventVMCreated, got, want)
	}
	if got, want := counters[vmspool.EventAcquired], 1; got != want {
		t.Errorf("%v: got %v, want %v", vmspool.EventAcquired, got, want)
	}
	if got, want := counters[vmspool.EventReplenishFailed], 1; got != want {
		t.Errorf("%v: got %v, want %v", vmspool.EventReplenishFailed, got, want)
	}
	if _, updated := tr.Counters("one"); updated.IsZero() {
		t.Error("the time of the last event counted was not recorded")
	}

	counters[vmspool.EventRefreshFailed] = 99
	if again, _ := tr.Counters("one"); again[vmspool.EventRefreshFailed] != 0 {
		t.Error("Counters returned a mutable reference to the accounting itself")
	}
}

// TestStatusTrackerCountsPerPool covers one tracker serving several pools,
// which it can because every event names the pool it came from: the counters
// are kept apart, and a pool that has reported nothing reports nothing rather
// than another pool's figures.
func TestStatusTrackerCountsPerPool(t *testing.T) {
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 10)
	defer tr.Close()
	// A channel each, as a pool each would have, so the two pools are counted
	// from separate sources rather than one interleaved stream.
	one := make(chan vmspool.Event, 10)
	two := make(chan vmspool.Event, 10)
	tr.Watch(ctx, one)
	tr.Watch(ctx, two)

	one <- vmspool.Event{Pool: "one", Kind: vmspool.EventAcquired, Time: time.Now()}
	two <- vmspool.Event{Pool: "two", Kind: vmspool.EventVMCreated, Time: time.Now()}
	two <- vmspool.Event{Pool: "two", Kind: vmspool.EventVMCreated, Time: time.Now()}

	oneCounters := countersWithin(t, tr, "one", vmspool.EventAcquired, 1)
	if got := oneCounters[vmspool.EventVMCreated]; got != 0 {
		t.Errorf("one: %v counted %v times, want 0: they are pool two's", vmspool.EventVMCreated, got)
	}
	twoCounters := countersWithin(t, tr, "two", vmspool.EventVMCreated, 2)
	if got := twoCounters[vmspool.EventAcquired]; got != 0 {
		t.Errorf("two: %v counted %v times, want 0: it is pool one's", vmspool.EventAcquired, got)
	}
	if _, updated := tr.Counters("one"); updated.IsZero() {
		t.Error("one: the time of its own last event was not recorded")
	}
	if _, updated := tr.Counters("two"); updated.IsZero() {
		t.Error("two: the time of its own last event was not recorded")
	}

	// A pool that has reported nothing has an empty snapshot of its own, and
	// is not one of the pools the tracker has counted.
	counters, updated := tr.Counters("three")
	if len(counters) != 0 || !updated.IsZero() {
		t.Errorf("unreported pool: got %v at %v, want nothing", counters, updated)
	}
	if got, want := tr.Pools(), []string{"one", "two"}; !slices.Equal(got, want) {
		t.Errorf("Pools: got %v, want %v", got, want)
	}
}

// TestStatusTrackerNotifies covers the change signal: a subscriber is woken
// when a pool reports, which is what a user interface redraws on.
func TestStatusTrackerNotifies(t *testing.T) {
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 10)
	defer tr.Close()
	changed, release := tr.Subscribe(ctx)
	defer release()
	// Subscribed before anything is reported, so the signal cannot be missed.
	ch := make(chan vmspool.Event, 10)
	tr.Watch(ctx, ch)
	ch <- vmspool.Event{Pool: "one", Kind: vmspool.EventAcquired, Time: time.Now()}

	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a change notification")
	}
	// Releasing a subscription more than once is harmless.
	release()
	release()
}

// TestStatusTrackerTracksPool drives a real pool through the option the
// tracker hands out, which is the whole point of the type: the counters follow
// the pool's lifecycle without the pool knowing anything of the tracker, and
// they outlive the pool itself.
func TestStatusTrackerTracksPool(t *testing.T) {
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 100)
	defer tr.Close()
	p := vmspool.New(vmstestutil.NewMockFactory(true), testPoolName,
		vmspool.WithSize(1),
		vmspool.WithCreateBackoff(fastCreateBackoff()),
		tr.Track(ctx))
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
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// EventReleased is the last of the acquire-and-release sequence, so the
	// rest of it has been counted by the time it has.
	counters := countersWithin(t, tr, testPoolName, vmspool.EventReleased, 1)
	for _, kind := range []vmspool.EventKind{
		vmspool.EventVMCreated, vmspool.EventVMDequeued, vmspool.EventAcquired,
		vmspool.EventRelease, vmspool.EventReleased,
	} {
		if counters[kind] == 0 {
			t.Errorf("%v was not counted: %v", kind, counters)
		}
	}
	if _, updated := tr.Counters(testPoolName); updated.IsZero() {
		t.Error("the time of the pool's last event was not recorded")
	}
	// The pool is closed, and the tracker still reports what it did: the
	// counting is the tracker's, not the pool's.
	if got, want := tr.Pools(), []string{testPoolName}; !slices.Equal(got, want) {
		t.Errorf("Pools: got %v, want %v", got, want)
	}
}

// TestStatusTrackerClose covers what Close releases: the counting, the
// subscriptions, and the forwarding of every channel the tracker was given,
// including one belonging to an option that never reached a pool. What it does
// not release is the counters themselves.
func TestStatusTrackerClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, 5*time.Second)()
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 10)
	changed, release := tr.Subscribe(ctx)
	defer release()
	ch := make(chan vmspool.Event, 10)
	tr.Watch(ctx, ch)
	// An option that is never passed to a pool: nothing will ever be sent on
	// the channel behind it, and Close must release it regardless.
	_ = tr.Track(ctx)

	ch <- vmspool.Event{Pool: "one", Kind: vmspool.EventAcquired, Time: time.Now()}
	countersWithin(t, tr, "one", vmspool.EventAcquired, 1)

	tr.Close()
	// Close is idempotent, as it is for a set of pools that is closed twice.
	tr.Close()

	// A subscriber's channel is closed once what it had buffered has been
	// read, so draining it is what releases the subscription.
	for range changed { //nolint:revive // draining until closed is the point.
	}

	// Events that arrive after Close are discarded rather than counted, and
	// what was counted before it stands.
	ch <- vmspool.Event{Pool: "one", Kind: vmspool.EventAcquired, Time: time.Now()}
	counters, updated := tr.Counters("one")
	if got, want := counters[vmspool.EventAcquired], 1; got != want {
		t.Errorf("%v: got %v, want %v", vmspool.EventAcquired, got, want)
	}
	if updated.IsZero() {
		t.Error("a closed tracker forgot when its last event was")
	}
}

// TestStatusTrackerDefaultBuffer covers a tracker created with no buffer size
// of its own: the channels it hands out are buffered regardless, since an
// unbuffered one would have a pool dropping every event it emits.
func TestStatusTrackerDefaultBuffer(t *testing.T) {
	ctx := context.Background()
	tr := vmspool.NewStatusTracker(ctx, 0)
	defer tr.Close()
	p := vmspool.New(vmstestutil.NewMockFactory(true), testPoolName,
		vmspool.WithSize(1),
		vmspool.WithCreateBackoff(fastCreateBackoff()),
		tr.Track(ctx))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	countersWithin(t, tr, testPoolName, vmspool.EventVMCreated, 1)
	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
