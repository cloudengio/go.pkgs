// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

const (
	outcomeSuccess = "success"
	outcomeFailure = "failure"
)

// completed is a CompletedVM holding the VM a piece of work ran on, or none.
type completed struct {
	vm *vmspool.VM
}

func (c completed) GetVM() *vmspool.VM { return c.vm }

func (c completed) GetLogger(l *slog.Logger) *slog.Logger { return l }

// quietContext returns a context whose logger discards everything, so that the
// queue's reports of expiring work do not appear in the test output.
func quietContext(t *testing.T) context.Context {
	t.Helper()
	return ctxlog.WithLogger(context.Background(), slog.New(slog.DiscardHandler))
}

// newQueue returns a queue with a success and a failure outcome, each holding
// up to 10 events for the given retention.
func newQueue(ctx context.Context, retention time.Duration) *vmspool.CompletionQueue {
	return vmspool.NewCompletionQueue(ctx,
		vmspool.WithCompletionOutcome(outcomeSuccess, 10, retention),
		vmspool.WithCompletionOutcome(outcomeFailure, 10, retention))
}

// closeWithin closes the queue, failing if that takes longer than within. The
// queue releases each VM it is holding as it drains, so a close that needs a
// lock held for the whole drain would deadlock rather than merely be slow,
// which is what the bound catches.
func closeWithin(t *testing.T, q *vmspool.CompletionQueue, within time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- q.Close(context.Background()) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	case <-time.After(within):
		t.Fatalf("Close did not return within %v", within)
	}
}

// acquireVM returns a VM from a new pool of the given size, which is closed
// when the test ends.
func acquireVM(t *testing.T, factory *vmstestutil.MockFactory, size int) *vmspool.VM {
	t.Helper()
	p := newPool(t, size, vmspool.StagingBehaviourSuspended, factory)
	vm, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	return vm
}

// holdsVM reports whether the queue is holding the VM with the given ID, which
// is what a caller reporting on a VM it can no longer find in its pool asks of
// the set the queue publishes.
func holdsVM(q *vmspool.CompletionQueue, vmID string) bool {
	return slices.Contains(q.VMIDs(), vmID)
}

// awaitSignal waits for the queue to report that the set of VMs it holds has
// changed.
func awaitSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("no notification: %v", msg)
	}
}

// TestCompletionQueueFromVM covers how work finds the queue to park the VM it
// ran on: the pool reports the queue it was configured with, and so does every
// VM that pool hands out, since a job that has finished has nothing but its VM
// to go on.
func TestCompletionQueueFromVM(t *testing.T) {
	ctx := quietContext(t)
	q := newQueue(ctx, time.Hour)
	// Closing the queue is what deletes the VM parked below, and it runs before
	// the pool's own cleanup, which requires that the pool was left empty.
	defer closeWithin(t, q, 30*time.Second)

	factory := vmstestutil.NewMockFactory(true)
	p := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory, vmspool.WithCompletionQueue(q))
	if got := p.CompletionQueue(); got != q {
		t.Errorf("Pool.CompletionQueue: got %v, want the configured queue %v", got, q)
	}

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	cq := vm.CompletionQueue()
	if cq != q {
		t.Fatalf("VM.CompletionQueue: got %v, want the pool's queue %v", cq, q)
	}
	// The VM is parked through the queue it reported, which is the whole point
	// of reporting it.
	cq.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	if got, want := q.NumEventsPerPool(vm.PoolName()), 1; got != want {
		t.Errorf("NumEventsPerPool(%q): got %v, want %v", vm.PoolName(), got, want)
	}
	if !holdsVM(q, vm.ID()) {
		t.Errorf("the queue does not report holding %v", vm.ID())
	}
}

