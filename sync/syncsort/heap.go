// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package syncsort provides support for synchronised sorting.
package syncsort

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ErrStopped is returned by NextItem once the sequencer has been stopped
// via Stop or its ordering goroutine has exited, including at the end of
// the input stream.
var ErrStopped = errors.New("sequencer stopped")

type options struct {
	maxBuffered int
}

// Option represents an option to NewSequencer.
type Option func(*options)

// WithMaxBuffered limits the maximum number of unconsumed items (those
// obtained via NextItem and not yet yielded by Scan) to n. When n <= 0,
// buffering is unbounded.
//
// When a positive limit is set, NextItem blocks if n unconsumed items are
// already in flight until an item is consumed via Scan, the context is
// cancelled, or Stop is called. Callers using WithMaxBuffered must therefore
// consume items via Scan concurrently with producers calling NextItem.
// Because the internal ordering goroutine continues to drain inputCh regardless
// of arrival order, out-of-order arrival among in-flight items will not deadlock.
func WithMaxBuffered(n int) Option {
	return func(o *options) {
		o.maxBuffered = n
	}
}

// Sequencer implements a streaming sequencer that will accept a stream
// of unordered items (sent to it over a channel) and allow for that stream
// to be scanned in order. The end of the unordered stream is signaled by
// closing this channel. Items to be sent in the stream are obtained via calls to
// NextItem and the order of calls to NextItem determines the order of
// items returned by the scanner.
type Sequencer[T any] struct {
	next atomic.Uint64
	// ctx bounds the lifetime of the sequencer and is retained so that
	// Scan and NextItem can observe its cancellation.
	ctx      context.Context
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
	inFlight chan struct{} // nil when buffering is unbounded.
	inputCh  <-chan Item[T]
	scanCh   chan T
	heap     streamingHeap[T]
	asyncErr error
	err      error
	v        T
}

// Item represents a single item in a stream that is to be ordered. It is
// returned by NextItem and simply wraps the supplied type with a monotonically
// increasing sequence number that determines its position in the ordered
// stream. This sequence number is assigned by NextItem.
type Item[T any] struct {
	V T
	s uint64
}

// NewSequencer returns a new instance of Sequencer.
func NewSequencer[T any](ctx context.Context, inputCh <-chan Item[T], opts ...Option) *Sequencer[T] {
	var o options
	for _, fn := range opts {
		fn(&o)
	}
	heapCap := 100
	var inFlight chan struct{}
	if o.maxBuffered > 0 {
		inFlight = make(chan struct{}, o.maxBuffered)
		heapCap = min(o.maxBuffered, 100)
	}
	seq := &Sequencer[T]{
		ctx:      ctx,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		inFlight: inFlight,
		inputCh:  inputCh,
		heap:     make(streamingHeap[T], 0, heapCap),
		scanCh:   make(chan T, 1),
	}
	heap.Init(&seq.heap)
	go seq.order()
	return seq
}

// Stop terminates the sequencer early so its internal goroutine exits
// without requiring the consumer to drain remaining items from Scan and
// without cancelling the context passed to NewSequencer. Any calls waiting in
// NextItem return ErrStopped, and Done is closed. Stop is safe to call
// multiple times.
func (s *Sequencer[T]) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

// Done returns a channel that is closed when the internal ordering goroutine
// exits (due to Stop, context cancellation, or end of stream). Producers can
// select on Done when sending items to the input channel so they do not block
// if the consumer stops early.
func (s *Sequencer[T]) Done() <-chan struct{} {
	return s.doneCh
}

// stopErr returns the reason NextItem cannot proceed, or nil if it can.
// Stop takes precedence over context cancellation.
func (s *Sequencer[T]) stopErr(ctx context.Context) error {
	select {
	case <-s.stopCh:
		return ErrStopped
	default:
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-s.doneCh:
		return ErrStopped
	default:
	}
	return nil
}

