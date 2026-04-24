// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

func newPool(t *testing.T, size int, factory *vmstestutil.MockFactory) *vmspool.Pool {
	t.Helper()
	statusCh := make(chan vmspool.Event, size*4)
	p := vmspool.New(factory, vmspool.WithSize(size), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("pool.Start: %v", err)
	}
	// Start returns once the first VM is ready; wait for the full pool to fill.
	for range size {
		waitForEvent(t, statusCh, vmspool.EventVMCreated, 5*time.Second)
	}
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
	})
	return p
}

func allInState(t *testing.T, mocks []*vmstestutil.Mock, want vms.State) {
	t.Helper()
	ctx := context.Background()
	for i, m := range mocks {
		if got := m.State(ctx); got != want {
			t.Errorf("mock[%d]: state = %s, want %s", i, got, want)
		}
	}
}

func countInState(mocks []*vmstestutil.Mock, want vms.State) int {
	ctx := context.Background()
	n := 0
	for _, m := range mocks {
		if m.State(ctx) == want {
			n++
		}
	}
	return n
}

// TestPool_Start verifies that the pool creates the right number of suspended VMs.
func TestPool_Start(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	newPool(t, 3, factory)

	mocks := factory.Mocks()
	if len(mocks) != 3 {
		t.Fatalf("expected 3 mocks after Start, got %d", len(mocks))
	}
	allInState(t, mocks, vms.StateSuspended)
}

// TestPool_Acquire verifies that acquiring a VM starts it (Running state)
// and leaves remaining pool VMs suspended.
func TestPool_Acquire(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newPool(t, 2, factory)

	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	mocks := factory.Mocks()
	if n := countInState(mocks, vms.StateRunning); n != 1 {
		t.Errorf("running VMs after Acquire: got %d, want 1", n)
	}
	if n := countInState(mocks, vms.StateSuspended); n != 1 {
		t.Errorf("suspended VMs after Acquire: got %d, want 1", n)
	}
}

// TestPool_Exec verifies that Exec is forwarded to the underlying VM and recorded.
func TestPool_Exec(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newPool(t, 1, factory)

	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	if err := vm.Exec(context.Background(), io.Discard, io.Discard, "echo", "hello"); err != nil {
		t.Fatalf("Exec: %v", err)
	}

	// Find the running mock and verify it recorded the call.
	for _, m := range factory.Mocks() {
		calls := m.ExecCalls()
		if len(calls) == 0 {
			continue
		}
		if calls[0].Cmd != "echo" || len(calls[0].Args) != 1 || calls[0].Args[0] != "hello" {
			t.Errorf("unexpected ExecCall: %+v", calls[0])
		}
		return
	}
	t.Error("no mock recorded an Exec call")
}

// TestPool_Release verifies that releasing a VM deletes it and replenishes the pool.
func TestPool_Release(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newPool(t, 1, factory)

	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// The original VM should now be deleted.
	firstMocks := factory.Mocks()
	if got := firstMocks[0].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("released VM state = %s, want Deleted", got)
	}

	// Acquire again: this blocks until the replenishment goroutine delivers a
	// new suspended VM and the pool starts it.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vm2, err := pool.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire after replenishment: %v", err)
	}
	defer vm2.Release(context.Background()) //nolint

	// Factory must have created a second mock for the replenishment.
	if n := len(factory.Mocks()); n != 2 {
		t.Errorf("expected 2 total mocks after replenishment, got %d", n)
	}
}