// TestCompletionQueueUnset covers a pool configured without a queue, which is
// an application that deletes each VM as it finishes with it. Its VMs report no
// queue rather than a dead one, so that work can tell that there is nowhere to
// park them.
func TestCompletionQueueUnset(t *testing.T) {
	ctx := quietContext(t)
	p := newPool(t, 1, vmspool.StagingBehaviourSuspended, vmstestutil.NewMockFactory(true))
	if got := p.CompletionQueue(); got != nil {
		t.Errorf("Pool.CompletionQueue: got %v, want nil", got)
	}
	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if got := vm.CompletionQueue(); got != nil {
		t.Errorf("VM.CompletionQueue: got %v, want nil", got)
	}
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

// TestCompletionQueueOutcomes covers the routing of events to the outcome they
// name, and what becomes of one that names an outcome the queue was not
// configured with.
func TestCompletionQueueOutcomes(t *testing.T) {
	ctx := quietContext(t)
	q := newQueue(ctx, time.Minute)
	defer closeWithin(t, q, 30*time.Second)

	if got := q.Out("unconfigured"); got != nil {
		t.Errorf("Out of an unconfigured outcome: got %v, want nil", got)
	}

	q.Push(vmspool.CompletionEvent{VM: completed{}, Outcome: outcomeSuccess})
	q.Push(vmspool.CompletionEvent{VM: completed{}, Outcome: outcomeFailure})

	for _, outcome := range []string{outcomeSuccess, outcomeFailure} {
		select {
		case e := <-q.Out(outcome):
			if got, want := e.Outcome, outcome; got != want {
				t.Errorf("outcome: got %q, want %q", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no event delivered for outcome %q", outcome)
		}
	}

	// An event for an unconfigured outcome has nowhere to go and is dropped
	// rather than held, so the VM it names is not reported as parked.
	vm := acquireVM(t, vmstestutil.NewMockFactory(true), 1)
	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: "unconfigured"})
	if got, want := q.NumEvents(), 0; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	if holdsVM(q, vm.ID()) {
		t.Errorf("the queue reports holding %v, which it dropped", vm.ID())
	}
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

// parkVMs acquires every VM in p and pushes each as completed work, waiting
// for the queue to report each one. It returns their IDs, sorted, so that a
// caller can compare them with what the queue reports in no particular order.
func parkVMs(t *testing.T, q *vmspool.CompletionQueue, p *vmspool.Pool, signal <-chan struct{}) []string {
	t.Helper()
	var vmIDs []string
	for range p.Size() {
		vm, err := p.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire: %v", err)
		}
		vmIDs = append(vmIDs, vm.ID())
		q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
		awaitSignal(t, signal, "push")
	}
	slices.Sort(vmIDs)
	return vmIDs
}

// TestCompletionQueueTracking covers the account the queue keeps of the VMs it
// is holding, which is how a pool reports the VMs it created that are parked
// rather than lost.
func TestCompletionQueueTracking(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Minute)
	defer closeWithin(t, q, 30*time.Second)

	signal, release := q.Subscribe(ctx)
	defer release()

	p := newPool(t, 2, vmspool.StagingBehaviourSuspended, factory)
	vmIDs := parkVMs(t, q, p, signal)

	if got, want := q.NumEvents(), 2; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	if got, want := q.NumEventsPerPool(p.Name()), 2; got != want {
		t.Errorf("NumEventsPerPool(%q): got %v, want %v", p.Name(), got, want)
	}
	if got := q.VMIDs(); !slices.Equal(slices.Sorted(slices.Values(got)), vmIDs) {
		t.Errorf("VMIDs: got %v, want %v", got, vmIDs)
	}
	if got := q.VMIDsForPool(p.Name()); !slices.Equal(slices.Sorted(slices.Values(got)), vmIDs) {
		t.Errorf("VMIDsForPool(%q): got %v, want %v", p.Name(), got, vmIDs)
	}
	for _, id := range vmIDs {
		if !holdsVM(q, id) {
			t.Errorf("the queue does not report holding %v", id)
		}
	}
}

