// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/sync/patterns"
)

// DefaultStatusBuffer is the number of events a StatusTracker buffers for a
// pool whilst they are waiting to be counted, see NewStatusTracker.
const DefaultStatusBuffer = 100

// StatusTracker counts the lifecycle events of one or more pools by kind,
// logs them, and notifies its subscribers whenever those counts change. It is
// safe for concurrent use.
//
// It uses nothing of a pool but its public API: a pool reports to a channel,
// see WithStatus, and the tracker counts what arrives there. Every event names
// the pool it came from, see Event.Pool, so one tracker serves a whole set of
// pools and its change signal covers all of them.
type StatusTracker struct {
	// buffer is the capacity of each channel Track creates, and of the
	// tracker's own subscription to events.
	buffer int

	mu       sync.Mutex
	counters map[string]map[EventKind]int // pool name -> event kind -> count
	updated  map[string]time.Time         // pool name -> time of its last event

	// events is the single sink the pools' own channels are forwarded into, so
	// that a burst from one cannot displace another's events, see Track.
	events *patterns.PubSub[Event]
	// changes is the coalescing change signal handed to subscribers.
	changes *patterns.PubSub[struct{}]
}

// NewStatusTracker returns a tracker and starts the goroutine that counts and
// logs what the pools report, which Close ends. buffer is how many of a pool's
// events may be waiting to be counted before the oldest are dropped, and is
// the capacity of each channel Track creates; 0 or less is
// DefaultStatusBuffer.
//
// Events are logged as they are counted, through the logger carried by ctx at
// Info, or Error for an event carrying one: that is the only report a pool's
// asynchronous failures ever get. ctx is used for its logger alone, so
// cancelling it does not stop the counting; Close does.
func NewStatusTracker(ctx context.Context, buffer int) *StatusTracker {
	if buffer <= 0 {
		buffer = DefaultStatusBuffer
	}
	t := &StatusTracker{
		buffer:   buffer,
		counters: make(map[string]map[EventKind]int),
		updated:  make(map[string]time.Time),
		events:   patterns.NewPubSub[Event](),
		changes:  patterns.NewPubSub[struct{}](),
	}
	// Subscribed before any pool can be tracked, so that nothing is published
	// before there is anything counting it.
	sub := t.events.Subscribe(context.WithoutCancel(ctx), buffer)
	go t.count(ctx, sub)
	return t
}

// Track returns the option that connects a pool to the tracker, to be passed
// to New alongside the pool's other options:
//
//	pool := vmspool.New(provider, "builders", tracker.Track(ctx), vmspool.WithSize(4))
//
// Each call creates a channel of its own, counted from here rather than from
// when the pool is created, so nothing a pool reports as it fills is missed;
// Close releases a channel whose option never reached a pool.
//
// It registers that channel with WithStatus, which holds one, so a pool given
// both reports to whichever was applied last. A caller that wants both keeps
// its own channel and feeds the tracker with Watch.
func (t *StatusTracker) Track(ctx context.Context) Option {
	ch := make(chan Event, t.buffer)
	t.Watch(ctx, ch)
	return WithStatus(ch)
}

// Watch counts the events arriving on ch until Close ends the tracker, for a
// caller that has a status channel already; Track is the usual way to connect
// a pool. Closing ch ends the forwarding of that channel alone and takes
// nothing out of the counters.
func (t *StatusTracker) Watch(ctx context.Context, ch <-chan Event) {
	go t.events.Forward(context.WithoutCancel(ctx), ch)
}

// count counts and logs the pools' events until Close ends the tracker, which
// closes sub's channel once what it had buffered has been counted.
func (t *StatusTracker) count(ctx context.Context, sub *patterns.Subscriber[Event]) {
	defer t.events.Unsubscribe(sub)
	for {
		select {
		case <-sub.Done():
			return
		case e, ok := <-sub.C():
			if !ok {
				return
			}
			t.record(e)
			t.log(ctx, e)
		}
	}
}

// log reports one event, which for an asynchronous failure such as a failed
// replenishment is the only report it gets.
func (t *StatusTracker) log(ctx context.Context, e Event) {
	args := make([]any, 0, 8)
	args = append(args, "pool", e.Pool, "event", e.Kind)
	if e.VM != "" {
		args = append(args, "vm", e.VM)
	}
	if e.Err != nil {
		ctxlog.Error(ctx, "vm pool event", append(args, "error", e.Err)...)
		return
	}
	ctxlog.Info(ctx, "vm pool event", args...)
}

// record counts one event against the pool that emitted it and signals the
// change.
func (t *StatusTracker) record(e Event) {
	t.mu.Lock()
	counters := t.counters[e.Pool]
	if counters == nil {
		counters = make(map[EventKind]int)
		t.counters[e.Pool] = counters
	}
	counters[e.Kind]++
	t.updated[e.Pool] = e.Time
	t.mu.Unlock()
	t.changes.Publish(struct{}{})
}

// Counters returns a copy of the named pool's counts by event kind and the
// time of the last event counted for it; a pool that has reported nothing has
// neither. A pool reports asynchronously, so an event of an operation that has
// just returned may not have been counted yet.
func (t *StatusTracker) Counters(pool string) (map[EventKind]int, time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[EventKind]int, len(t.counters[pool]))
	maps.Copy(out, t.counters[pool])
	return out, t.updated[pool]
}

// Pools returns the names of the pools the tracker has counted an event for,
// sorted. A pool that has reported nothing yet is not among them.
func (t *StatusTracker) Pools() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Sorted(maps.Keys(t.counters))
}

// Subscribe returns a channel signalled whenever any tracked pool's counters
// change, and a function that must be called to release the subscription;
// cancelling ctx releases it too, and releasing it twice is harmless. Signals
// carry nothing and are coalesced, so a reader takes a fresh set of Counters.
func (t *StatusTracker) Subscribe(ctx context.Context) (<-chan struct{}, func()) {
	sub := t.changes.Subscribe(ctx, 1)
	return sub.C(), func() { t.changes.Unsubscribe(sub) }
}

// Close ends the counting, every subscription and the forwarding of every
// channel the tracker was given; later events are discarded. It is idempotent,
// and the counters it accumulated stand.
func (t *StatusTracker) Close() {
	t.events.Close()
	t.changes.Close()
}
