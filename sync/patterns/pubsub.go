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
func (s *Subscriber[T]) C() <-chan T {
	return s.ch.Out()
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
}

// New returns a new PubSub instance.
func New[T any]() *PubSub[T] {
	return &PubSub[T]{
		subscribers: make(map[*Subscriber[T]]struct{}),
	}
}

// Subscribe creates and returns a new Subscriber with the given buffer capacity.
// If capacity is <=0, it defaults to DefaultPubSubCapacity.
// ctx is passed to the underlying FIFO.
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
// Subscribers whose run goroutine has exited (e.g. context cancelled) are
// detected via their alive channel and pruned from the map without blocking.
func (ps *PubSub[T]) Publish(item T) {
	ps.mu.RLock()
	if ps.closed {
		ps.mu.RUnlock()
		return
	}
	var dead []*Subscriber[T]
	for sub := range ps.subscribers {
		select {
		case sub.ch.in <- item:
		case <-sub.alive:
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