// TestCompletionQueueOtherPool covers what the queue reports for a pool that
// has parked nothing, which is what every other pool sharing the queue asks it.
func TestCompletionQueueOtherPool(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Minute)
	defer closeWithin(t, q, 30*time.Second)

	signal, release := q.Subscribe(ctx)
	defer release()

	p := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory)
	parkVMs(t, q, p, signal)

	if got, want := q.NumEventsPerPool("another-pool"), 0; got != want {
		t.Errorf("NumEventsPerPool: got %v, want %v", got, want)
	}
	if got := q.VMIDsForPool("another-pool"); got != nil {
		t.Errorf("VMIDsForPool: got %v, want nil", got)
	}
	if holdsVM(q, "another-pool-vm") {
		t.Error("the queue reports holding a VM it was never given")
	}
}

// TestCompletionQueueRemove covers a caller taking a parked VM back, which is
// what one draining an outcome's channel does: it has become responsible for
// the VM, so the queue must stop reporting and releasing it.
func TestCompletionQueueRemove(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Minute)
	defer closeWithin(t, q, 30*time.Second)

	signal, release := q.Subscribe(ctx)
	defer release()

	p := newPool(t, 2, vmspool.StagingBehaviourSuspended, factory)
	vmIDs := parkVMs(t, q, p, signal)

	q.Remove(p.Name(), vmIDs[0])
	awaitSignal(t, signal, "remove")
	if holdsVM(q, vmIDs[0]) {
		t.Errorf("the queue still reports holding %v after Remove", vmIDs[0])
	}
	if got, want := q.NumEvents(), 1; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	if got, want := q.NumEventsPerPool(p.Name()), 1; got != want {
		t.Errorf("NumEventsPerPool(%q): got %v, want %v", p.Name(), got, want)
	}
	// The VM that was not removed is still held, which only a per-pool list
	// that actually drops the removed entry can report.
	if got, want := q.VMIDsForPool(p.Name()), vmIDs[1:]; !slices.Equal(got, want) {
		t.Errorf("VMIDsForPool(%q): got %v, want %v", p.Name(), got, want)
	}

	// Removing it again changes nothing: it is already gone.
	q.Remove(p.Name(), vmIDs[0])
	if got, want := q.NumEventsPerPool(p.Name()), 1; got != want {
		t.Errorf("NumEventsPerPool(%q): got %v, want %v", p.Name(), got, want)
	}
}

// TestCompletionQueueSameVMTwice covers two pieces of work reported for one
// VM, which must leave the VM counted once: the counts stand for the VMs the
// queue is holding, and there is only one to release.
func TestCompletionQueueSameVMTwice(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Minute)
	defer closeWithin(t, q, 30*time.Second)

	vm := acquireVM(t, factory, 1)
	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeFailure})

	if got, want := q.NumEvents(), 1; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	if got, want := q.NumEventsPerPool(vm.PoolName()), 1; got != want {
		t.Errorf("NumEventsPerPool(%q): got %v, want %v", vm.PoolName(), got, want)
	}
	if got, want := q.VMIDsForPool(vm.PoolName()), []string{vm.ID()}; !slices.Equal(got, want) {
		t.Errorf("VMIDsForPool(%q): got %v, want %v", vm.PoolName(), got, want)
	}
}

// TestCompletionQueueWithoutVM covers work that ran on no VM, and an event
// with no payload at all, both of which leave the queue nothing to release.
func TestCompletionQueueWithoutVM(t *testing.T) {
	ctx := quietContext(t)
	q := newQueue(ctx, time.Minute)

	q.Push(vmspool.CompletionEvent{VM: completed{}, Outcome: outcomeSuccess})
	q.Push(vmspool.CompletionEvent{Outcome: outcomeFailure})

	if got, want := q.NumEvents(), 0; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	// Draining both, including the event with no payload to ask for a logger.
	closeWithin(t, q, 30*time.Second)
}

