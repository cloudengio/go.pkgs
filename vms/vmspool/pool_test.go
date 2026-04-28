// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/os/executil"
	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// newPool creates a pool and waits for it be full.
func newPool(t *testing.T, size int, stagingBehaviour vmspool.StagingBehaviour, factory *vmstestutil.MockFactory) *vmspool.Pool {
	t.Helper()
	statusCh := make(chan vmspool.Event, size*4)
	p := vmspool.New(factory,
		vmspool.WithSize(size),
		vmspool.WithStagingBehaviour(stagingBehaviour),
		vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("pool.Start: %v", err)
	}
	waitForEvent(t, statusCh, vmspool.EventStartPoolFull, 5*time.Second)
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

// TestPoolStart verifies that the pool creates the right number of VMs in the correct staging state.
func TestPoolStartAndAquire(t *testing.T) {
	cases := []struct {
		name        string
		suspendable bool
		behaviour   vmspool.StagingBehaviour
		wantState   vms.State
	}{
		{"suspended", true, vmspool.StagingBehaviourSuspended, vms.StateSuspended},
		{"running", true, vmspool.StagingBehaviourRunning, vms.StateRunning},
		{"stopped", true, vmspool.StagingBehaviourStopped, vms.StateStopped},
		{"suspended", false, vmspool.StagingBehaviourSuspended, vms.StateStopped},
		{"running", false, vmspool.StagingBehaviourRunning, vms.StateRunning},
		{"stopped", false, vmspool.StagingBehaviourStopped, vms.StateStopped},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			factory := vmstestutil.NewMockFactory(tc.suspendable)
			pool := newPool(t, 3, tc.behaviour, factory)

			mocks := factory.Mocks()
			if len(mocks) != 3 {
				t.Fatalf("expected 3 mocks after Start, got %d", len(mocks))
			}
			allInState(t, mocks, tc.wantState)

			vm, err := pool.Acquire(context.Background())
			if err != nil {
				t.Fatalf("Acquire: %v", err)
			}

			running := 1
			if tc.behaviour == vmspool.StagingBehaviourRunning {
				running = 3
			}
			if got, want := countInState(mocks, vms.StateRunning), running; got != want {
				t.Errorf("running VMs after Acquire: got %d, want %d", got, want)
			}

			vm.Release(context.Background())
			if got, want := countInState(mocks, vms.StateDeleted), 1; got != want {
				t.Errorf("deleted VMs after Release: got %d, want %d", got, want)
			}

			if got, want := countInState(mocks, tc.wantState), 2; got != want {
				t.Errorf("running VMs after Acquire: got %d, want %d", got, want)
			}
		})
	}
}

// TestPoolExec verifies that Exec is forwarded to the underlying VM and recorded.
func TestPoolExec(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	pool := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory)

	vm, err := pool.Acquire(context.Background())
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

// TestPoolRelease verifies that releasing a VM deletes it and replenishes the pool.
func TestPoolRelease(t *testing.T) {
	for _, behaviour := range []vmspool.StagingBehaviour{
		vmspool.StagingBehaviourSuspended,
		vmspool.StagingBehaviourRunning,
		vmspool.StagingBehaviourStopped,
	} {
		t.Run(behaviour.String(), func(t *testing.T) {
			factory := vmstestutil.NewMockFactory(true)
			pool := newPool(t, 1, behaviour, factory)

			vm, err := pool.Acquire(context.Background())
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

			vm2, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatalf("Acquire after replenishment: %v", err)
			}
			defer vm2.Release(context.Background()) //nolint

			// Factory must have created a second mock for the replenishment.
			if n := len(factory.Mocks()); n != 2 {
				t.Errorf("expected 2 total mocks after replenishment, got %d", n)
			}
		})
	}
}

