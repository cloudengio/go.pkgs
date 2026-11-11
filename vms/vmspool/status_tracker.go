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
// It is an entity of its own rather than part of a Pool, and it uses nothing
// of a pool but its public API: a pool reports its events to a channel, see
// WithStatus, and the tracker counts what arrives there. A pool can therefore
// be created with a tracker, without one, or with a status channel of the
// caller's own, and one tracker can count pools that are created and closed at
// different times.
//
// Every event names the pool that emitted it, see Event.Pool, so a single
// tracker serves a whole set of pools: it keeps the counters per pool and its
// change signal covers all of them, which is what a user interface watching
// every pool at once wants, and what one pool gets as a set of one.
type StatusTracker struct {
	// buffer is the capacity of each channel Track creates, and of the
	// tracker's own subscription to events.
	buffer int

	mu       sync.Mutex
	counters map[string]map[EventKind]int // pool name -> event kind -> count
	updated  map[string]time.Time         // pool name -> time of its last event

	// events carries the pools' events to the counting goroutine. Each pool
	// reports on a channel of its own, which is forwarded into this single
	// sink, see Track, so that a burst from one pool cannot displace another's
	// events before they are counted: what the pools share is the sink, not
	// the buffer.
	events *patterns.PubSub[Event]
	// changes is the coalescing change signal handed to subscribers.
	changes *patterns.PubSub[struct{}]
}

// NewStatusTracker returns a tracker and starts the goroutine that counts and
// logs what the pools report, which Close ends. buffer is how many of a pool's
// events may be waiting to be counted before the oldest of them are dropped,
// and is the capacity of each channel Track creates; DefaultStatusBuffer is
// used if it is 0 or negative.
//
// Events are logged as they are counted, through the logger carried by ctx,
// see cloudeng.io/logging/ctxlog: at Info, or at Error for an event that
// carries one. That is the only report some of a pool's failures ever get: a
// replenishment that could not create a VM, or a refresh that could not delete
// one, happens with no call of the caller's for an error to be returned from.
//
// The counting belongs to the pools rather than to whoever created them, so it
// is not tied to ctx: a tracker whose context has been cancelled still counts
// for the pools that are still running, and ctx is used for its logger alone.
// Close is what ends it.
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
// Each call creates a channel of its own for the pool to report on, so that a
// burst of events from one pool cannot displace another's before they are
// counted; a pool drops its events rather than blocking when that channel is
// full, which is what the tracker's buffer bounds, see NewStatusTracker.
//
// The channel is counted from the moment Track is called rather than from the
// moment the pool is created, so nothing a pool reports as it fills goes
// uncounted. An option that is never passed to a pool therefore leaves behind
// a channel that nothing ever sends on, which Close releases.
//
// Note that WithStatus registers a single channel, so a pool created with this
// option and WithStatus reports to whichever of them was applied last. A
// caller that wants both has one channel of its own, forwards what it reads to
// the tracker with Watch, and passes WithStatus rather than this option.
func (t *StatusTracker) Track(ctx context.Context) Option {
	ch := make(chan Event, t.buffer)
	t.Watch(ctx, ch)
	return WithStatus(ch)
}

// Watch counts the events that arrive on ch until Close ends the tracker. It
// is for a caller that has a status channel already, such as one that reports
// a pool's events elsewhere as well; Track is the usual way to connect a pool
// and creates the channel itself.
//
// Closing ch ends the forwarding of that one channel and not the tracker: a
// tracker counts any number of pools, each with a channel of its own, and
// nothing is taken out of the counters when one of them goes, since a pool
// that has been closed still reported what it reported.
//
// ctx is used for the forwarding goroutine's logger alone, as in
// NewStatusTracker: cancelling it does not stop the events of a pool that is
// still running from being counted.
func (t *StatusTracker) Watch(ctx context.Context, ch <-chan Event) {
	go t.events.Forward(context.WithoutCancel(ctx), ch)
}

// count counts and logs the pools' events until Close ends the tracker, which
// closes sub's channel once the events it had buffered have been counted. Done
// covers the subscription ending any other way, which the context it was
// created with rules out, see NewStatusTracker.
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

// log reports one event. The failures a pool reports this way are
// asynchronous - a replenishment, or a refresh that could not delete a VM -
// so this is the only place they are ever reported.
func (t *StatusTracker) log(ctx context.Context, e Event) {
	args := []any{"pool", e.Pool, "event", e.Kind}
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

// Counters returns a copy of the named pool's counts by event kind, so that a
// caller cannot reach the accounting itself, and the time of the most recent
// event counted for that pool. A pool that has reported nothing, including one
// that was never tracked, has no counters and the zero time.
//
// The counts are those of the events counted so far: a pool reports
// asynchronously, so an event of an operation that has just returned may not
// have been counted yet.
func (t *StatusTracker) Counters(pool string) (map[EventKind]int, time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[EventKind]int, len(t.counters[pool]))
	maps.Copy(out, t.counters[pool])
	return out, t.updated[pool]
}

// Pools returns the names of the pools the tracker has counted an event for,
// sorted, which is not every pool it was given: a pool that has reported
// nothing yet is not among them.
func (t *StatusTracker) Pools() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Sorted(maps.Keys(t.counters))
}

// Subscribe returns a channel that is signalled whenever the counters of any
// pool the tracker counts change, and a function that must be called to
// release the subscription; the subscription is also released when ctx is
// cancelled. Releasing it more than once is harmless.
//
// Signals carry no information and are coalesced, a subscriber's buffer being
// one, so a reader takes a fresh set of Counters rather than counting them. It
// is one subscription covering every pool the tracker counts rather than one
// per pool merged together.
func (t *StatusTracker) Subscribe(ctx context.Context) (<-chan struct{}, func()) {
	sub := t.changes.Subscribe(ctx, 1)
	return sub.C(), func() { t.changes.Unsubscribe(sub) }
}

// Close ends the counting, every subscription to the change signal, and the
// forwarding of every channel the tracker was given, whether or not the pools
// that report on them are closed; events that arrive afterwards are discarded.
// It is idempotent, so a pool that failed to start and a set of pools that is
// closed twice both end the counting exactly once.
//
// The counters are left as they are: a tracker that has stopped counting still
// reports what it counted.
func (t *StatusTracker) Close() {
	t.events.Close()
	t.changes.Close()
}