// TestCompletionQueueExpiry covers the point of the queue: a VM is held for the
// outcome's retention and then deleted, and the queue stops reporting it.
func TestCompletionQueueExpiry(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	// Shorter than the queue's one-second scan interval, so the VM is deleted
	// by the first scan that sees it.
	q := newQueue(ctx, time.Millisecond)
	defer closeWithin(t, q, 30*time.Second)

	signal, release := q.Subscribe(ctx)
	defer release()

	vm := acquireVM(t, factory, 1)
	id := vm.ID()
	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	awaitSignal(t, signal, "push")

	awaitSignal(t, signal, "expiry")
	if holdsVM(q, id) {
		t.Errorf("the queue still reports holding %v after it expired", id)
	}
	if got, want := q.NumEvents(), 0; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}

	// Expiry is what deletes the VM; until then it is there to be inspected.
	deadline := time.Now().Add(5 * time.Second)
	for {
		state := factory.Mocks()[0].State(ctx)
		if state == vms.StateDeleted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the expired VM was not deleted, it is %v", state)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestCompletionQueueClose covers the ordinary close: the queue's context is
// still live, so closing each outcome's input makes it flush and close its
// output, and the VMs it was holding are deleted rather than left behind.
func TestCompletionQueueClose(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Hour)

	vm := acquireVM(t, factory, 1)
	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	if got, want := q.NumEvents(), 1; got != want {
		t.Fatalf("NumEvents: got %v, want %v", got, want)
	}

	closeWithin(t, q, 30*time.Second)

	if holdsVM(q, vm.ID()) {
		t.Errorf("the queue still reports holding %v after it was closed", vm.ID())
	}
	if got, want := factory.Mocks()[0].State(ctx), vms.StateDeleted; got != want {
		t.Errorf("the VM the queue was holding is %v, want %v", got, want)
	}

	// Close is idempotent, since a queue shared by several pools is closed by
	// each of them.
	closeWithin(t, q, 30*time.Second)
}

// TestCompletionQueueCloseAfterCancel is the interrupt case. The queue's
// context is cancelled on ctrl-c, after which its goroutines return without
// closing their output channels, so a close that waited for them to be closed
// would never let the application exit.
func TestCompletionQueueCloseAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(quietContext(t))
	q := newQueue(ctx, time.Hour)
	q.Push(vmspool.CompletionEvent{VM: completed{}, Outcome: outcomeSuccess})

	cancel()
	// Let the queue's goroutines observe the cancellation and exit.
	time.Sleep(50 * time.Millisecond)

	closeWithin(t, q, 30*time.Second)
}

// TestCompletionQueueCloseEmptyAfterCancel covers the same case with nothing
// queued, which is the common one: an idle application interrupted at rest.
func TestCompletionQueueCloseEmptyAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(quietContext(t))
	q := newQueue(ctx, time.Hour)
	cancel()
	time.Sleep(50 * time.Millisecond)

	closeWithin(t, q, 30*time.Second)
}

// TestCompletionQueuePushAfterClose covers a handler that reports a completion
// while the application is shutting down, which races the close of a queue
// shared by several pools. The event is dropped; the pool that created the VM
// deletes it as it closes.
func TestCompletionQueuePushAfterClose(t *testing.T) {
	ctx := quietContext(t)
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Hour)
	vm := acquireVM(t, factory, 1)

	closeWithin(t, q, 30*time.Second)

	q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	if got, want := q.NumEvents(), 0; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

// TestCompletionQueuePushAfterCancel covers the same race against an interrupt
// rather than a close: the queue's goroutines have gone, so nothing would ever
// receive the event.
func TestCompletionQueuePushAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(quietContext(t))
	factory := vmstestutil.NewMockFactory(true)
	q := newQueue(ctx, time.Hour)
	vm := acquireVM(t, factory, 1)

	cancel()
	time.Sleep(50 * time.Millisecond)

	pushed := make(chan struct{})
	go func() {
		defer close(pushed)
		q.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcomeSuccess})
	}()
	select {
	case <-pushed:
	case <-time.After(30 * time.Second):
		t.Fatal("Push did not return")
	}
	if got, want := q.NumEvents(), 0; got != want {
		t.Errorf("NumEvents: got %v, want %v", got, want)
	}

	closeWithin(t, q, 30*time.Second)
	if err := vm.Delete(context.Background()); err != nil {
		t.Errorf("Delete: %v", err)
	}
}
