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
// buffer (size items) is full. b.out is unbuffered; items are only delivered
// when a receiver is ready. The internal []T slice is accessed exclusively by
// the run goroutine, so the drop-oldest step never races with external readers.
type FIFO[T any] struct {
	in     chan T
	out    chan T
	doneCh chan struct{}
	wg     ctxsync.WaitGroup
	size   int
}

func NewFIFO[T any](ctx context.Context, size int) *FIFO[T] {
	bf := &FIFO[T]{
		in:     make(chan T),
		out:    make(chan T),
		doneCh: make(chan struct{}),
		size:   size,
	}
	bf.wg.Go(func() { bf.run(ctx) })
	return bf
}

func (b *FIFO[T]) Stop(ctx context.Context) {
	close(b.doneCh)
	b.wg.Wait(ctx)
}

func (b *FIFO[T]) run(ctx context.Context) {
	buf := make([]T, 0, b.size)
	for {
		if len(buf) == 0 {
			// Nothing buffered: block until an item arrives or we're done.
			select {
			case v, ok := <-b.in:
				if !ok {
					close(b.out)
					return
				}
				buf = append(buf, v)
			case <-b.doneCh:
				return
			case <-ctx.Done():
				return
			}
		} else {
			// Items buffered: try to deliver the front or accept a new item.
			select {
			case b.out <- buf[0]:
				buf = buf[1:]
			case v, ok := <-b.in:
				if !ok {
					// Flush remaining items then signal EOF.
					for _, item := range buf {
						select {
						case b.out <- item:
						case <-b.doneCh:
							return
						case <-ctx.Done():
							return
						}
					}
					close(b.out)
					return
				}
				if len(buf) >= b.size {
					buf = buf[1:] // drop oldest
				}
				buf = append(buf, v)
			case <-b.doneCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}
}

func (b *FIFO[T]) In() chan<- T {
	return b.in
}

func (b *FIFO[T]) Out() <-chan T {
	return b.out
}
