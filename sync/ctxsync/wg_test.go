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
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	wg.Wait(ctx)
	if got, want := ctx.Err().Error(), "context canceled"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWaitGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
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

// TestWaitGroup_ConcurrentAddDone races many goroutines calling Done while
// Wait is blocked on the same WaitGroup. Run with -race.
func TestWaitGroup_ConcurrentAddDone(t *testing.T) {
	const n = 100
	var wg ctxsync.WaitGroup
	wg.Add(n + 1)
	releaseDone := make(chan struct{})
	waitStarted := make(chan struct{})
	waitReturned := make(chan struct{})
	for range n {
		go func() {
			<-releaseDone
			wg.Done()
		}()
	}
	go func() {
		close(waitStarted)
		wg.Wait(t.Context())
		close(waitReturned)
	}()
	<-waitStarted
	close(releaseDone)
	wg.Done()
	<-waitReturned
}

// TestWaitGroup_CancelRacesWithCompletion races context cancellation against
// the WaitGroup counter reaching zero. Run with -race.
func TestWaitGroup_CancelRacesWithCompletion(t *testing.T) {
	const iterations = 500
	for range iterations {
		var wg ctxsync.WaitGroup
		ctx, cancel := context.WithCancel(t.Context())
		wg.Add(1)
		go cancel()
		go wg.Done()
		wg.Wait(ctx)
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
			wg.Wait(t.Context())
		}()
	}
	wg.Done()
	stdWG.Wait()
}

// TestWaitGroup_CancelledWaitDoneCleanup verifies that after Wait returns due
// to context cancellation, Done does not race and Add can be called immediately
// to reuse the WaitGroup. Run with -race.
func TestWaitGroup_CancelledWaitDoneCleanup(t *testing.T) {
	var wg ctxsync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // pre-cancel so Wait returns immediately
	wg.Wait(ctx)

	wg.Done() // must not race with a concurrent Add or a subsequent Wait

	// Immediate reuse must work correctly.
	wg.Add(1)
	go wg.Done()
	wg.Wait(t.Context())
}

// TestWaitGroup_CancelWhileManyGoroutinesRunning cancels the context while n
// goroutines are still outstanding, then confirms all goroutines finish without
// leaving the WaitGroup in a broken state. Run with -race.
func TestWaitGroup_CancelWhileManyGoroutinesRunning(t *testing.T) {
	const n = 50
	var wg ctxsync.WaitGroup
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var barrier, tracker sync.WaitGroup
	barrier.Add(n)
	tracker.Add(n)
	wg.Add(n)
	start := make(chan struct{})
	for range n {
		go func() {
			barrier.Done() // signal that this goroutine has started
			defer tracker.Done()
			<-start
			wg.Done()
		}()
	}
	barrier.Wait() // all goroutines are running
	cancel()       // cancel while they are still waiting
	wg.Wait(ctx)   // returns due to cancellation, not completion
	if ctx.Err() == nil {
		t.Fatal("Wait should have returned due to context cancellation")
	}
	close(start)   // release goroutines
	tracker.Wait() // all goroutines should finish cleanly
	wg.Wait(t.Context())
}

// TestWaitGroup_Go verifies that Go starts the function in a goroutine and
// that Wait unblocks once all goroutines have returned. Run with -race.
func TestWaitGroup_Go(t *testing.T) {
	const n = 50
	var wg ctxsync.WaitGroup
	var mu sync.Mutex
	results := make([]int, 0, n)
	for i := range n {
		wg.Go(func() {
			mu.Lock()
			results = append(results, i)
			mu.Unlock()
		})
	}
	wg.Wait(t.Context())
	if got, want := len(results), n; got != want {
		t.Errorf("got %d results, want %d", got, want)
	}
}

// TestWaitGroup_RapidCycles stresses Add/Done/Wait reuse across many iterations.
// Run with -race.
func TestWaitGroup_RapidCycles(t *testing.T) {
	const iterations = 500
	var wg ctxsync.WaitGroup
	for range iterations {
		wg.Add(1)
		go wg.Done()
		wg.Wait(t.Context())
	}
}
