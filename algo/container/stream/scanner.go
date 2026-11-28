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
// ahead of the batch being consumed, including one that is being fetched, so
// that fetching overlaps with consuming. Zero disables prefetching, so that
// the PopN is only called, from the caller's goroutine, when more items are
// needed. The default is DefaultPrefetch.
func WithPrefetch(n int) Option {
	return func(o *options) { o.prefetch = max(n, 0) }
}

// batch is the result of one call to a PopN.
type batch[T any] struct {
	items []T
	err   error
	last  bool // no batch follows this one.
}

// request asks the prefetching goroutine to fill buf, using ctx.
type request[T any] struct {
	ctx context.Context
	buf []T
}

// Scanner iterates over the items of a stream, in the manner of bufio.Scanner:
//
//	sc := stream.NewScanner(popn)
//	defer sc.Close()
//	for sc.Scan(ctx) {
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

	// Used when prefetching.
	quit    context.Context
	cancel  context.CancelFunc
	started bool
	reqs    chan request[T]
	out     chan batch[T]
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
// goroutine that the Scanner uses for prefetching is started by the first
// call to Scan, and runs until the stream is exhausted or Close is called.
// Close must be called if the Scanner is not used to the end.
func NewScanner[T any](popn PopN[T], opts ...Option) *Scanner[T] {
	s := &Scanner[T]{popn: popn, opts: options{batchSize: DefaultBatchSize, prefetch: DefaultPrefetch}}
	for _, fn := range opts {
		fn(&s.opts)
	}
	if s.opts.batchSize < 1 {
		s.opts.batchSize = DefaultBatchSize
	}
	s.quit, s.cancel = context.WithCancel(context.Background()) //nolint:gosec // cancel is called by finish and Close.
	return s
}

// Scan advances to the next item, which is then available via Data. It returns
// false when there are no more items, either because the stream has been
// exhausted or because of an error, which Err returns.
//
// ctx is passed to each call to the PopN that this call to Scan initiates,
// which, when prefetching, is a call for a batch that will be consumed by a
// later Scan. It must therefore remain valid for as long as the Scanner is
// used; if it is done the scan ends with its error.
func (s *Scanner[T]) Scan(ctx context.Context) bool {
	if s.closed {
		s.done = true
	}
	for s.pos >= len(s.cur) {
		if s.done {
			return false
		}
		s.recycle(ctx)
		if s.last {
			s.finish(s.pending)
			return false
		}
		b, ok := s.fetch(ctx)
		if !ok {
			// The context is done.
			s.finish(ctx.Err())
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

// recycle takes the storage of the batch that has been consumed and, when
// prefetching, uses it to request another batch unless that was the last.
func (s *Scanner[T]) recycle(ctx context.Context) {
	if s.cur == nil {
		return
	}
	clear(s.cur)
	if s.opts.prefetch > 0 && !s.last {
		// There is always room: the number of buffers is fixed.
		s.reqs <- request[T]{ctx, s.cur[:cap(s.cur)]}
	}
	s.cur = nil
}

// fetch returns the next batch, calling the PopN directly if not prefetching.
func (s *Scanner[T]) fetch(ctx context.Context) (batch[T], bool) {
	if s.opts.prefetch == 0 {
		if s.buf == nil {
			s.buf = make([]T, s.opts.batchSize)
		}
		if err := ctx.Err(); err != nil {
			return batch[T]{err: err, last: true}, true
		}
		return s.call(ctx, s.buf), true
	}
	if !s.started {
		s.start(ctx)
	}
	select {
	case b, ok := <-s.out:
		return b, ok
	case <-ctx.Done():
		return batch[T]{}, false
	}
}

// call calls the PopN, validating what it returns.
func (s *Scanner[T]) call(ctx context.Context, buf []T) batch[T] {
	n, empty, err := s.popn(ctx, buf)
	if n < 0 || n > len(buf) {
		return batch[T]{err: fmt.Errorf("stream: PopN returned %d items for a request for %d", n, len(buf)), last: true}
	}
	if n == 0 && !empty && err == nil {
		return batch[T]{err: fmt.Errorf("stream: PopN returned 0 items without empty or error"), last: true}
	}
	return batch[T]{items: buf[:n], err: err, last: empty || err != nil}
}

// start starts the goroutine that prefetches batches and requests the
// first of them. The number of buffers allows for one being consumed and
// prefetch that are queued, being filled or filled, so that the goroutine
// never blocks sending the results. All are requested at once: the first is
// for the batch about to be consumed and the rest are the prefetch.
func (s *Scanner[T]) start(ctx context.Context) {
	s.started = true
	buffers := s.opts.prefetch + 1
	s.reqs = make(chan request[T], buffers)
	s.out = make(chan batch[T], buffers)
	for range buffers {
		s.reqs <- request[T]{ctx, make([]T, s.opts.batchSize)}
	}
	s.wg.Add(1)
	go s.prefetch()
}

func (s *Scanner[T]) prefetch() {
	defer s.wg.Done()
	defer close(s.out)
	for {
		var r request[T]
		select {
		case r = <-s.reqs:
		case <-s.quit.Done():
			return
		}
		// The PopN is stopped by either the context of the request or Close.
		ctx, cancel := context.WithCancel(r.ctx)
		stop := context.AfterFunc(s.quit, cancel)
		b := s.call(ctx, r.buf)
		stop()
		cancel()
		s.out <- b
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
		sc := NewScanner(popn, opts...)
		defer sc.Close()
		for sc.Scan(ctx) {
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
