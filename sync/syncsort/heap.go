// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package syncsort provides support for synchronised sorting.
package syncsort

import (
	"container/heap"
	"context"
	"sync/atomic"
)

// Sequencer implements a streaming sequencer that will accept a stream
// of unordered items (sent to it over a channel) and allow for that stream
// to be scanned in order. The end of the unordered stream is signaled by
// closing this chanel. Items to be sent in the stream are obtained via calls to
// NextItem and the order of calls to NextItem determines the order of
// items returned by the scanner.
type Sequencer[T any] struct {
	next    uint64
	inputCh <-chan Item[T]
	scanCh  chan nextItem[T]
	heap    streamingHeap[T]
	err     error
	v       T
}

// Item represents a single item in a stream that is to be ordered. It is
// returned by NextItem and simply wraps the supplied type with a monotonically
// increasing sequence number that determines its position in the ordered
// stream. This sequence number is assigned by NextItem.
type Item[T any] struct {
	V T
	s uint64
}

type nextItem[T any] struct {
	v   T
	err error
}

// NewSequencer returns a new instance of Sequencer.
func NewSequencer[T any](ctx context.Context, inputCh <-chan Item[T]) *Sequencer[T] {
	seq := &Sequencer[T]{
		inputCh: inputCh,
		heap:    make(streamingHeap[T], 0, 100),
		scanCh:  make(chan nextItem[T], 1),
	}
	heap.Init(&seq.heap)
	go seq.order(ctx)
	return seq
}

// NextItem returns a new Item to be used with Sequencer. The order of calls
// made to NextItem determines the order that they are returned by the scanner.
func (s *Sequencer[T]) NextItem(item T) Item[T] {
	return Item[T]{
		V: item,
		s: atomic.AddUint64(&s.next, 1),
	}
}

// Scan returns true of the next ordered item in the stream is available to
// be reead.
func (s *Sequencer[T]) Scan() bool {
	if s.err != nil {
		return false
	}
	ni, ok := <-s.scanCh
	if !ok {
		return false
	}
	s.err = ni.err
	s.v = ni.v
	return true
}

// Item returns the current item available in the scanner.
func (s *Sequencer[T]) Item() T {
	return s.v
}

// Err returns any errors encountered by the scanner.
func (s *Sequencer[T]) Err() error {
	return s.err
}

func (s *Sequencer[T]) order(ctx context.Context) {
	expected := uint64(1)
	for {
		select {
		case <-ctx.Done():
			s.scanCh <- nextItem[T]{err: ctx.Err()}
			return
		case item, ok := <-s.inputCh:
			if ok {
				heap.Push(&s.heap, item)
			}
			for len(s.heap) > 0 {
				min := (s.heap)[0]
				if min.s != expected {
					break
				}
				item := heap.Remove(&s.heap, 0).(Item[T])
				expected++
				s.scanCh <- nextItem[T]{v: item.V}
			}
			if !ok && len(s.heap) == 0 {
				close(s.scanCh)
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

func (s *streamingHeap[T]) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}