// TestPoolClose verifies that Close deletes all suspended VMs in the pool.
func TestPoolClose(t *testing.T) {
	const size = 3
	for _, behaviour := range []vmspool.StagingBehaviour{
		vmspool.StagingBehaviourSuspended,
		vmspool.StagingBehaviourRunning,
		vmspool.StagingBehaviourStopped,
	} {
		t.Run(behaviour.String(), func(t *testing.T) {
			statusCh := make(chan vmspool.Event, size*4)
			factory := vmstestutil.NewMockFactory(true)
			p := vmspool.New(factory,
				vmspool.WithSize(size),
				vmspool.WithStagingBehaviour(behaviour),
				vmspool.WithStatus(statusCh))
			if err := p.Start(context.Background()); err != nil {
				t.Fatalf("Start: %v", err)
			}
			waitForEvent(t, statusCh, vmspool.EventStartPoolFull, 5*time.Second)
			if err := p.Close(context.Background()); err != nil {
				t.Fatalf("Close: %v", err)
			}

			allInState(t, factory.Mocks(), vms.StateDeleted)
		})
	}
}

// TestPoolAcquireCancelled verifies that a cancelled context causes Acquire to return.
func TestPoolAcquireCancelled(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	pool := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory)

	// Drain the pool so the next Acquire will block.
	vm, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	// Now try to acquire with a pre-cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = pool.Acquire(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestPoolConcurrency acquires all pool VMs concurrently and releases them,
// verifying that the pool replenishes and remains usable.
func TestPoolConcurrency(t *testing.T) {
	const size = 4
	factory := vmstestutil.NewMockFactory(true)
	pool := newPool(t, size, vmspool.StagingBehaviourSuspended, factory)

	// Acquire all VMs concurrently.
	vmsAcquired := make([]*vmspool.VM, size)
	errs := make([]error, size)
	var wg sync.WaitGroup
	wg.Add(size)
	for i := range size {
		go func(i int) {
			defer wg.Done()
			vmsAcquired[i], errs[i] = pool.Acquire(context.Background())
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
		vm, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("re-Acquire[%d] after replenishment: %v", i, err)
		}
		defer vm.Release(context.Background()) //nolint
	}
}

// TestPoolStartCancelled verifies that cancelling the context passed to Start
// causes Start to return context.Canceled and that the pool can be closed cleanly.
func TestPoolStartCancelled(t *testing.T) {
	const timeout = 5 * time.Second

	cloneBlock := make(chan struct{})
	blockingMock := vmstestutil.NewMock("")
	blockingMock.CloneBlock = cloneBlock

	statusCh := make(chan vmspool.Event, 16)
	factory := vmstestutil.NewMockFactory(true)
	factory.Inject(blockingMock)

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh))

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

