// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stream

import (
	"context"
	"fmt"
	"iter"
	"sync"
)

const (
	// DefaultBatchSize is the number of items requested from a PopN at a time.
	DefaultBatchSize = 64
	// DefaultPrefetch is the number of batches fetched ahead of their use.
	DefaultPrefetch = 2
)

type options struct {
	batchSize int
	prefetch  int
}

// Option represents an option for NewScanner and Iter.
type Option func(*options)

// WithBatchSize sets the number of items requested from a PopN at a time,
// DefaultBatchSize if n is less than one.
func WithBatchSize(n int) Option {
	return func(o *options) { o.batchSize = n }
}

// WithPrefetch sets the number of batches that are fetched, by a goroutine,
// ahead of the items being consumed so that fetching overlaps with
// consuming. Zero disables prefetching, so that the PopN is only called, from
// the caller's goroutine, when more items are needed. The default is
// DefaultPrefetch.
func WithPrefetch(n int) Option {
	return func(o *options) { o.prefetch = max(n, 0) }
}

// batch is the result of one call to a PopN.
type batch[T any] struct {
	items []T
	err   error
	last  bool // no batch follows this one.
}

// Scanner iterates over the items of a stream, in the manner of bufio.Scanner:
//
//	sc := stream.NewScanner(ctx, popn)
//	defer sc.Close()
//	for sc.Scan() {
//	    item := sc.Data()
//	}
//	if err := sc.Err(); err != nil {
//	    ...
//	}
//
// A Scanner is not safe for concurrent use.
type Scanner[T any] struct {
	popn PopN[T]
	opts options
	ctx  context.Context

	// Used when prefetching.
	cancel  context.CancelFunc
	started bool
	out     chan batch[T]
	free    chan []T
	wg      sync.WaitGroup

	// The batch being consumed, which is cur[pos:], and its backing storage.
	cur     []T
	pos     int
	buf     []T // used when not prefetching.
	pending error
	last    bool // cur is the last batch.
	done    bool
	closed  bool
	data    T
	err     error
}

// NewScanner returns a Scanner for the stream implemented by popn. The
// Scanner, and the goroutine that it uses for prefetching, which is started
// by the first call to Scan, run until the stream is exhausted, ctx is done or
// Close is called. Close must be called if the Scanner is not used to the end.
func NewScanner[T any](ctx context.Context, popn PopN[T], opts ...Option) *Scanner[T] {
	s := &Scanner[T]{popn: popn, opts: options{batchSize: DefaultBatchSize, prefetch: DefaultPrefetch}}
	for _, fn := range opts {
		fn(&s.opts)
	}
	if s.opts.batchSize < 1 {
		s.opts.batchSize = DefaultBatchSize
	}
	s.ctx, s.cancel = context.WithCancel(ctx) //nolint:gosec // cancel is called by finish and Close.
	return s
}

// Scan advances to the next item, which is then available via Data. It returns
// false when there are no more items, either because the stream has been
// exhausted or because of an error, which Err returns.
func (s *Scanner[T]) Scan() bool {
	if s.closed {
		s.done = true
	}
	for s.pos >= len(s.cur) {
		if s.done {
			return false
		}
		s.recycle()
		if s.last {
			s.finish(s.pending)
			return false
		}
		b, ok := s.fetch()
		if !ok {
			// Prefetching ended early, because the context is done.
			s.finish(s.ctx.Err())
			return false
		}
		s.cur, s.pos, s.pending, s.last = b.items, 0, b.err, b.last
	}
	s.data = s.cur[s.pos]
	s.pos++
	return true
}

// Data returns the item found by the most recent call to Scan.
func (s *Scanner[T]) Data() T {
	return s.data
}

// Err returns the error, if any, that ended the scan: that returned by the
// PopN, or the error of the context if it was done. It is nil if the stream
// was exhausted, or the Scanner closed.
func (s *Scanner[T]) Err() error {
	return s.err
}

// Close stops prefetching, waiting for the goroutine that does so to exit,
// and releases the resources of the Scanner. It is safe to call more than once.
// Scan returns false after Close.
func (s *Scanner[T]) Close() error {
	s.closed = true
	s.cancel()
	s.wg.Wait()
	s.cur, s.pos = nil, 0
	return nil
}

func (s *Scanner[T]) finish(err error) {
	s.done = true
	s.cur, s.pos = nil, 0
	if err != nil && !s.closed {
		s.err = err
	}
	s.cancel()
}

// recycle returns the storage of the batch that has been consumed.
func (s *Scanner[T]) recycle() {
	if s.cur == nil {
		return
	}
	clear(s.cur)
	if s.opts.prefetch > 0 {
		// There is always room: the number of buffers is fixed.
		s.free <- s.cur[:cap(s.cur)]
	}
	s.cur = nil
}

// fetch returns the next batch, calling the PopN directly if not prefetching.
func (s *Scanner[T]) fetch() (batch[T], bool) {
	if s.opts.prefetch == 0 {
		if s.buf == nil {
			s.buf = make([]T, s.opts.batchSize)
		}
		if err := s.ctx.Err(); err != nil {
			return batch[T]{err: err, last: true}, true
		}
		return s.call(s.buf), true
	}
	if !s.started {
		s.start()
	}
	b, ok := <-s.out
	return b, ok
}

// call calls the PopN, validating what it returns.
func (s *Scanner[T]) call(buf []T) batch[T] {
	n, empty, err := s.popn(s.ctx, buf)
	if n < 0 || n > len(buf) {
		return batch[T]{err: fmt.Errorf("stream: PopN returned %d items for a request for %d", n, len(buf)), last: true}
	}
	if n == 0 && !empty && err == nil {
		return batch[T]{err: fmt.Errorf("stream: PopN returned 0 items without empty or error"), last: true}
	}
	return batch[T]{items: buf[:n], err: err, last: empty || err != nil}
}

// start starts the goroutine that prefetches batches. The number of
// buffers allows for one being consumed, one being filled and one for each of
// the batches that are queued.
func (s *Scanner[T]) start() {
	s.started = true
	s.out = make(chan batch[T], s.opts.prefetch)
	s.free = make(chan []T, s.opts.prefetch+2)
	for range s.opts.prefetch + 2 {
		s.free <- make([]T, s.opts.batchSize)
	}
	s.wg.Add(1)
	go s.prefetch()
}

func (s *Scanner[T]) prefetch() {
	defer s.wg.Done()
	defer close(s.out)
	for {
		var buf []T
		select {
		case buf = <-s.free:
		case <-s.ctx.Done():
			return
		}
		b := s.call(buf)
		select {
		case s.out <- b:
		case <-s.ctx.Done():
			return
		}
		if b.last {
			return
		}
	}
}

// Iter returns an iterator over the items of the stream implemented by popn,
// as used with range. If the stream ends with an error it is yielded, along
// with the zero value, as the last element. The items are prefetched as per
// NewScanner, and the prefetching is stopped when the iteration ends, whether
// or not it was run to the end.
func Iter[T any](ctx context.Context, popn PopN[T], opts ...Option) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		sc := NewScanner(ctx, popn, opts...)
		defer sc.Close()
		for sc.Scan() {
			if !yield(sc.Data(), nil) {
				return
			}
		}
		if err := sc.Err(); err != nil {
			var zero T
			yield(zero, err)
		}
	}
}