// TestPool_Close verifies that Close deletes all suspended VMs in the pool.
func TestPool_Close(t *testing.T) {
	const size = 3
	statusCh := make(chan vmspool.Event, size*4)
	factory := vmstestutil.NewMockFactory()
	p := vmspool.New(factory, vmspool.WithSize(size), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range size {
		waitForEvent(t, statusCh, vmspool.EventVMCreated, 5*time.Second)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	allInState(t, factory.Mocks(), vms.StateDeleted)
}

// TestPool_AcquireCancelled verifies that a cancelled context causes Acquire to return.
func TestPool_AcquireCancelled(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newPool(t, 1, factory)

	// Drain the pool so the next Acquire will block.
	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	// Now try to acquire with a pre-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = pool.Acquire(ctx, io.Discard, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestPool_Concurrency acquires all pool VMs concurrently and releases them,
// verifying that the pool replenishes and remains usable.
func TestPool_Concurrency(t *testing.T) {
	const size = 4
	factory := vmstestutil.NewMockFactory()
	pool := newPool(t, size, factory)

	// Acquire all VMs concurrently.
	vmsAcquired := make([]*vmspool.VM, size)
	errs := make([]error, size)
	var wg sync.WaitGroup
	wg.Add(size)
	for i := range size {
		go func(i int) {
			defer wg.Done()
			vmsAcquired[i], errs[i] = pool.Acquire(context.Background(), io.Discard, io.Discard)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Acquire[%d]: %v", i, err)
		}
	}

	// Release all VMs.
	for _, vm := range vmsAcquired {
		if vm != nil {
			if err := vm.Release(context.Background()); err != nil {
				t.Errorf("Release: %v", err)
			}
		}
	}

	// Pool should replenish back to full; acquire all again to confirm.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := range size {
		vm, err := pool.Acquire(ctx, io.Discard, io.Discard)
		if err != nil {
			t.Fatalf("re-Acquire[%d] after replenishment: %v", i, err)
		}
		defer vm.Release(context.Background()) //nolint
	}
}

// TestPool_StartCancelled verifies that cancelling the context passed to Start
// causes Start to return context.Canceled and that the pool can be closed cleanly.
func TestPool_StartCancelled(t *testing.T) {
	const timeout = 5 * time.Second

	cloneBlock := make(chan struct{})
	blockingMock := vmstestutil.NewMock()
	blockingMock.CloneBlock = cloneBlock

	statusCh := make(chan vmspool.Event, 16)
	factory := vmstestutil.NewMockFactory()
	factory.Inject(blockingMock)

	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startDone := make(chan error, 1)
	go func() { startDone <- p.Start(ctx) }()

	// Wait until createVMAndNotify has fired EventVMCreateStarted, confirming
	// the goroutine is about to call Clone (which will block on cloneBlock).
	waitForEvent(t, statusCh, vmspool.EventVMCreateStarted, timeout)
	runtime.Gosched() // let the goroutine reach Clone's blocking select

	cancel()

	select {
	case err := <-startDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Start: got %v, want context.Canceled", err)
		}
	case <-time.After(timeout):
		t.Fatal("Start did not return after context cancellation")
	}

	// Pool was never fully started; Close must still complete cleanly.
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close after cancelled Start: %v", err)
	}
}

// TestPool_StartError verifies that Start retries VM creation on failure and
// eventually succeeds once a good VM can be created.
func TestPool_StartError(t *testing.T) {
	cloneErr := errors.New("clone failed")
	factory := vmstestutil.NewMockFactory()

	bad := vmstestutil.NewMock()
	bad.CloneErr = cloneErr
	factory.Inject(bad)

	p := vmspool.New(factory, vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start should succeed via retry, got: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
	mocks := factory.Mocks()
	if len(mocks) != 2 {
		t.Fatalf("expected 2 mocks (1 failed + 1 retry), got %d", len(mocks))
	}
	// Clone failure leaves the mock in StateInitial (no state transition occurs).
	if got := mocks[0].State(context.Background()); got != vms.StateInitial {
		t.Errorf("failed mock state = %s, want Initial", got)
	}
	// Second mock was successfully created and deleted by Close.
	if got := mocks[1].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("retry mock state = %s, want Deleted", got)
	}
}

// TestPool_CreateVM_PartialCleanup verifies that a VM is cleaned up when Suspend
// fails after Clone+Start, and that Start retries and eventually succeeds.
func TestPool_CreateVM_PartialCleanup(t *testing.T) {
	ctx := context.Background()
	suspendErr := errors.New("suspend failed")

	// Inject a mock that succeeds Clone+Start but fails Suspend.
	// After Suspend fails the instance is Running; createVM must delete it.
	// Start then retries with a fresh mock and succeeds.
	factory := vmstestutil.NewMockFactory()
	bad := vmstestutil.NewMock()
	bad.SuspendErr = suspendErr
	factory.Inject(bad)

	p := vmspool.New(factory, vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond))
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start should succeed via retry, got: %v", err)
	}
	if err := p.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}

	mocks := factory.Mocks()
	if len(mocks) != 2 {
		t.Fatalf("expected 2 mocks (1 failed + 1 retry), got %d", len(mocks))
	}
	// The VM that failed Suspend was Running; createVM must have cleaned it up.
	if got := mocks[0].State(ctx); got != vms.StateDeleted {
		t.Errorf("partially-created VM state = %s, want Deleted", got)
	}
	// The retry VM was successfully created and deleted by Close.
	if got := mocks[1].State(ctx); got != vms.StateDeleted {
		t.Errorf("retry VM state = %s, want Deleted", got)
	}
}

