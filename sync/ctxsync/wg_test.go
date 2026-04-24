// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxsync_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloudeng.io/sync/ctxsync"
)

func ExampleWaitGroup() {
	var wg ctxsync.WaitGroup
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // G118 false positive
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	wg.Wait(ctx)
	fmt.Println(ctx.Err())
	// Output:
	// context canceled
}

func TestWaitGroupInline(t *testing.T) {
	var wg ctxsync.WaitGroup
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wg.Wait(ctx)
	if got, want := ctx.Err().Error(), "context canceled"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWaitGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg ctxsync.WaitGroup
	wg.Add(1)
	var out string
	go func() {
		out = "done"
		wg.Done()
	}()
	wg.Wait(ctx)
	if got, want := out, "done"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	cancel()
	wg.Add(1)
	wg.Wait(ctx)
	// The test will timeout if we never get here.
}

// TestWaitGroup_ConcurrentAddDone races many goroutines calling Done against
// a blocking Wait. Run with -race.
func TestWaitGroup_ConcurrentAddDone(t *testing.T) {
	const n = 100
	var wg ctxsync.WaitGroup
	wg.Add(n)
	for range n {
		go wg.Done()
	}
	wg.Wait(context.Background())
}

// TestWaitGroup_CancelRacesWithCompletion races context cancellation against
// the WaitGroup counter reaching zero. Run with -race.
func TestWaitGroup_CancelRacesWithCompletion(t *testing.T) {
	const iterations = 500
	for range iterations {
		var wg ctxsync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())
		wg.Add(1)
		go cancel()
		go wg.Done()
		wg.Wait(ctx)
		// Ensure the background goroutine spawned inside Wait can exit: by the
		// time we reach here either Done was already called (counter=0) or it
		// will be called imminently. Either way the goroutine unblocks quickly.
		cancel() // no-op if already cancelled; satisfies gosec G118
	}
}

// TestWaitGroup_MultipleConcurrentWaiters races several goroutines all calling
// Wait on the same WaitGroup concurrently. Run with -race.
func TestWaitGroup_MultipleConcurrentWaiters(t *testing.T) {
	const waiters = 10
	var wg ctxsync.WaitGroup
	wg.Add(1)

	var stdWG sync.WaitGroup
	stdWG.Add(waiters)
	for range waiters {
		go func() {
			defer stdWG.Done()
			wg.Wait(context.Background())
		}()
	}
	wg.Done()
	stdWG.Wait()
}

// TestWaitGroup_CancelledWaitDoneCleanup verifies that after Wait returns due
// to context cancellation, calling Done brings the internal goroutine to zero
// without a race. Callers must not call Add again until they can guarantee the
// internal goroutine from the cancelled Wait has exited (i.e. the counter
// reached zero and sync.WaitGroup.Wait fully returned). Run with -race.
func TestWaitGroup_CancelledWaitDoneCleanup(t *testing.T) {
	var wg ctxsync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so Wait returns immediately
	wg.Wait(ctx)

	// The goroutine spawned inside Wait is still blocked on sync.WaitGroup.Wait.
	// Calling Done releases it; this must not race.
	wg.Done()
}

// TestWaitGroup_CancelWhileManyGoroutinesRunning cancels the context while n
// goroutines are still outstanding, then confirms all goroutines finish without
// leaving the WaitGroup in a broken state. Run with -race.
func TestWaitGroup_CancelWhileManyGoroutinesRunning(t *testing.T) {
	const n = 50
	var wg ctxsync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var barrier, tracker sync.WaitGroup
	barrier.Add(n)
	tracker.Add(n)
	wg.Add(n)
	for range n {
		go func() {
			barrier.Done() // signal that this goroutine has started
			defer tracker.Done()
			time.Sleep(5 * time.Millisecond)
			wg.Done()
		}()
	}
	barrier.Wait() // all goroutines are running
	cancel()       // cancel while they are still sleeping
	wg.Wait(ctx)   // returns due to cancellation, not completion

	// Wait for all goroutines to finish so the test does not leak them.
	// After tracker.Wait() the counter is 0 and the goroutine spawned inside
	// wg.Wait(ctx) above has already exited (or will imminently).
	tracker.Wait()
}

// TestWaitGroup_RapidCycles stresses Add/Done/Wait reuse across many iterations.
// Run with -race.
func TestWaitGroup_RapidCycles(t *testing.T) {
	const iterations = 500
	var wg ctxsync.WaitGroup
	for range iterations {
		wg.Add(1)
		go wg.Done()
		wg.Wait(context.Background())
	}
}
