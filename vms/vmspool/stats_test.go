// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// waitForStats polls until the pool's stats equal want. The pool creates and
// replenishes VMs asynchronously, so a snapshot taken immediately after an
// operation may still show the previous state.
func waitForStats(t *testing.T, p *vmspool.Pool, want vmspool.Stats) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := p.Stats()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for stats %+v, last saw %+v", want, got)
		}
		time.Sleep(time.Millisecond)
	}
}

func requireStats(t *testing.T, p *vmspool.Pool, want vmspool.Stats) {
	t.Helper()
	if got := p.Stats(); got != want {
		t.Fatalf("stats: got %+v, want %+v", got, want)
	}
}

// statsViolation reports how s breaks the invariants that hold of every
// snapshot of a pool of the given size, whatever it is doing and however its
// operations have failed, or returns "" if it breaks none of them.
func statsViolation(s vmspool.Stats, size int) string {
	switch {
	case s.Size != size:
		return "Size is not the configured size"
	case s.Available < 0 || s.Acquired < 0 || s.Pending < 0:
		return "a count is negative"
	case s.Available > s.Size:
		return "more VMs are available than the pool can hold"
	}
	return ""
}

// requireSettled waits for a pool of the given size to have nothing in flight
// and every slot filled: all VMs available, none acquired or being created.
func requireSettled(t *testing.T, p *vmspool.Pool, size int) {
	t.Helper()
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size})
}

// requireEmpty verifies that a closed pool of the given size holds no VMs at
// all: every one it created, in whatever state, has been deleted.
func requireEmpty(t *testing.T, p *vmspool.Pool, size int) {
	t.Helper()
	requireStats(t, p, vmspool.Stats{Size: size})
}

// TestPoolStatsLifecycle follows the counts through the steps of a VM's life
// in the pool: creation, acquisition, and deletion with its replenishment.
func TestPoolStatsLifecycle(t *testing.T) {
	ctx := context.Background()
	const size = 3
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(size))

	// Nothing exists until Start, but the size is known.
	requireStats(t, p, vmspool.Stats{Size: size})

	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	// Start returns once one VM is ready; the rest are created behind it.
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size})

	vm1, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: size, Available: size - 1, Acquired: 1})

	vm2, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: size, Available: size - 2, Acquired: 2})

	// Deleting a VM gives up its slot, which is refilled, so that the pool
	// returns to its size once the replacement has been created.
	if err := vm1.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size - 1, Acquired: 1})

	if err := vm2.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size})
}

// TestPoolStatsPending verifies that a VM which is still being created is
// counted as pending, rather than as available or acquired.
func TestPoolStatsPending(t *testing.T) {
	ctx := context.Background()
	const timeout = 5 * time.Second
	statusCh := make(chan vmspool.Event, 64)

	cloneBlock := make(chan struct{})
	blocked := vmstestutil.NewMock("")
	blocked.CloneBlock = cloneBlock

	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(1), vmspool.WithStatus(statusCh))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	waitForEvent(t, statusCh, vmspool.EventStartPoolFull, timeout)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: 1, Acquired: 1})

	// The replacement's creation is held up in Clone.
	factory.Inject(blocked)
	if err := vm.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	waitForStats(t, p, vmspool.Stats{Size: 1, Pending: 1})

	// It stays pending for as long as creation does, and is not counted twice.
	time.Sleep(20 * time.Millisecond)
	requireStats(t, p, vmspool.Stats{Size: 1, Pending: 1})

	close(cloneBlock)
	waitForStats(t, p, vmspool.Stats{Size: 1, Available: 1})
}

// TestPoolStatsStopAndRelease verifies the accounting for a VM stopped by
// StopAndRelease: the caller still holds it, so it stays acquired, but its slot
// has been given up, so a replacement is created and the pool momentarily holds
// more VMs than its size.
func TestPoolStatsStopAndRelease(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(1))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	waitForStats(t, p, vmspool.Stats{Size: 1, Available: 1})

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, stopErr := vm.StopAndRelease(ctx, time.Second); stopErr != nil {
		t.Fatal(stopErr)
	}
	waitForStats(t, p, vmspool.Stats{Size: 1, Available: 1, Acquired: 1})

	// Deleting the stopped VM does not request a second replacement.
	if err := vm.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: 1, Available: 1})
	time.Sleep(20 * time.Millisecond)
	requireStats(t, p, vmspool.Stats{Size: 1, Available: 1})
}