// TestPool_Status verifies that the status channel receives the expected events
// for a basic acquire → exec → release cycle.
func TestPool_Status(t *testing.T) {
	statusCh := make(chan vmspool.Event, 64)

	factory := vmstestutil.NewMockFactory()
	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vm, err := p.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := vm.Exec(context.Background(), io.Discard, io.Discard, "true"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Block until the replenishment goroutine completes by acquiring the new VM.
	// We do not release vm2: releasing it would launch a second replenishment
	// goroutine that could race with p.Close cancelling the background context.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	vm2, err := p.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("second Acquire (wait for replenishment): %v", err)
	}
	_ = vm2 // held intentionally; p.Close cleans up the pool without affecting it.

	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var events []vmspool.EventKind
	for {
		select {
		case e := <-statusCh:
			events = append(events, e.Kind)
		default:
			goto done
		}
	}
done:

	// One acquire→release cycle emits each of these exactly once;
	// the second Acquire (to confirm replenishment) contributes Waiting/Dequeued/Acquired.
	wantCounts := map[vmspool.EventKind]int{
		vmspool.EventAcquireWaiting:   2,
		vmspool.EventVMDequeued:       2,
		vmspool.EventAcquired:         2,
		vmspool.EventRelease:          1,
		vmspool.EventReleased:         1,
		vmspool.EventReplenishStarted: 1,
		vmspool.EventReplenished:      1,
	}
	counts := make(map[vmspool.EventKind]int)
	for _, e := range events {
		counts[e]++
	}
	for kind, want := range wantCounts {
		if counts[kind] != want {
			t.Errorf("event %s: got %d, want %d  (full sequence: %v)", kind, counts[kind], want, events)
		}
	}

	// Within the same goroutine, Waiting → Dequeued → Acquired is guaranteed.
	// Verify the two Waiting events each precede their paired Dequeued.
	assertPrecedes(t, events, vmspool.EventAcquireWaiting, vmspool.EventVMDequeued)
	assertPrecedes(t, events, vmspool.EventVMDequeued, vmspool.EventAcquired)
	// Release → ReplenishStarted → Released happen in the same goroutine.
	assertPrecedes(t, events, vmspool.EventRelease, vmspool.EventReplenishStarted)
	assertPrecedes(t, events, vmspool.EventReplenishStarted, vmspool.EventReleased)
	// ReplenishStarted always precedes Replenished.
	assertPrecedes(t, events, vmspool.EventReplenishStarted, vmspool.EventReplenished)
}

// assertPrecedes checks that at least one occurrence of before appears earlier
// in events than at least one occurrence of after.
func assertPrecedes(t *testing.T, events []vmspool.EventKind, before, after vmspool.EventKind) {
	t.Helper()
	for i, e := range events {
		if e != before {
			continue
		}
		for _, e2 := range events[i+1:] {
			if e2 == after {
				return
			}
		}
	}
	t.Errorf("no occurrence of %s found before %s in %v", before, after, events)
}

// waitForEvent drains statusCh until it receives an event of the given kind,
// then returns. It fails the test if the event is not seen within timeout.
func waitForEvent(t *testing.T, statusCh <-chan vmspool.Event, kind vmspool.EventKind, timeout time.Duration) {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case e := <-statusCh:
			if e.Kind == kind {
				return
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for event %s", kind)
		}
	}
}

