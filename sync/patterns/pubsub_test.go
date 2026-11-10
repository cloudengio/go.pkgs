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

// drainSub collects all items from sub.C() until the channel closes.
func drainSub[T any](sub *patterns.Subscriber[T]) []T {
	var out []T
	for v := range sub.C() {
		out = append(out, v)
	}
	return out
}

func TestPubSubSingleSubscriber(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Publish(1)
	ps.Publish(2)
	ps.Publish(3)
	ps.Close()

	got := drainSub(sub)
	if !slices.Equal(got, []int{1, 2, 3}) {
		t.Errorf("got %v, want [1 2 3]", got)
	}
}

func TestPubSubMultipleSubscribers(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	subs := make([]*patterns.Subscriber[int], 4)
	for i := range subs {
		subs[i] = ps.Subscribe(context.Background(), 10)
	}

	ps.Publish(10)
	ps.Publish(20)
	ps.Close()

	want := []int{10, 20}
	for i, sub := range subs {
		if got := drainSub(sub); !slices.Equal(got, want) {
			t.Errorf("sub[%d]: got %v, want %v", i, got, want)
		}
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Publish(1)
	ps.Unsubscribe(sub) // closes sub's In(); run exits; Out() will close
	ps.Publish(2)       // sub is gone from the map, so 2 is not sent to it
	ps.Close()

	// sub receives 1 (already forwarded to out before Unsubscribe), not 2.
	if got := drainSub(sub); !slices.Equal(got, []int{1}) {
		t.Errorf("got %v, want [1]", got)
	}
}

func TestPubSubUnsubscribeIdempotent(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Unsubscribe(sub)
	ps.Unsubscribe(sub) // second call must not panic
	ps.Close()

	drainSub(sub) // drain (empty)
}

func TestPubSubUnsubscribePartial(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub1 := ps.Subscribe(context.Background(), 10)
	sub2 := ps.Subscribe(context.Background(), 10)

	ps.Publish(1)
	ps.Unsubscribe(sub1)
	ps.Publish(2)
	ps.Close()

	if got := drainSub(sub1); !slices.Equal(got, []int{1}) {
		t.Errorf("sub1: got %v, want [1]", got)
	}
	if got := drainSub(sub2); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("sub2: got %v, want [1 2]", got)
	}
}

func TestPubSubCloseIdempotent(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Close()
	ps.Close() // must not panic

	drainSub(sub)
}

func TestPubSubSubscribeAfterClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()
	ps.Close()

	// Subscribe on a closed PubSub immediately closes the subscriber channel.
	sub := ps.Subscribe(context.Background(), 10)
	_, ok := <-sub.C()
	if ok {
		t.Error("expected channel to be closed immediately after subscribing to a closed PubSub")
	}
}

func TestPubSubPublishAfterClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Close()
	ps.Publish(42) // must not panic; silently dropped

	if got := drainSub(sub); len(got) != 0 {
		t.Errorf("expected no items after Publish to closed PubSub, got %v", got)
	}
}

// TestPubSubPublishAfterSubscriberContextCancel verifies that Publish does not
// block when a subscriber's context has been cancelled.
//
// Cancelling sub's context exits its FIFO run goroutine via ctx.Done() whilst
// the subscriber is still in ps.subscribers. A Publish that only sent to
// sub.ch.In() (unbuffered b.in) would then block forever, since nothing reads
// it any more, and would do so holding ps.mu.RLock, which in turn would
// prevent Close and Unsubscribe from ever acquiring ps.mu.Lock: a total
// deadlock. Publish instead selects on the subscriber's Done channel and
// prunes it, see Subscriber.Done.
func TestPubSubPublishAfterSubscriberContextCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, 2*time.Second)()

	ps := patterns.NewPubSub[int]()

	// Use a short-lived context so the run goroutine exits almost immediately.
	subCtx, cancelSub := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancelSub()
	_ = ps.Subscribe(subCtx, 4)

	// Wait long enough that the run goroutine has seen ctx.Done() and returned.
	time.Sleep(50 * time.Millisecond)

	// Publish must not block.
	done := make(chan struct{})
	go func() {
		ps.Publish(1)
		close(done)
	}()

	select {
	case <-done:
		ps.Close()
	case <-time.After(200 * time.Millisecond):
		t.Error("Publish deadlocked: blocked forever after subscriber context was cancelled")
		// ps.Close() would also deadlock here — Publish holds ps.mu.RLock
		// and Close needs ps.mu.Lock — so there is nothing to clean up with,
		// and the goroutine leak the deferred check reports is a consequence
		// of the failure above rather than a second fault.
	}
}

