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
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub := ps.Subscribe(context.Background())
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
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	subs := make([]*patterns.Subscriber[int], 4)
	for i := range subs {
		subs[i] = ps.Subscribe(context.Background())
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
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub := ps.Subscribe(context.Background())
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
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub := ps.Subscribe(context.Background())
	ps.Unsubscribe(sub)
	ps.Unsubscribe(sub) // second call must not panic
	ps.Close()

	drainSub(sub) // drain (empty)
}

func TestPubSubUnsubscribePartial(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub1 := ps.Subscribe(context.Background())
	sub2 := ps.Subscribe(context.Background())

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
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub := ps.Subscribe(context.Background())
	ps.Close()
	ps.Close() // must not panic

	drainSub(sub)
}

func TestPubSubSubscribeAfterClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)
	ps.Close()

	// Subscribe on a closed PubSub immediately closes the subscriber channel.
	sub := ps.Subscribe(context.Background())
	_, ok := <-sub.C()
	if ok {
		t.Error("expected channel to be closed immediately after subscribing to a closed PubSub")
	}
}

func TestPubSubPublishAfterClose(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](10)

	sub := ps.Subscribe(context.Background())
	ps.Close()
	ps.Publish(42) // must not panic; silently dropped

	if got := drainSub(sub); len(got) != 0 {
		t.Errorf("expected no items after Publish to closed PubSub, got %v", got)
	}
}

func TestPubSubDropOldest(t *testing.T) {
	defer synctestutil.AssertNoGoroutines(t)()
	ps := patterns.New[int](3)

	sub := ps.Subscribe(context.Background())
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
	defer synctestutil.AssertNoGoroutines(t)()
	// Each subscriber owns a separate FIFO with its own capacity-2 buffer.
	// With 8 publishes, each subscriber buffers at most 2 of the published
	// values. The internal buffer is managed exclusively by each run goroutine,
	// so drop-oldest is deterministic: the last capacity items are always
	// delivered in order.
	const (
		capacity  = 2
		publishes = 8
	)
	ps := patterns.New[int](capacity)

	sub1 := ps.Subscribe(context.Background())
	sub2 := ps.Subscribe(context.Background())

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
	defer synctestutil.AssertNoGoroutines(t)()
	const (
		publishers = 8
		perPub     = 20
		capacity   = publishers * perPub
	)
	ps := patterns.New[int](capacity)

	sub := ps.Subscribe(context.Background())

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
	defer synctestutil.AssertNoGoroutines(t)()
	// Tiny capacity so the internal buffer fills after just 2 items.
	const (
		capacity = 2
		count    = 1000
	)
	ps := patterns.New[int](capacity)

	// Subscribe but deliberately do not read — simulates a completely stalled
	// consumer.
	sub := ps.Subscribe(context.Background())

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
	ps := patterns.New[int](8)

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		subs []*patterns.Subscriber[int]
	)

	// Concurrently subscribe.
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := ps.Subscribe(context.Background())
			mu.Lock()
			subs = append(subs, sub)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// Publish while unsubscribing half the subscribers.
	var pubWg sync.WaitGroup
	pubWg.Add(1)
	go func() {
		defer pubWg.Done()
		for i := range 10 {
			ps.Publish(i)
		}
	}()

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