// TestPool_ReplenishBlockedOnReadySend tests that Close can unblock a
// replenishment goroutine that is stuck trying to send to p.ready when the
// channel is already at capacity. Without a select/ctx.Done() guard on that
// send, the goroutine blocks indefinitely, preventing p.wg.Wait from returning
// and deadlocking Close.
//
// Setup:
//  1. Acquire the only pool VM so p.ready becomes empty.
//  2. Inject a blocking mock (Clone waits for a signal) then Release, which
//     triggers a replenishment goroutine that pauses inside Clone.
//  3. While the goroutine is paused, fill p.ready to capacity via ReadyCh.
//  4. Unblock Clone — the goroutine proceeds through Start+Suspend and
//     arrives at p.ready <- inst, which now blocks (channel is full).
//  5. Call Close. With the select/ctx.Done() fix, Close cancels the context,
//     the goroutine exits, and Close completes. Without the fix, Close hangs.
func TestPool_ReplenishBlockedOnReadySend(t *testing.T) {
	const timeout = 5 * time.Second
	statusCh := make(chan vmspool.Event, 64)

	cloneBlock := make(chan struct{})
	blockingMock := vmstestutil.NewMock()
	blockingMock.CloneBlock = cloneBlock

	factory := vmstestutil.NewMockFactory()
	pool := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Drain the EventVMCreated emitted during Start before we begin.
	waitForEvent(t, statusCh, vmspool.EventVMCreated, timeout)

	// Drain the pool so p.ready is empty.
	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Queue the blocking mock for the next constructor call, then Release to
	// trigger replenishment. The goroutine will block inside Clone immediately.
	factory.Inject(blockingMock)
	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Wait until the replenishment goroutine has started running.
	waitForEvent(t, statusCh, vmspool.EventReplenishStarted, timeout)

	// Fill p.ready to capacity while the goroutine is paused in Clone.
	// When Clone unblocks, the goroutine will try to send its newly-created
	// VM and find the channel full.
	pool.ReadyCh() <- vmstestutil.NewMock()

	// Unblock Clone. The goroutine races through Start+Suspend (both
	// instantaneous in the mock) towards p.ready <- inst.
	close(cloneBlock)

	// EventVMCreated fires inside createVMAndNotifiy, one statement before the
	// select that contains p.ready <- inst. Waiting for it means the goroutine
	// is about to attempt the send. runtime.Gosched() then yields the scheduler
	// so the goroutine can take that final step and block on the full channel
	// before Close cancels the context.
	waitForEvent(t, statusCh, vmspool.EventVMCreated, timeout)
	runtime.Gosched()

	// Close must cancel the goroutine's context and complete within timeout.
	// Without the select/ctx.Done() guard on p.ready <- inst, the goroutine
	// stays blocked and Close deadlocks.
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- pool.Close(context.Background())
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("Close deadlocked: replenishment goroutine is blocked on p.ready <- inst")
	}
}

// TestPool_CloseUnblocksAcquire verifies that Close can run and unblock a
// goroutine that is blocked inside Acquire waiting for a VM. If Acquire holds
// a lock across the blocking select, Close will deadlock because it needs the
// same lock to signal pool shutdown.
func TestPool_CloseUnblocksAcquire(t *testing.T) {
	const timeout = 5 * time.Second
	statusCh := make(chan vmspool.Event, 32)

	factory := vmstestutil.NewMockFactory()
	pool := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
	)
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Acquire and hold the only VM so the pool becomes empty.
	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	// Start a second Acquire that will block because the pool is empty.
	acquireDone := make(chan error, 1)
	go func() {
		_, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
		acquireDone <- err
	}()

	// Wait until the blocked Acquire has sent EventAcquireWaiting (the first
	// event was from the initial Acquire above; the second confirms the
	// goroutine is now blocking on the ready channel).
	seen := 0
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for seen < 2 {
		select {
		case e := <-statusCh:
			if e.Kind == vmspool.EventAcquireWaiting {
				seen++
			}
		case <-deadline.C:
			t.Fatal("timed out waiting for second Acquire to start blocking")
		}
	}

	// Close must succeed. If Acquire holds a lock while blocking on the ready
	// channel, Close will deadlock trying to acquire that same lock.
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- pool.Close(context.Background())
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("Close deadlocked: Acquire is holding a lock while blocking on the ready channel")
	}

	// The blocked Acquire must have been unblocked by Close and returned an error.
	select {
	case err := <-acquireDone:
		if err == nil {
			t.Error("expected Acquire to return an error after Close, got nil")
		}
	case <-time.After(time.Second):
		t.Error("Acquire did not return after Close completed")
	}
}

// TestPool_ReleaseCloseRace verifies that Release and Close can run concurrently
// without triggering sync.WaitGroup misuse (Add after Wait) or a data race.
// Run with -race to validate.
func TestPool_ReleaseCloseRace(t *testing.T) {
	const iterations = 100
	for range iterations {
		factory := vmstestutil.NewMockFactory()
		p := vmspool.New(factory, vmspool.WithSize(1))
		if err := p.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}

		vm, err := p.Acquire(context.Background(), io.Discard, io.Discard)
		if err != nil {
			t.Fatalf("Acquire: %v", err)
		}

		// Release and Close race: exactly the scenario that caused wg misuse.
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = vm.Release(context.Background())
		}()
		go func() {
			defer wg.Done()
			_ = p.Close(context.Background())
		}()
		wg.Wait()
	}
}