// TestPubSubSubscriberDoneOnUnsubscribe covers the Done channel of a
// subscription that is released by its owner: it is closed only once the items
// the subscriber had buffered have been received, so a consumer selecting on
// both C and Done sees every one of them.
func TestPubSubSubscriberDoneOnUnsubscribe(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	ps.Publish(1)
	ps.Publish(2)

	select {
	case <-sub.Done():
		t.Fatal("Done closed whilst the subscription was still live")
	default:
	}

	ps.Unsubscribe(sub)

	// The consumer loop a cancellable subscription requires: both arms, with
	// Done only reached once C has been closed and drained.
	var got []int
done:
	for {
		select {
		case v, ok := <-sub.C():
			if !ok {
				break done
			}
			got = append(got, v)
		case <-sub.Done():
			t.Fatal("Done closed before the buffered items were received")
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for the subscriber's items")
		}
	}
	if !slices.Equal(got, []int{1, 2}) {
		t.Errorf("got %v, want [1 2]", got)
	}

	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Error("Done was not closed after the subscription ended")
	}
}

// TestPubSubSubscriberDoneOnContextCancel covers the Done channel of a
// subscription ended by cancelling the context it was created with, which is
// the case a consumer cannot detect any other way: C is abandoned rather than
// closed, so ranging over it alone would wait forever.
func TestPubSubSubscriberDoneOnContextCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()
	defer ps.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sub := ps.Subscribe(ctx, 10)
	cancel()

	select {
	case <-sub.Done():
	case <-time.After(time.Second):
		t.Fatal("Done was not closed when the subscription's context was cancelled")
	}

	// C is left open: a receive from it yields nothing rather than ok=false.
	select {
	case _, ok := <-sub.C():
		if !ok {
			t.Error("C was closed by cancellation; expected it to be abandoned")
		} else {
			t.Error("C delivered an item after cancellation")
		}
	default:
	}
}

// TestPubSubForward covers feeding a PubSub from a channel, which is how a
// producer that hands out a receive channel is connected to one. Closing the
// producer's channel ends the forwarding and nothing else: the PubSub is still
// open, since it may have other sources and only its owner closes it.
func TestPubSubForward(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 10)
	src := make(chan int)
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		ps.Forward(context.Background(), src)
	}()

	for i := 1; i <= 3; i++ {
		src <- i
	}
	close(src)

	select {
	case <-forwarded:
	case <-time.After(time.Second):
		t.Fatal("Forward did not return when its source channel was closed")
	}

	// The PubSub outlives the source, so it is still publishing.
	ps.Publish(4)
	ps.Close()

	if got := drainSub(sub); !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Errorf("got %v, want [1 2 3 4]", got)
	}
}

// TestPubSubForwardReturnsOnClose covers a Forward whose PubSub is closed
// underneath it: it returns rather than waiting for a source that nothing is
// listening to any more.
func TestPubSubForwardReturnsOnClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()
	sub := ps.Subscribe(context.Background(), 10)

	// Never closed, and never written to again: only the Close can end this.
	src := make(chan int, 1)
	src <- 1
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		ps.Forward(context.Background(), src)
	}()

	ps.Close()
	select {
	case <-forwarded:
	case <-time.After(time.Second):
		t.Fatal("Forward did not return when the PubSub was closed")
	}
	drainSub(sub)
}

// TestPubSubForwardReturnsOnContextCancel covers the other way a Forward is
// ended, by the context of whoever started it, leaving the PubSub to carry on
// with its remaining sources.
func TestPubSubForwardReturnsOnContextCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()
	defer ps.Close()

	ctx, cancel := context.WithCancel(context.Background())
	src := make(chan int)
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		ps.Forward(ctx, src)
	}()

	cancel()
	select {
	case <-forwarded:
	case <-time.After(time.Second):
		t.Fatal("Forward did not return when its context was cancelled")
	}
}

// TestPubSubForwardOrder covers the ordering a single forwarding channel
// imposes: every subscriber observes the forwarded items in the order they
// were received, which concurrent Publish calls do not guarantee.
func TestPubSubForwardOrder(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	const count = 50
	ps := patterns.NewPubSub[int]()

	subs := []*patterns.Subscriber[int]{
		ps.Subscribe(context.Background(), count),
		ps.Subscribe(context.Background(), count),
	}
	src := make(chan int)
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		ps.Forward(context.Background(), src)
	}()

	want := make([]int, 0, count)
	for i := range count {
		src <- i
		want = append(want, i)
	}
	close(src)
	<-forwarded
	ps.Close()

	for i, sub := range subs {
		if got := drainSub(sub); !slices.Equal(got, want) {
			t.Errorf("sub[%d]: got %v, want %v", i, got, want)
		}
	}
}