// TestPoolStatsClose verifies that Close leaves nothing counted, including a
// VM that a caller acquired and never deleted, which Close deletes.
func TestPoolStatsClose(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(2))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	waitForStats(t, p, vmspool.Stats{Size: 2, Available: 2})
	if _, err := p.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: 2, Available: 1, Acquired: 1})

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: 2})
	// Idempotent, like Close.
	requireStats(t, p, vmspool.Stats{Size: 2})
}

// TestPoolStatsDoesNotBlock verifies that Stats can be read while an Acquire is
// in the middle of starting a VM, which holds the lock that serialises Acquire
// and Close. A Stats that took that lock would wait for the VM to start.
func TestPoolStatsDoesNotBlock(t *testing.T) {
	ctx := context.Background()
	const timeout = 5 * time.Second

	startBlock := make(chan struct{})
	blocked := vmstestutil.NewMock("")
	blocked.StartBlock = startBlock

	factory := vmstestutil.NewMockFactory(true)
	factory.Inject(blocked)
	// A staged-stopped VM must be started by Acquire, which is what blocks.
	p := vmspool.New(factory, vmspool.WithSize(1),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourStopped))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	// Cleanups run last-in first-out: release the blocked Start before Close,
	// which would otherwise wait for it on a test that failed part way through.
	var release sync.Once
	releaseStart := func() { release.Do(func() { close(startBlock) }) }
	t.Cleanup(releaseStart)
	waitForStats(t, p, vmspool.Stats{Size: 1, Available: 1})

	acquired := make(chan error, 1)
	go func() {
		_, err := p.Acquire(ctx)
		acquired <- err
	}()
	// statsWithin reads Stats on its own goroutine, so that a Stats that blocks
	// is reported as such rather than hanging the test.
	statsWithin := func() vmspool.Stats {
		done := make(chan vmspool.Stats, 1)
		go func() { done <- p.Stats() }()
		select {
		case got := <-done:
			return got
		case <-time.After(timeout):
			t.Fatal("Stats blocked while Acquire was starting a VM")
			return vmspool.Stats{}
		}
	}

	// Wait until Acquire has taken the VM from the pool and is starting it.
	want := vmspool.Stats{Size: 1, Pending: 1}
	deadline := time.Now().Add(timeout)
	for got := statsWithin(); got != want; got = statsWithin() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for stats %+v, last saw %+v", want, got)
		}
		time.Sleep(time.Millisecond)
	}

	releaseStart()
	if err := <-acquired; err != nil {
		t.Fatal(err)
	}
	requireStats(t, p, vmspool.Stats{Size: 1, Acquired: 1})
}

// readStatsUntil reads Stats repeatedly until stop is closed, reporting any
// snapshot that is inconsistent with a pool of the given size that at most
// maxAcquired callers can hold VMs from.
func readStatsUntil(t *testing.T, p *vmspool.Pool, stop <-chan struct{}, size, maxAcquired int) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		s := p.Stats()
		if v := statsViolation(s, size); v != "" {
			t.Errorf("%s: %+v", v, s)
			return
		}
		if s.Acquired > maxAcquired {
			t.Errorf("Acquired = %d with only %d callers: %+v", s.Acquired, maxAcquired, s)
			return
		}
	}
}

// acquireDeleteRounds acquires and deletes a VM the given number of times.
func acquireDeleteRounds(ctx context.Context, t *testing.T, p *vmspool.Pool, rounds int) {
	for range rounds {
		vm, err := p.Acquire(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		if err := vm.Delete(ctx); err != nil {
			t.Error(err)
			return
		}
	}
}

// TestPoolStatsConcurrent reads Stats from many goroutines while others
// acquire and delete VMs, checking under the race detector that reading it is
// safe and that the counts are always consistent.
func TestPoolStatsConcurrent(t *testing.T) {
	ctx := context.Background()
	const size, workers, rounds = 4, 4, 15
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(size))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size})

	stop := make(chan struct{})
	var readers, writers sync.WaitGroup
	for range 4 {
		readers.Go(func() { readStatsUntil(t, p, stop, size, workers) })
	}
	for range workers {
		writers.Go(func() { acquireDeleteRounds(ctx, t, p, rounds) })
	}
	writers.Wait()
	close(stop)
	readers.Wait()

	// Every VM was deleted and replaced, so the pool ends up full again.
	waitForStats(t, p, vmspool.Stats{Size: size, Available: size})
}

