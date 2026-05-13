// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package patterns_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"cloudeng.io/sync/patterns"
	"cloudeng.io/sync/synctestutil"
)

// continuousSender starts a goroutine that sends integers to f.In() until
// senderCtx is cancelled. The returned WaitGroup lets the caller join it.
func continuousSender(senderCtx context.Context, f *patterns.FIFO[int]) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case f.In() <- i:
			case <-senderCtx.Done():
				return
			}
		}
	}()
	return &wg
}

// TestBoundedFIFOOrdering verifies items exit in FIFO order when the buffer
// is large enough that nothing is dropped.
func TestBoundedFIFOOrdering(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 10)

	for i := range 5 {
		f.In() <- i
	}
	close(f.In())

	var got []int
	for v := range f.Out() {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{0, 1, 2, 3, 4}) {
		t.Errorf("got %v, want [0 1 2 3 4]", got)
	}
}

// TestBoundedFIFODropOldest verifies that the oldest buffered item is discarded
// when the output buffer is full.
func TestBoundedFIFODropOldest(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 3)

	// Sequential sends are safe: in is unbuffered, so each send blocks until
	// run receives it. run processes items one at a time, so by the time send
	// N+1 returns, send N has already been forwarded to out.
	for i := range 3 {
		f.In() <- i // out fills up: [0, 1, 2]
	}
	f.In() <- 3 // out full → drops 0 → [1, 2, 3]
	f.In() <- 4 // out full → drops 1 → [2, 3, 4]

	close(f.In())
	var got []int
	for v := range f.Out() {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{2, 3, 4}) {
		t.Errorf("got %v, want [2 3 4]", got)
	}
}

// TestBoundedFIFOSizeOne verifies a buffer of size one: every new item evicts
// the single buffered item.
func TestBoundedFIFOSizeOne(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 1)

	f.In() <- 1 // out = [1]
	f.In() <- 2 // drops 1 → out = [2]
	f.In() <- 3 // drops 2 → out = [3]

	close(f.In())
	var got []int
	for v := range f.Out() {
		got = append(got, v)
	}
	if !slices.Equal(got, []int{3}) {
		t.Errorf("got %v, want [3]", got)
	}
}

// TestBoundedFIFOCloseIn verifies that closing In() causes Out() to close
// after all buffered items have been drained (the clean-shutdown path).
func TestBoundedFIFOCloseIn(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 5)

	go func() {
		for i := range 4 {
			f.In() <- i
		}
		close(f.In())
	}()

	var got []int
	for v := range f.Out() { // exits when Out() closes
		got = append(got, v)
	}
	if !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Errorf("got %v, want [0 1 2 3]", got)
	}
}

// TestBoundedFIFOStop verifies that Stop() causes the run goroutine to exit
// while a continuous sender is active. doneCh is checked inside the select so
// it fires as soon as run is processing an item.
func TestBoundedFIFOStop(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 4)

	senderCtx, cancelSender := context.WithCancel(context.Background())
	wg := continuousSender(senderCtx, f)

	time.Sleep(5 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	f.Stop(stopCtx) // blocks until run exits

	cancelSender()
	wg.Wait()
}

// TestBoundedFIFOContextCancel verifies that cancelling the context passed to
// NewBoundedFIFO causes the run goroutine to exit. Like doneCh, ctx.Done() is
// only checked inside the select, so an active sender is needed to ensure run
// re-enters the select.
func TestBoundedFIFOContextCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()

	fifoCtx, cancelFIFO := context.WithCancel(context.Background())
	f := patterns.NewFIFO[int](fifoCtx, 4)

	senderCtx, cancelSender := context.WithCancel(context.Background())
	wg := continuousSender(senderCtx, f)

	time.Sleep(5 * time.Millisecond)
	cancelFIFO()

	// Stop() doubles as a barrier: wg.Wait inside it blocks until run exits.
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	f.Stop(stopCtx)

	cancelSender()
	wg.Wait()
}

// TestBoundedFIFOOutNotClosedAfterStop verifies that Out() is NOT closed after
// Stop(). When run exits via return (Stop or ctx cancel), it skips the
// close(b.out) call that follows the for-range loop.
func TestBoundedFIFOOutNotClosedAfterStop(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 4)

	senderCtx, cancelSender := context.WithCancel(context.Background())
	wg := continuousSender(senderCtx, f)

	time.Sleep(5 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	f.Stop(stopCtx)
	cancelSender()
	wg.Wait()

	// Drain whatever items remain in the buffer.
	for {
		select {
		case _, ok := <-f.Out():
			if !ok {
				t.Fatal("Out() closed unexpectedly after Stop()")
			}
		default:
			goto drained
		}
	}
drained:
	// A fresh blocking receive must time out, not return ok=false.
	select {
	case _, ok := <-f.Out():
		if !ok {
			t.Error("Out() closed after Stop(); expected it to remain open")
		}
	case <-time.After(50 * time.Millisecond):
		// Out() is open but empty — correct
	}
}

// TestBoundedFIFOStopWhileIdle verifies that Stop() exits the run goroutine
// even when there is no active sender. The run goroutine always blocks in a
// select that includes doneCh, so Stop() is effective regardless of load.
func TestBoundedFIFOStopWhileIdle(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 3)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	f.Stop(stopCtx) // exits promptly even with no sender
}

// TestBoundedFIFOConcurrentSenders exercises multiple goroutines sending to
// In() simultaneously. Because In() is unbuffered, senders serialise through
// the run goroutine — no data corruption or panics should occur.
func TestBoundedFIFOConcurrentSenders(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	f := patterns.NewFIFO[int](context.Background(), 16)

	const (
		senders   = 8
		perSender = 50
	)
	var wg sync.WaitGroup
	for i := range senders {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range perSender {
				f.In() <- id*1000 + j
			}
		}(i)
	}
	wg.Wait()
	close(f.In())

	var got []int
	for v := range f.Out() {
		got = append(got, v)
	}
	for _, v := range got {
		id, j := v/1000, v%1000
		if id < 0 || id >= senders || j < 0 || j >= perSender {
			t.Errorf("out-of-range value %d (id=%d j=%d)", v, id, j)
		}
	}
	if len(got) == 0 {
		t.Error("expected some items in Out(), got none")
	}
}

// TestBoundedFIFOSendNotBlockedBySlowConsumer verifies that sending to In()
// never blocks due to a consumer that is not reading from Out().
//
// Why this holds: run's select always includes `case v, ok := <-b.in` in
// both the empty-buffer and non-empty-buffer branches. When the internal
// buffer is full, run drops the oldest entry and accepts the new one — all
// without touching b.out. The consumer's read pace has no influence on how
// quickly a send to In() returns.
func TestBoundedFIFOSendNotBlockedBySlowConsumer(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	const (
		capacity = 2
		count    = 1000
	)
	f := patterns.NewFIFO[int](context.Background(), capacity)

	// Send 1000 items with no consumer reading Out(). Each send must return
	// as soon as run receives the item, regardless of consumer pace.
	done := make(chan struct{})
	go func() {
		for i := range count {
			f.In() <- i
		}
		close(done)
	}()

	select {
	case <-done:
		// All sends completed without blocking on the stalled consumer.
	case <-time.After(5 * time.Second):
		t.Fatal("In() blocked: did not complete within 5s; likely waiting on slow consumer")
	}

	// Close In() and drain Out() so run can flush and exit cleanly.
	close(f.In())
	for range f.Out() {
	}
}