func TestPubSubDropOldest(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), 3)
	// Sequential Publish calls are safe: each blocks until the FIFO's run
	// goroutine reads the item, which means the prior item has already been
	// forwarded to out before the next Publish can proceed.
	ps.Publish(1) // out = [1]
	ps.Publish(2) // out = [1, 2]
	ps.Publish(3) // out = [1, 2, 3]  full
	ps.Publish(4) // drops 1 → [2, 3, 4]
	ps.Publish(5) // drops 2 → [3, 4, 5]
	ps.Close()

	if got := drainSub(sub); !slices.Equal(got, []int{3, 4, 5}) {
		t.Errorf("got %v, want [3 4 5]", got)
	}
}

func TestPubSubDropOldestPerSubscriber(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	// Each subscriber owns a separate FIFO with its own capacity-2 buffer.
	// With 8 publishes, each subscriber buffers at most 2 of the published
	// values. The internal buffer is managed exclusively by each run goroutine,
	// so drop-oldest is deterministic: the last capacity items are always
	// delivered in order.
	const (
		capacity  = 2
		publishes = 8
	)
	ps := patterns.NewPubSub[int]()

	sub1 := ps.Subscribe(context.Background(), capacity)
	sub2 := ps.Subscribe(context.Background(), capacity)

	for i := range publishes {
		ps.Publish(i + 1)
	}
	ps.Close()

	want := []int{publishes - capacity + 1, publishes} // last two published values
	for idx, sub := range []*patterns.Subscriber[int]{sub1, sub2} {
		if got := drainSub(sub); !slices.Equal(got, want) {
			t.Errorf("sub[%d]: got %v, want %v", idx, got, want)
		}
	}
}

func TestPubSubConcurrentPublish(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	const (
		publishers = 8
		perPub     = 20
		capacity   = publishers * perPub
	)
	ps := patterns.NewPubSub[int]()

	sub := ps.Subscribe(context.Background(), capacity)

	var wg sync.WaitGroup
	for p := range publishers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range perPub {
				ps.Publish(id*1000 + j)
			}
		}(p)
	}
	wg.Wait()
	ps.Close()

	got := drainSub(sub)
	// With capacity == publishers*perPub no items should be dropped.
	if len(got) != publishers*perPub {
		t.Errorf("got %d items, want %d", len(got), publishers*perPub)
	}
	for _, v := range got {
		id, j := v/1000, v%1000
		if id < 0 || id >= publishers || j < 0 || j >= perPub {
			t.Errorf("out-of-range value %d (id=%d j=%d)", v, id, j)
		}
	}
}

// TestPubSubPublishNotBlockedBySlowSubscriber verifies that Publish never
// blocks due to a subscriber that is not reading from its output channel.
//
// Why this holds: each subscriber's FIFO.run goroutine always includes
// `case v, ok := <-b.in` in its select — both when the internal buffer is
// empty and when it is non-empty. So run immediately accepts every item from
// Publish regardless of whether any consumer is reading from b.out. When the
// internal buffer is full, run drops the oldest entry and accepts the new one;
// the consumer's read pace has no influence on how quickly Publish returns.
func TestPubSubPublishNotBlockedBySlowSubscriber(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	// Tiny capacity so the internal buffer fills after just 2 items.
	const (
		capacity = 2
		count    = 1000
	)
	ps := patterns.NewPubSub[int]()

	// Subscribe but deliberately do not read — simulates a completely stalled
	// consumer.
	sub := ps.Subscribe(context.Background(), capacity)

	// All 1000 publishes must complete well within 5 seconds. If Publish
	// blocked per consumer read even for 1 ms each, this would take 1 s.
	done := make(chan struct{})
	go func() {
		for i := range count {
			ps.Publish(i)
		}
		close(done)
	}()

	select {
	case <-done:
		// Publishes completed without blocking on the stalled subscriber.
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked: did not complete within 5s; likely waiting on slow subscriber")
	}

	// Close and drain so the run goroutine can flush and exit cleanly.
	ps.Close()
	drainSub(sub)
}

func TestPubSubConcurrentSubscribeUnsubscribe(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ps := patterns.NewPubSub[int]()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		subs []*patterns.Subscriber[int]
	)

	// Concurrently subscribe.
	for range 8 {
		wg.Go(func() {
			sub := ps.Subscribe(context.Background(), 8)
			mu.Lock()
			subs = append(subs, sub)
			mu.Unlock()
		})
	}
	wg.Wait()

	// Publish while unsubscribing half the subscribers.
	var pubWg sync.WaitGroup
	pubWg.Go(func() {
		for i := range 10 {
			ps.Publish(i)
		}
	})

	for _, sub := range subs[:len(subs)/2] {
		ps.Unsubscribe(sub)
	}
	pubWg.Wait()
	ps.Close()

	// Drain all remaining subscribers.
	for _, sub := range subs {
		drainSub(sub)
	}
}