// TestPoolStartError verifies that Start retries VM creation on failure and
// eventually succeeds once a good VM can be created.
func TestPoolStartError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		behaviour vmspool.StagingBehaviour
		inject    func(*vmstestutil.Mock)
	}{
		{"suspended", vmspool.StagingBehaviourSuspended, func(m *vmstestutil.Mock) { m.CloneErr = errors.New("clone failed") }},
		{"running", vmspool.StagingBehaviourRunning, func(m *vmstestutil.Mock) { m.StartErr = errors.New("start failed") }},
		{"stopped", vmspool.StagingBehaviourStopped, func(m *vmstestutil.Mock) { m.CloneErr = errors.New("clone failed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			factory := vmstestutil.NewMockFactory(true)

			bad := vmstestutil.NewMock("")
			tc.inject(bad)
			factory.Inject(bad)

			p := vmspool.New(factory,
				vmspool.WithSize(1),
				vmspool.WithStagingBehaviour(tc.behaviour),
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
			// In case of start error, the state is Deleted (since it's cleaned up), while clone error leaves it Initial.
			wantState := vms.StateInitial
			if tc.behaviour == vmspool.StagingBehaviourRunning {
				wantState = vms.StateDeleted
			}
			if got := mocks[0].State(context.Background()); got != wantState {
				t.Errorf("failed mock state = %s, want %s", got, wantState)
			}
			// Second mock was successfully created and deleted by Close.
			if got := mocks[1].State(context.Background()); got != vms.StateDeleted {
				t.Errorf("retry mock state = %s, want Deleted", got)
			}
		})
	}
}

// TestPoolCreateVM_PartialCleanup verifies that a VM is cleaned up when Suspend
// fails after Clone+Start, and that Start retries and eventually succeeds.
func TestPoolCreateVM_PartialCleanup(t *testing.T) {
	ctx := context.Background()
	suspendErr := errors.New("suspend failed")

	// Inject a mock that succeeds Clone+Start but fails Suspend.
	// After Suspend fails the instance is Running; createVM must delete it.
	// Start then retries with a fresh mock and succeeds.
	factory := vmstestutil.NewMockFactory(true)
	bad := vmstestutil.NewMock("")
	bad.SuspendErr = suspendErr
	factory.Inject(bad)

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourSuspended),
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

// TestPoolStatus verifies that the status channel receives the expected events
// for a basic acquire → exec → release cycle.
func TestPoolStatus(t *testing.T) {
	statusCh := make(chan vmspool.Event, 64)

	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vm, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := vm.Exec(context.Background(), io.Discard, io.Discard, "true"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Close the pool before releasing vm2 so that any replenish request made by
	// Release becomes a no-op against the closed pool.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	vm2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("second Acquire (wait for replenishment): %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := vm2.Release(context.Background()); err != nil {
		t.Fatalf("Release after Close: %v", err)
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

	// vm.Release triggers replenishment; vm2.Release (after Close) does not.
	// Both releases emit Release+Released, so counts are 2.
	wantCounts := map[vmspool.EventKind]int{
		vmspool.EventAcquireWaiting:   2,
		vmspool.EventVMDequeued:       2,
		vmspool.EventAcquired:         2,
		vmspool.EventRelease:          2,
		vmspool.EventReleased:         2,
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

// TestPoolReplenishBlockedOnReadySend tests that Close can unblock a
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
func TestPoolReplenishBlockedOnReadySend(t *testing.T) {
	const timeout = 5 * time.Second
	statusCh := make(chan vmspool.Event, 64)

	cloneBlock := make(chan struct{})
	blockingMock := vmstestutil.NewMock("")
	blockingMock.CloneBlock = cloneBlock

	factory := vmstestutil.NewMockFactory(true)
	pool := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Drain events emitted during Start before we begin.
	waitForEvent(t, statusCh, vmspool.EventStartPoolFull, timeout)

	// Drain the pool so p.ready is empty.
	vm, err := pool.Acquire(context.Background())
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
	pool.InjectVM(vmstestutil.NewMock("injected"))

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

// TestPoolCloseUnblocksAcquire verifies that Close can run and unblock a
// goroutine that is blocked inside Acquire waiting for a VM. If Acquire holds
// a lock across the blocking select, Close will deadlock because it needs the
// same lock to signal pool shutdown.
func TestPoolCloseUnblocksAcquire(t *testing.T) {
	const timeout = 5 * time.Second
	statusCh := make(chan vmspool.Event, 32)

	factory := vmstestutil.NewMockFactory(true)
	pool := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
	)
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Acquire and hold the only VM so the pool becomes empty.
	vm, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer vm.Release(context.Background()) //nolint

	// Start a second Acquire that will block because the pool is empty.
	acquireDone := make(chan error, 1)
	go func() {
		_, err := pool.Acquire(context.Background())
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

// TestPoolReleaseCloseRace verifies that Release and Close can run concurrently
// without triggering sync.WaitGroup misuse (Add after Wait) or a data race.
// Run with -race to validate.
func TestPoolReleaseCloseRace(t *testing.T) {
	const iterations = 100
	for range iterations {
		factory := vmstestutil.NewMockFactory(true)
		p := vmspool.New(factory, vmspool.WithSize(1))
		if err := p.Start(context.Background()); err != nil {
			t.Fatalf("Start: %v", err)
		}

		vm, err := p.Acquire(context.Background())
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

// TestPoolCloseBeforePoolFull verifies that calling Close while the async fill
// goroutine (launched by Start for the remaining size-1 VMs) is still running
// cancels those goroutines via p.replenishCtx and completes without deadlocking.
func TestPoolCloseBeforePoolFull(t *testing.T) {
	const timeout = 5 * time.Second

	// Block the second VM's Clone so the async fill goroutine is stuck.
	cloneBlock := make(chan struct{})
	blockingMock := vmstestutil.NewMock("")
	blockingMock.CloneBlock = cloneBlock

	statusCh := make(chan vmspool.Event, 16)
	factory := vmstestutil.NewMockFactory(true)
	// Inject in order: first New() → plain mock (used by synchronous fill),
	// second New() → blocking mock (used by async fill goroutine).
	factory.Inject(vmstestutil.NewMock(""))
	factory.Inject(blockingMock)

	pool := vmspool.New(factory, vmspool.WithSize(2), vmspool.WithStatus(statusCh))
	if err := pool.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait until the first VM has been created (sync) and the async fill goroutine
	// has started on the second VM (blocking in Clone).
	waitForEvent(t, statusCh, vmspool.EventVMCreated, timeout)
	waitForEvent(t, statusCh, vmspool.EventVMCreateStarted, timeout)
	runtime.Gosched() // let the goroutine reach Clone's blocking select

	// Close must cancel p.replenishCtx, which unblocks Clone and lets the async
	// fill goroutine exit. Without this cancellation, Close would hang.
	closeDone := make(chan error, 1)
	go func() { closeDone <- pool.Close(context.Background()) }()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("Close deadlocked: async fill goroutine was not cancelled by replenish context")
	}
}

// trackingCloser is an io.ReadWriteCloser that records whether Close was called.
type trackingCloser struct {
	id     string
	mu     sync.Mutex
	closed bool
}

func (tc *trackingCloser) Write(p []byte) (n int, err error) { return len(p), nil }
func (tc *trackingCloser) Read(p []byte) (n int, err error)  { return 0, io.EOF }
func (tc *trackingCloser) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.closed = true
	return nil
}
func (tc *trackingCloser) isClosed() bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.closed
}

// closerTracker creates and tracks trackingCloser instances via a factory
// function compatible with WithStdoutStderr.
type closerTracker struct {
	mu      sync.Mutex
	closers []*trackingCloser
}

func (ct *closerTracker) factory(id string) (io.ReadWriteCloser, error) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	tc := &trackingCloser{id: id}
	ct.closers = append(ct.closers, tc)
	return tc, nil
}

func (ct *closerTracker) snapshot() []*trackingCloser {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return append([]*trackingCloser(nil), ct.closers...)
}

// TestStdoutStderrClosedOnPoolClose verifies that Close calls Close() on
// the stdout and stderr ReadWriteClosers of every VM remaining in the ready
// queue when the pool is closed without any Acquires.
func TestStdoutStderrClosedOnPoolClose(t *testing.T) {
	const size = 2
	var outTracker, errTracker closerTracker

	statusCh := make(chan vmspool.Event, size*4)
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(size),
		vmspool.WithStatus(statusCh),
		vmspool.WithStdoutStderr(outTracker.factory, errTracker.factory),
	)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForEvent(t, statusCh, vmspool.EventStartPoolFull, 5*time.Second)

	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	outs := outTracker.snapshot()
	if len(outs) != size {
		t.Errorf("expected %d stdout closers, got %d", size, len(outs))
	}
	for i, c := range outs {
		if !c.isClosed() {
			t.Errorf("stdout[%d] (id=%q) not closed after pool.Close", i, c.id)
		}
	}
	errs := errTracker.snapshot()
	if len(errs) != size {
		t.Errorf("expected %d stderr closers, got %d", size, len(errs))
	}
	for i, c := range errs {
		if !c.isClosed() {
			t.Errorf("stderr[%d] (id=%q) not closed after pool.Close", i, c.id)
		}
	}
}

// TestStdoutStderrClosedOnRelease verifies that vm.Release calls Close() on
// the stdout and stderr ReadWriteClosers that were assigned to the VM at
// creation time.
func TestStdoutStderrClosedOnRelease(t *testing.T) {
	var outTracker, errTracker closerTracker

	statusCh := make(chan vmspool.Event, 16)
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStatus(statusCh),
		vmspool.WithStdoutStderr(outTracker.factory, errTracker.factory),
	)
	t.Cleanup(func() { p.Close(context.Background()) }) //nolint

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForEvent(t, statusCh, vmspool.EventStartPoolFull, 5*time.Second)

	vm, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Snapshot the closers that belong to this VM before releasing it.
	outsBefore := outTracker.snapshot()
	errsBefore := errTracker.snapshot()

	if err := vm.Release(context.Background()); err != nil {
		t.Fatalf("Release: %v", err)
	}

	for i, c := range outsBefore {
		if !c.isClosed() {
			t.Errorf("stdout[%d] (id=%q) not closed after vm.Release", i, c.id)
		}
	}
	for i, c := range errsBefore {
		if !c.isClosed() {
			t.Errorf("stderr[%d] (id=%q) not closed after vm.Release", i, c.id)
		}
	}
}

// TestStdoutStderrClosedOnCreateFailure verifies that when createVM fails
// (Start error), the stdout and stderr closers already allocated for that VM
// are closed by the cleanupVM path before retrying.
func TestStdoutStderrClosedOnCreateFailure(t *testing.T) {
	var outTracker, errTracker closerTracker

	startErr := errors.New("start failed")
	factory := vmstestutil.NewMockFactory(true)
	bad := vmstestutil.NewMock("")
	bad.StartErr = startErr
	factory.Inject(bad)

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond),
		vmspool.WithStdoutStderr(outTracker.factory, errTracker.factory),
	)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}

	// Every closer that was ever allocated — including the failed VM's — must
	// have been closed.
	for i, c := range outTracker.snapshot() {
		if !c.isClosed() {
			t.Errorf("stdout[%d] (id=%q) not closed after create failure", i, c.id)
		}
	}
	for i, c := range errTracker.snapshot() {
		if !c.isClosed() {
			t.Errorf("stderr[%d] (id=%q) not closed after create failure", i, c.id)
		}
	}
}