// stopped reports whether the ordering goroutine should exit, recording
// any context error in asyncErr.
func (s *Sequencer[T]) stopped() bool {
	select {
	case <-s.stopCh:
		return true
	default:
	}
	if err := s.ctx.Err(); err != nil {
		s.asyncErr = err
		return true
	}
	return false
}

// NextItem returns a new Item to be used with Sequencer. The order of calls
// made to NextItem determines the order that they are returned by the scanner.
// When WithMaxBuffered is configured, NextItem blocks until a buffer slot is
// available, ctx or the sequencer's context is cancelled, or Stop is called.
// ctx must not be nil.
func (s *Sequencer[T]) NextItem(ctx context.Context, item T) (Item[T], error) {
	if err := s.stopErr(ctx); err != nil {
		return Item[T]{}, err
	}
	if s.inFlight != nil {
		// Stop and cancellation of the sequencer's context both cause
		// doneCh to be closed.
		select {
		case s.inFlight <- struct{}{}:
		case <-s.doneCh:
			return Item[T]{}, s.stopErr(ctx)
		case <-ctx.Done():
			return Item[T]{}, s.stopErr(ctx)
		}
	}
	return Item[T]{
		V: item,
		s: s.next.Add(1),
	}, nil
}

// Scan returns true if the next ordered item in the stream is available to
// be read via Item. On cancellation, Stop, or end of stream, Scan returns false
// without yielding a zero item; call Err to inspect any error.
func (s *Sequencer[T]) Scan() bool {
	if s.err != nil {
		return false
	}
	if err := s.ctx.Err(); err != nil {
		s.err = err
		return false
	}
	select {
	case <-s.stopCh:
		return false
	default:
	}
	select {
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		return false
	case <-s.stopCh:
		return false
	case v, ok := <-s.scanCh:
		if !ok {
			s.err = s.asyncErr
			return false
		}
		if s.inFlight != nil {
			<-s.inFlight
		}
		s.v = v
		return true
	}
}

// Item returns the current item available in the scanner.
func (s *Sequencer[T]) Item() T {
	return s.v
}

// Err returns any errors encountered by the scanner.
func (s *Sequencer[T]) Err() error {
	return s.err
}

func (s *Sequencer[T]) flushHeap(expected *uint64) bool {
	for len(s.heap) > 0 && s.heap[0].s == *expected {
		if s.stopped() {
			return false
		}
		item := heap.Remove(&s.heap, 0).(Item[T])
		*expected++
		select {
		case s.scanCh <- item.V:
		case <-s.ctx.Done():
			s.asyncErr = s.ctx.Err()
			return false
		case <-s.stopCh:
			return false
		}
	}
	return true
}

func (s *Sequencer[T]) order() {
	defer close(s.doneCh)
	defer close(s.scanCh)
	expected := uint64(1)
	for {
		if s.stopped() {
			return
		}
		select {
		case <-s.ctx.Done():
			s.asyncErr = s.ctx.Err()
			return
		case <-s.stopCh:
			return
		case item, ok := <-s.inputCh:
			if ok {
				heap.Push(&s.heap, item)
			}
			if !s.flushHeap(&expected) {
				return
			}
			if !ok {
				if len(s.heap) > 0 {
					s.asyncErr = fmt.Errorf("syncsort: input channel closed before receiving expected sequence number %d (%d items remaining)", expected, len(s.heap))
				}
				return
			}
		}
	}
}

// streamingHeap implements heap.Interface.
type streamingHeap[T any] []Item[T]

func (s *streamingHeap[T]) Len() int           { return len(*s) }
func (s *streamingHeap[T]) Less(i, j int) bool { return (*s)[i].s < (*s)[j].s }
func (s *streamingHeap[T]) Swap(i, j int)      { (*s)[i], (*s)[j] = (*s)[j], (*s)[i] }

// Push and Pop use pointer receivers because they modify the slice's length,
// not just its contents.
func (s *streamingHeap[T]) Push(x any) {
	*s = append(*s, x.(Item[T]))
}

func (s *streamingHeap[T]) Pop() any {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}
