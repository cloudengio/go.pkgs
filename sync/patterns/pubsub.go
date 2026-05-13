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
	ch *FIFO[T]
}

// C returns the underlying receive-only channel for use in select statements.
func (s *Subscriber[T]) C() <-chan T {
	return s.ch.Out()
}

// PubSub provides a concurrent pub-sub mechanism that drops the oldest
// items for slow subscribers when their buffer is full.
type PubSub[T any] struct {
	mu          sync.RWMutex
	subscribers map[*Subscriber[T]]struct{}
	capacity    int
	closed      bool
}

// New returns a new PubSub instance with the given buffer capacity for
// each subscriber. capacity must be > 0.
func New[T any](capacity int) *PubSub[T] {
	return &PubSub[T]{
		subscribers: make(map[*Subscriber[T]]struct{}),
		capacity:    capacity,
	}
}

// Subscribe creates and returns a new Subscriber. ctx is passed to the underlying FIFO.
func (ps *PubSub[T]) Subscribe(ctx context.Context) *Subscriber[T] {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	sub := &Subscriber[T]{
		ch: NewFIFO[T](ctx, ps.capacity),
	}
	if ps.closed {
		close(sub.ch.In())
		return sub
	}
	ps.subscribers[sub] = struct{}{}
	return sub
}

// Unsubscribe removes a subscriber and closes its underlying channel.
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
func (ps *PubSub[T]) Publish(item T) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.closed {
		return
	}
	for sub := range ps.subscribers {
		sub.ch.In() <- item
	}
}

// Close closes the PubSub instance and all of its active subscribers.
func (ps *PubSub[T]) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return
	}
	ps.closed = true
	for sub := range ps.subscribers {
		close(sub.ch.In())
	}
	ps.subscribers = nil
}
