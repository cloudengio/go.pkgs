// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package patterns provides common synchronization and communication
// patterns built using channels and other primitives.
package patterns

import (
	"context"

	"cloudeng.io/sync/ctxsync"
)

// FIFO is a goroutine-safe queue that drops the oldest item when the internal
// buffer (capacity items) is full. b.out is unbuffered; items are only
// delivered when a receiver is ready.
//
// The internal state (buf, head, tail, count) is a ring buffer accessed
// exclusively by the run goroutine, so drop-oldest is atomic with respect to
// external readers and requires no allocations after the initial make.
type FIFO[T any] struct {
	in     chan T
	out    chan T
	doneCh chan struct{}
	wg     ctxsync.WaitGroup
	size   int
	buf    []T // ring buffer backing array; len=cap=size; safe to read after Stop()
	head   int // index of the oldest item
	tail   int // index where the next item will be written
	count  int // number of items currently buffered
}

const DefaultFIFOSize = 100

// NewFIFO creates a new FIFO with the specified buffer capacity.
// If capacity is <= 0, it defaults to DefaultFIFOSize.
func NewFIFO[T any](ctx context.Context, capacity int) *FIFO[T] {
	return newFIFO[T](ctx, capacity, nil)
}

// newFIFO is the internal constructor. If notify is non-nil it is closed when
// the run goroutine exits, allowing callers to observe liveness without
// coupling that concept to FIFO itself.
func newFIFO[T any](ctx context.Context, capacity int, notify chan struct{}) *FIFO[T] {
	if capacity <= 0 {
		capacity = DefaultFIFOSize
	}
	bf := &FIFO[T]{
		in:     make(chan T),
		out:    make(chan T),
		doneCh: make(chan struct{}),
		size:   capacity,
		buf:    make([]T, capacity), // allocated once; never resized
	}
	bf.wg.Go(func() {
		if notify != nil {
			defer close(notify)
		}
		bf.run(ctx)
	})
	return bf
}

func (b *FIFO[T]) Stop(ctx context.Context) {
	close(b.doneCh)
	b.wg.Wait(ctx)
}

// push adds v at the tail. Caller must ensure count < size.
func (b *FIFO[T]) push(v T) {
	b.buf[b.tail] = v
	b.tail = (b.tail + 1) % b.size
	b.count++
}

// front returns the oldest item without removing it. Caller must ensure count > 0.
func (b *FIFO[T]) front() T {
	return b.buf[b.head]
}

// pop removes the oldest item. Caller must ensure count > 0.
func (b *FIFO[T]) pop() {
	var zero T
	b.buf[b.head] = zero // clear slot so GC can collect pointer-typed items
	b.head = (b.head + 1) % b.size
	b.count--
}

func (b *FIFO[T]) runBuffered(ctx context.Context) bool {
	select {
	case b.out <- b.front():
		b.pop()
		return false
	case v, ok := <-b.in:
		if !ok {
			// b.in closed: flush buffered items in FIFO order then close b.out.
			for b.count > 0 {
				select {
				case b.out <- b.front():
					b.pop()
				case <-b.doneCh:
					return true
				case <-ctx.Done():
					return true
				}
			}
			close(b.out)
			return true
		}
		if b.count == b.size {
			b.pop() // drop oldest to make room
		}
		b.push(v)
		return false
	case <-b.doneCh:
		return true
	case <-ctx.Done():
		return true
	}
}

func (b *FIFO[T]) run(ctx context.Context) {
	for {
		if b.count == 0 {
			select {
			case v, ok := <-b.in:
				if !ok {
					close(b.out)
					return
				}
				b.push(v)
			case <-b.doneCh:
				return
			case <-ctx.Done():
				return
			}
			continue
		}
		if b.runBuffered(ctx) {
			return
		}
	}
}

func (b *FIFO[T]) In() chan<- T {
	return b.in
}

func (b *FIFO[T]) Out() <-chan T {
	return b.out
}