// TestPoolDeleteTwice verifies that deleting a VM more than once, from one
// goroutine or several, gives up its slot once only. Replenishing for every
// call would grow the pool beyond its size, leaving the surplus VMs pending,
// blocked on a full pool, until it was closed.
func TestPoolDeleteTwice(t *testing.T) {
	ctx := context.Background()
	const callers = 8
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, vmspool.WithSize(1))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	requireSettled(t, p, 1)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			if err := vm.Delete(ctx); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	// One replacement, however many times the VM was deleted.
	requireSettled(t, p, 1)
	if err := vm.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	requireSettled(t, p, 1)
	if got, want := len(factory.Mocks()), 2; got != want {
		t.Errorf("mocks created: got %d, want %d: a repeated Delete replenished the pool", got, want)
	}
}

// TestPoolStatsDeleteFails verifies that a VM whose deletion fails is no longer
// counted as acquired, since the pool has given it up, and that its slot is
// still refilled: a failed Delete must not leave the pool short.
func TestPoolStatsDeleteFails(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	bad := vmstestutil.NewMock("")
	bad.DeleteErr = errors.New("delete failed")
	factory.Inject(bad)
	p := vmspool.New(factory, vmspool.WithSize(1))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })
	requireSettled(t, p, 1)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.Delete(ctx); err == nil {
		t.Fatal("Delete: expected the failure to be reported")
	}
	requireSettled(t, p, 1)
	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	requireEmpty(t, p, 1)
}

// aliveMocks counts the VMs the factory has created that exist and have not
// been deleted. A VM whose Clone failed never existed, and is left in its
// initial state.
func aliveMocks(mocks []*vmstestutil.Mock) int {
	n := 0
	for _, m := range mocks {
		switch m.State(context.Background()) {
		case vms.StateDeleted, vms.StateInitial:
		default:
			n++
		}
	}
	return n
}

// TestPoolStatsUnderFailures runs many callers against a pool whose VMs fail
// in the ways that they can: creation fails, so Start must retry, and VMs that
// were created fine fail when Acquire starts them, so it must clean them up and
// replace them. Some callers stop VMs before deleting them. Throughout, the
// counts are checked against the invariants of every snapshot; once the callers
// are done, that nothing is left acquired or pending, that the counts agree
// with the VMs that actually exist, and that closing the pool deletes every one
// of them.
func TestPoolStatsUnderFailures(t *testing.T) {
	ctx := context.Background()
	const size, callers, rounds = 3, 6, 12

	factory := vmstestutil.NewMockFactory(true)
	for range 3 {
		bad := vmstestutil.NewMock("")
		bad.CloneErr = errors.New("clone failed")
		factory.Inject(bad)
	}
	// Staged stopped, these are created without error but fail when Acquire
	// starts them, and each replacement drawn from the queue does the same
	// until it is exhausted.
	for range 10 {
		bad := vmstestutil.NewMock("")
		bad.StartErr = errors.New("start failed")
		factory.Inject(bad)
	}

	p := vmspool.New(factory,
		vmspool.WithSize(size),
		vmspool.WithStagingBehaviour(vmspool.StagingBehaviourStopped),
		vmspool.WithCreateBackoff(fastCreateBackoff()))
	if err := p.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close(ctx) })

	stop := make(chan struct{})
	var readers, workers sync.WaitGroup
	for range 4 {
		readers.Go(func() { readStatsUntil(t, p, stop, size, callers) })
	}

	var failures atomic.Int64
	for i := range callers {
		workers.Go(func() {
			for range rounds {
				vm, err := p.Acquire(ctx)
				if err != nil {
					failures.Add(1)
					continue
				}
				if i%2 == 1 {
					_, _ = vm.StopAndRelease(ctx, time.Second)
				}
				if err := vm.Delete(ctx); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	workers.Wait()
	close(stop)
	readers.Wait()
	if failures.Load() == 0 {
		t.Error("no Acquire failed, so the failure paths were not exercised")
	}

	// Everything the callers took has been deleted, and every slot refilled.
	// Failed creations and acquisitions leave nothing counted.
	requireSettled(t, p, size)
	if got, want := aliveMocks(factory.Mocks()), size; got != want {
		t.Errorf("%d VMs exist, want the %d that are available", got, want)
	}

	if err := p.Close(ctx); err != nil {
		t.Fatal(err)
	}
	requireEmpty(t, p, size)
	if got := aliveMocks(factory.Mocks()); got != 0 {
		t.Errorf("%d VMs left undeleted after Close", got)
	}
	t.Logf("acquire failures: %d", failures.Load())
}

// TestStatsString verifies that Stats formats nicely as a string.
func TestStatsString(t *testing.T) {
	s := vmspool.Stats{
		Size:      3,
		Available: 2,
		Acquired:  1,
		Pending:   0,
	}
	wantStr := "size=3 available=2 acquired=1 pending=0"
	if got := s.String(); got != wantStr {
		t.Errorf("String(): got %q, want %q", got, wantStr)
	}
}
