// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

func newPool(t *testing.T, size int, factory *vmstestutil.MockFactory) *vmspool.Pool {
	t.Helper()
	p := vmspool.New(factory, vmspool.WithSize(size))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("pool.Start: %v", err)
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
	factory := vmstestutil.NewMockFactory("start-test")
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
	factory := vmstestutil.NewMockFactory("acquire-test")
	pool := newPool(t, 2, factory)

	vm, err := pool.Acquire(context.Background())
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
	factory := vmstestutil.NewMockFactory("exec-test")
	pool := newPool(t, 1, factory)

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

// TestPool_Release verifies that releasing a VM deletes it and replenishes the pool.
func TestPool_Release(t *testing.T) {
	factory := vmstestutil.NewMockFactory("release-test")
	pool := newPool(t, 1, factory)

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
}

// TestPool_Close verifies that Close deletes all suspended VMs in the pool.
func TestPool_Close(t *testing.T) {
	factory := vmstestutil.NewMockFactory("close-test")
	p := vmspool.New(factory, vmspool.WithSize(3))
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	allInState(t, factory.Mocks(), vms.StateDeleted)
}

// TestPool_AcquireCancelled verifies that a cancelled context causes Acquire to return.
func TestPool_AcquireCancelled(t *testing.T) {
	factory := vmstestutil.NewMockFactory("cancelled-test")
	pool := newPool(t, 1, factory)

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

// TestPool_Concurrency acquires all pool VMs concurrently and releases them,
// verifying that the pool replenishes and remains usable.
func TestPool_Concurrency(t *testing.T) {
	const size = 4
	factory := vmstestutil.NewMockFactory("concurrency-test")
	pool := newPool(t, size, factory)

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

// TestPool_StartError verifies that Start returns an error when VM creation fails.
func TestPool_StartError(t *testing.T) {
	cloneErr := errors.New("clone failed")
	factory := vmstestutil.NewMockFactory("start-error-test")

	bad := vmstestutil.NewMock()
	bad.CloneErr = cloneErr
	factory.Inject(bad)

	p := vmspool.New(factory, vmspool.WithSize(1))
	err := p.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to fail")
	}
	if !errors.Is(err, cloneErr) {
		t.Errorf("expected clone error, got %v", err)
	}
}

// TestPool_CreateVM_PartialCleanup verifies that a VM is cleaned up when Start
// or Suspend fails after Clone, preventing resource leaks.
func TestPool_CreateVM_PartialCleanup(t *testing.T) {
	ctx := context.Background()
	suspendErr := errors.New("suspend failed")

	// Inject a mock that succeeds Clone+Start but fails Suspend.
	// After Suspend fails the instance is Running; createVM must delete it.
	factory := vmstestutil.NewMockFactory("partial-cleanup-test")
	bad := vmstestutil.NewMock()
	bad.SuspendErr = suspendErr
	factory.Inject(bad)

	p := vmspool.New(factory, vmspool.WithSize(1))
	err := p.Start(ctx)
	if err == nil {
		t.Fatal("expected Start to fail")
	}
	if !errors.Is(err, suspendErr) {
		t.Errorf("expected suspend error, got %v", err)
	}

	// The VM that failed Suspend was Running; createVM must have cleaned it up.
	mocks := factory.Mocks()
	if len(mocks) != 1 {
		t.Fatalf("expected 1 mock, got %d", len(mocks))
	}
	if got := mocks[0].State(ctx); got != vms.StateDeleted {
		t.Errorf("partially-created VM state = %s, want Deleted", got)
	}
}

// TestPool_Status verifies that the status channel receives the expected events
// for a basic acquire → exec → release cycle.
func TestPool_Status(t *testing.T) {
	statusCh := make(chan vmspool.Event, 64)

	factory := vmstestutil.NewMockFactory("status-test")
	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
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

	// Block until the replenishment goroutine completes by acquiring the new VM.
	// We do not release vm2: releasing it would launch a second replenishment
	// goroutine that could race with p.Close cancelling the background context.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	vm2, err := p.Acquire(ctx)
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

// TestPool_ReleaseCloseRace verifies that Release and Close can run concurrently
// without triggering sync.WaitGroup misuse (Add after Wait) or a data race.
// Run with -race to validate.
func TestPool_ReleaseCloseRace(t *testing.T) {
	const iterations = 100
	for range iterations {
		factory := vmstestutil.NewMockFactory("release-close-race")
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
