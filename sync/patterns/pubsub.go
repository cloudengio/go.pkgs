// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package patterns

import (
	"context"
	"sync"
)

// Subscriber represents a subscription to a PubSub instance.
type Subscriber[T any] struct {
	ch    *FIFO[T]
	alive chan struct{} // closed when ch's run goroutine exits for any reason
}

// C returns the underlying receive-only channel for use in select statements.
//
// C is closed when the subscription is released by Unsubscribe or PubSub.Close,
// but not when the context passed to Subscribe is cancelled, which abandons it
// instead. A consumer that cannot guarantee that context outlives the
// subscription must therefore select on Done as well, or risk waiting on C
// forever; ranging over C alone is only safe for a subscription whose context
// is not cancelled.
func (s *Subscriber[T]) C() <-chan T {
	return s.ch.Out()
}

// Done returns a channel that is closed once nothing further will be delivered
// to C, whatever the reason: Unsubscribe, PubSub.Close, or cancellation of the
// context passed to Subscribe.
//
// When the subscription is released by Unsubscribe or PubSub.Close, Done is
// closed after every buffered item has been received from C and C itself has
// been closed, so a consumer selecting on both arms still sees all of them.
// When the context is cancelled instead, any buffered items are discarded and
// Done is the only indication that the subscription has ended, since C is left
// open.
func (s *Subscriber[T]) Done() <-chan struct{} {
	return s.alive
}

const (
	DefaultPubSubCapacity = 100
)

// PubSub provides a concurrent pub-sub mechanism that drops the oldest
// items for slow subscribers when their buffer is full.
type PubSub[T any] struct {
	mu          sync.RWMutex
	subscribers map[*Subscriber[T]]struct{}
	closed      bool
	// done is closed by Close, so that a Forward in progress ends with the
	// PubSub it is feeding rather than waiting on its source.
	done chan struct{}
}

// NewPubSub returns a new PubSub instance.
func NewPubSub[T any]() *PubSub[T] {
	return &PubSub[T]{
		subscribers: make(map[*Subscriber[T]]struct{}),
		done:        make(chan struct{}),
	}
}

// Subscribe creates and returns a new Subscriber with the given buffer capacity.
// If capacity is <=0, it defaults to DefaultPubSubCapacity.
//
// ctx governs the goroutine that buffers items for this subscriber: cancelling
// it ends the subscription, discarding whatever it had buffered and leaving
// Subscriber.C open rather than closing it, see Subscriber.Done.
//
// A subscription holds a goroutine until it ends, which it does when the
// subscriber's channel is closed and drained, when ctx is cancelled, or when
// the PubSub is closed and the subscriber drains the items it had buffered. A
// consumer that abandons Subscriber.C without draining it must therefore
// subscribe with a context it cancels, or the subscription outlives it.
func (ps *PubSub[T]) Subscribe(ctx context.Context, capacity int) *Subscriber[T] {
	if capacity <= 0 {
		capacity = DefaultPubSubCapacity
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	alive := make(chan struct{})
	sub := &Subscriber[T]{
		ch:    newFIFO[T](ctx, capacity, alive),
		alive: alive,
	}
	if ps.closed {
		close(sub.ch.In())
		return sub
	}
	ps.subscribers[sub] = struct{}{}
	return sub
}

// Unsubscribe removes a subscriber and closes its underlying channel. Calling
// it more than once, or for a subscriber of a closed PubSub, is harmless.
//
// The subscriber's channel is closed once the items it has already buffered
// have been received from it, so the subscriber must drain it for the
// subscription to be released, see Subscribe.
func (ps *PubSub[T]) Unsubscribe(sub *Subscriber[T]) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, ok := ps.subscribers[sub]; ok {
		delete(ps.subscribers, sub)
		close(sub.ch.In())
	}
}

// Publish sends an item to all active subscribers. If a subscriber's buffer
// is full, its oldest item is dropped to make room for the new one.
// Subscribers whose run goroutine has exited (e.g. context cancelled) are
// detected via their alive channel and pruned from the map without blocking.
//
// Items published by concurrent calls are not ordered with respect to one
// another, and two subscribers may observe two such items in opposite orders.
// Forward, which serializes what it publishes through a single channel, is the
// way to give every subscriber the same order.
func (ps *PubSub[T]) Publish(item T) {
	ps.mu.RLock()
	if ps.closed {
		ps.mu.RUnlock()
		return
	}
	var dead []*Subscriber[T]
	for sub := range ps.subscribers {
		// The send is safe only because a subscriber's channel is closed under
		// the write lock, by Unsubscribe or Close, which this read lock
		// excludes: moving either close out of the lock would make this a send
		// on a closed channel. The send itself does not block for long, since
		// the FIFO behind it always accepts an item and drops its oldest to
		// make room, see FIFO.run.
		select {
		case sub.ch.In() <- item:
		case <-sub.Done():
			dead = append(dead, sub)
		}
	}
	ps.mu.RUnlock()
	if len(dead) > 0 {
		ps.mu.Lock()
		for _, sub := range dead {
			delete(ps.subscribers, sub)
		}
		ps.mu.Unlock()
	}
}

// Forward publishes every item it receives from ch, returning when ch is
// closed, when ctx is cancelled or when ps is closed. It blocks, so that the
// caller decides which goroutine runs it, which is how a producer that hands
// out a receive channel is connected to a PubSub:
//
//	go ps.Forward(ctx, producer.Events())
//
// Everything forwarded through one channel is published in the order it was
// received, and so observed in that order by every subscriber, unlike the
// items of concurrent calls to Publish.
//
// Closing ch does not close ps: a PubSub may be fed by any number of sources
// and only its owner knows when the last of them is finished. Items that
// arrive on ch once ps has been closed are discarded, as they are when
// published directly.
func (ps *PubSub[T]) Forward(ctx context.Context, ch <-chan T) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ps.done:
			return
		case item, ok := <-ch:
			if !ok {
				return
			}
			ps.Publish(item)
		}
	}
}

// Close closes the PubSub instance and all of its active subscribers, and ends
// any Forward that is feeding it. Calling it more than once is harmless.
//
// Each subscriber's channel is closed once the items it has already buffered
// have been received from it, so every subscriber must drain its channel for
// the subscriptions to be released, see Subscribe.
func (ps *PubSub[T]) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return
	}
	ps.closed = true
	close(ps.done)
	for sub := range ps.subscribers {
		close(sub.ch.In())
	}
	ps.subscribers = nil
}