// --- WithSuspendVMs(false) tests ---
//
// When suspendVMs is false the pool keeps VMs in StateRunning rather than
// StateSuspended. Acquire calls Start on an already-running instance (a no-op
// for Mock), and Release/Close clean up running VMs.

func newRunningPool(t *testing.T, size int, factory *vmstestutil.MockFactory) *vmspool.Pool {
	t.Helper()
	statusCh := make(chan vmspool.Event, size*4)
	p := vmspool.New(factory, vmspool.WithSize(size), vmspool.WithSuspendVMs(false), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("pool.Start: %v", err)
	}
	for range size {
		waitForEvent(t, statusCh, vmspool.EventVMCreated, 5*time.Second)
	}
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Errorf("pool.Close: %v", err)
		}
	})
	return p
}

// TestPool_NoSuspend_Start verifies that Start places VMs in StateRunning
// (not StateSuspended) when suspend is disabled.
func TestPool_NoSuspend_Start(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	newRunningPool(t, 3, factory)

	mocks := factory.Mocks()
	if len(mocks) != 3 {
		t.Fatalf("expected 3 mocks after Start, got %d", len(mocks))
	}
	allInState(t, mocks, vms.StateRunning)
}

// TestPool_NoSuspend_Acquire verifies that Acquire hands out a running VM.
// All VMs remain in StateRunning because Start is idempotent on an already-
// running instance.
func TestPool_NoSuspend_Acquire(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newRunningPool(t, 2, factory)

	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	// Both the acquired VM and the one still in the pool are Running.
	if n := countInState(factory.Mocks(), vms.StateRunning); n != 2 {
		t.Errorf("running VMs after Acquire: got %d, want 2", n)
	}
}

// TestPool_NoSuspend_Release verifies that releasing a VM deletes it and
// replenishes the pool with a fresh running instance.
func TestPool_NoSuspend_Release(t *testing.T) {
	factory := vmstestutil.NewMockFactory()
	pool := newRunningPool(t, 1, factory)

	vm, err := pool.Acquire(context.Background(), io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Original VM must be deleted.
	if got := factory.Mocks()[0].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("released VM state = %s, want Deleted", got)
	}

	// Acquire again: blocks until the replenishment goroutine provides a new
	// running VM.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	vm2, err := pool.Acquire(ctx, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("Acquire after replenishment: %v", err)
	}
	defer vm2.Release(context.Background()) //nolint

	// Replenishment must have created a second mock.
	if n := len(factory.Mocks()); n != 2 {
		t.Errorf("expected 2 total mocks after replenishment, got %d", n)
	}
	// The new VM is running.
	if got := factory.Mocks()[1].State(context.Background()); got != vms.StateRunning {
		t.Errorf("replenished VM state = %s, want Running", got)
	}
}

// TestPool_NoSuspend_Close verifies that Close deletes all running VMs in the pool.
func TestPool_NoSuspend_Close(t *testing.T) {
	const size = 3
	statusCh := make(chan vmspool.Event, size*4)
	factory := vmstestutil.NewMockFactory()
	p := vmspool.New(factory, vmspool.WithSize(size), vmspool.WithSuspendVMs(false), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range size {
		waitForEvent(t, statusCh, vmspool.EventVMCreated, 5*time.Second)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	allInState(t, factory.Mocks(), vms.StateDeleted)
}

// TestPool_NoSuspend_StartError verifies that Start retries VM creation on failure
// (Start error with suspend disabled) and eventually succeeds.
func TestPool_NoSuspend_StartError(t *testing.T) {
	startErr := errors.New("start failed")
	factory := vmstestutil.NewMockFactory()

	bad := vmstestutil.NewMock()
	bad.StartErr = startErr
	factory.Inject(bad)

	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithSuspendVMs(false),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start should succeed via retry, got: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
	mocks := factory.Mocks()
	if len(mocks) != 2 {
		t.Fatalf("expected 2 mocks (1 failed + 1 retry), got %d", len(mocks))
	}
	// The Start-failed VM was cleaned up by createVM.
	if got := mocks[0].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("failed mock state = %s, want Deleted", got)
	}
	// The retry VM was successfully created and deleted by Close.
	if got := mocks[1].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("retry mock state = %s, want Deleted", got)
	}
}