// TestStdoutClosedWhenStderrFactoryFails verifies that when the stderr factory
// returns an error, the stdout ReadWriteCloser that was already created is
// closed immediately so it is not leaked.
func TestStdoutClosedWhenStderrFactoryFails(t *testing.T) {
	var outTracker closerTracker
	var stderrCallCount int32

	stderrFactory := func(id string) (io.ReadWriteCloser, error) {
		n := atomic.AddInt32(&stderrCallCount, 1)
		if n == 1 {
			return nil, errors.New("stderr factory error")
		}
		// Succeed on retries with a plain discard.
		return executil.NewLabelingPipe([]byte("test-"), '\n'), nil
	}

	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond),
		vmspool.WithStdoutStderr(outTracker.factory, stderrFactory),
	)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}

	// The stdout closer for the failed attempt must have been closed.
	for i, c := range outTracker.snapshot() {
		if !c.isClosed() {
			t.Errorf("stdout[%d] (id=%q) not closed after stderr factory failure", i, c.id)
		}
	}
}

// nilConstructor is a vmspool.Constructor that returns nil for its first New
// call and delegates to a real factory for all subsequent calls.
type nilConstructor struct {
	hitNil  bool
	factory *vmstestutil.MockFactory
}

func (c *nilConstructor) New() vms.Instance {
	if !c.hitNil {
		c.hitNil = true
		return nil
	}
	return c.factory.New()
}

// TestPoolNilConstructor verifies that a nil return from the constructor is
// handled gracefully: Start retries and succeeds once the constructor returns
// a real instance.
func TestPoolNilConstructor(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	ctor := &nilConstructor{factory: factory}

	p := vmspool.New(ctor, vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(5*time.Second, time.Millisecond))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start should succeed via retry after nil constructor, got: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
	// nilConstructor consumed one nil call, then the factory produced one real mock.
	mocks := factory.Mocks()
	if len(mocks) != 1 {
		t.Fatalf("expected 1 mock from retry, got %d", len(mocks))
	}
	if got := mocks[0].State(context.Background()); got != vms.StateDeleted {
		t.Errorf("mock state = %s, want Deleted", got)
	}
}

// TestPoolOptionsDefaults verifies that the pool handles invalid and default options gracefully.
func TestPoolOptionsDefaults(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory,
		vmspool.WithSize(-1),
		vmspool.WithCleanupTimeout(-1),
		vmspool.WithCreateTimeoutAndInterval(-1, -1),
		vmspool.WithStopTimeout(-1),
		vmspool.WithStdoutStderr(nil, nil),
	)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Close(context.Background()) //nolint:errcheck

	// DefaultPoolSize is 2, so acquiring 2 VMs should be possible.
	vm1, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire 1: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	vm2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 2: %v", err)
	}

	vm1.Release(context.Background()) //nolint:errcheck
	vm2.Release(context.Background()) //nolint:errcheck
}

// TestPoolStartTwice verifies that calling Start multiple times returns an error.
func TestPoolStartTwice(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(1))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	defer p.Close(context.Background()) //nolint:errcheck

	err := p.Start(context.Background())
	if err == nil || err.Error() != "vmspool: pool already started" {
		t.Errorf("expected 'vmspool: pool already started', got %v", err)
	}
}

// TestStagingBehaviourString verifies the string representation of StagingBehaviour.
func TestStagingBehaviourString(t *testing.T) {
	if got, want := vmspool.StagingBehaviour(99).String(), "Unknown"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestAcquireStartFails verifies that if a VM fails to start during Acquire
// (when using StagingBehaviourStopped), the error is returned and the pool recovers.
func TestAcquireStartFails(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	bad := vmstestutil.NewMock("")
	bad.StartErr = errors.New("start failed")
	factory.Inject(bad)

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourStopped),
	)
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Close(context.Background()) //nolint:errcheck

	_, err := p.Acquire(context.Background())
	if err == nil || !strings.Contains(err.Error(), "start failed") {
		t.Errorf("expected start failure, got %v", err)
	}
}

// TestCreateStdoutFails verifies that if creating stdout for a VM fails,
// the pool logs the error and retries.
func TestCreateStdoutFails(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	stdoutErr := errors.New("stdout creation failed")

	var attempt int32
	outFactory := func(id string) (io.ReadWriteCloser, error) {
		if atomic.AddInt32(&attempt, 1) == 1 {
			return nil, stdoutErr
		}
		return executil.NewLabelingPipe([]byte("test-"), '\n'), nil
	}

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithStdoutStderr(outFactory, nil),
	)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Close(context.Background()) //nolint:errcheck

	if atomic.LoadInt32(&attempt) < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempt)
	}
}

type errorCloser struct {
	io.ReadWriteCloser
	err error
}

func (e errorCloser) Close() error { return e.err }

// TestReleaseAndCloseErrors verifies that if closing a VM's stdout/stderr returns an error,
// it is propagated by Release and Close.
func TestReleaseAndCloseErrors(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	closeErr := errors.New("close error")
	outFactory := func(id string) (io.ReadWriteCloser, error) {
		return errorCloser{
			ReadWriteCloser: executil.NewLabelingPipe([]byte("test-"), '\n'),
			err:             closeErr,
		}, nil
	}

	p := vmspool.New(factory,
		vmspool.WithSize(2),
		vmspool.WithStdoutStderr(outFactory, outFactory),
	)

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	vm, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	if err := vm.Release(context.Background()); err == nil || !strings.Contains(err.Error(), "close error") {
		t.Errorf("Release expected to contain close error, got %v", err)
	}

	if err := p.Close(context.Background()); err == nil || !strings.Contains(err.Error(), "close error") {
		t.Errorf("Close expected to contain close error, got %v", err)
	}
}

// TestAcquireOnClosedPoolEvent verifies the EventAttemptToUseClosedPool event is emitted
// when acquiring from a closed pool.
func TestAcquireOnClosedPoolEvent(t *testing.T) {
	statusCh := make(chan vmspool.Event, 32)
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Give some time for async events to be sent to channel
	time.Sleep(100 * time.Millisecond)
	for len(statusCh) > 0 {
		<-statusCh
	}

	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := p.Acquire(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var found bool
	for len(statusCh) > 0 {
		e := <-statusCh
		if e.Kind == vmspool.EventAttemptToUseClosedPool {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected EventAttemptToUseClosedPool")
	}
}

// TestAttemptCreateVMTimeout verifies that if VM creation times out, it is handled and retried.
func TestAttemptCreateVMTimeout(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)

	blockingMock := vmstestutil.NewMock("")
	cloneBlock := make(chan struct{})
	blockingMock.CloneBlock = cloneBlock
	factory.Inject(blockingMock)
	factory.Inject(vmstestutil.NewMock("")) // Second mock succeeds

	p := vmspool.New(factory,
		vmspool.WithSize(1),
		vmspool.WithCreateTimeoutAndInterval(50*time.Millisecond, time.Millisecond),
	)

	go func() {
		// Wait for the timeout to elapse and trigger retry, then unblock the first mock
		time.Sleep(100 * time.Millisecond)
		close(cloneBlock)
	}()

	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
}
