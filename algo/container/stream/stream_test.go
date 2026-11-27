// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stream_test

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/algo/container/stream"
)

// counting returns a PopN for the integers [0, total), which fails with
// failWith, after returning any items that it had, once failAt items have
// been returned if failWith is not nil, and which records how many calls it
// has had and the largest request.
type counting struct {
	total, failAt int
	failWith      error
	next          int
	calls         atomic.Int64
	maxRequest    atomic.Int64
}

func (c *counting) popn(ctx context.Context, items []int) (int, bool, error) {
	c.calls.Add(1)
	if int64(len(items)) > c.maxRequest.Load() {
		c.maxRequest.Store(int64(len(items)))
	}
	if err := ctx.Err(); err != nil {
		return 0, false, err
	}
	n := 0
	for n < len(items) && c.next < c.total {
		if c.failWith != nil && c.next == c.failAt {
			return n, false, c.failWith
		}
		items[n] = c.next
		c.next++
		n++
	}
	return n, c.next >= c.total, nil
}

func seq(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func scanAll(sc *stream.Scanner[int]) []int {
	var got []int
	for sc.Scan() {
		got = append(got, sc.Data())
	}
	return got
}

func TestScanner(t *testing.T) {
	ctx := t.Context()
	for _, total := range []int{0, 1, 5, 63, 64, 65, 1000} {
		for _, prefetch := range []int{0, 1, 2, 5} {
			for _, batch := range []int{1, 3, 64} {
				c := &counting{total: total}
				sc := stream.NewScanner(ctx, c.popn, stream.WithPrefetch(prefetch), stream.WithBatchSize(batch))
				got := scanAll(sc)
				if err := sc.Err(); err != nil {
					t.Errorf("total %v, prefetch %v, batch %v: %v", total, prefetch, batch, err)
				}
				if !slices.Equal(got, seq(total)) {
					t.Errorf("total %v, prefetch %v, batch %v: got %v items, want %v", total, prefetch, batch, len(got), total)
				}
				if sc.Scan() {
					t.Errorf("Scan returned true after the end")
				}
				sc.Close()
				if got := int(c.maxRequest.Load()); total > 0 && got != batch {
					t.Errorf("batch %v: requests were for up to %v items", batch, got)
				}
			}
		}
	}
}

func TestScannerDefaults(t *testing.T) {
	c := &counting{total: 1000}
	sc := stream.NewScanner(t.Context(), c.popn, stream.WithBatchSize(0))
	defer sc.Close()
	if got := scanAll(sc); !slices.Equal(got, seq(1000)) {
		t.Errorf("got %v items", len(got))
	}
	if got, want := int(c.maxRequest.Load()), stream.DefaultBatchSize; got != want {
		t.Errorf("got batch size %v, want %v", got, want)
	}
}

// TestScannerError checks that the items that came before an error are
// delivered, and then the error is reported.
func TestScannerError(t *testing.T) {
	errFail := errors.New("failed")
	for _, prefetch := range []int{0, 2} {
		c := &counting{total: 100, failAt: 10, failWith: errFail}
		sc := stream.NewScanner(t.Context(), c.popn, stream.WithPrefetch(prefetch), stream.WithBatchSize(4))
		got := scanAll(sc)
		if !slices.Equal(got, seq(10)) {
			t.Errorf("prefetch %v: got %v, want the ten items before the error", prefetch, got)
		}
		if err := sc.Err(); !errors.Is(err, errFail) {
			t.Errorf("prefetch %v: got %v, want %v", prefetch, err, errFail)
		}
		sc.Close()
		if err := sc.Err(); !errors.Is(err, errFail) {
			t.Errorf("prefetch %v: Close lost the error: %v", prefetch, err)
		}
	}
}

func TestScannerInvalidCount(t *testing.T) {
	for _, n := range []int{-1, 0, 5} {
		popn := func(context.Context, []int) (int, bool, error) { return n, false, nil }
		sc := stream.NewScanner(t.Context(), popn, stream.WithBatchSize(4))
		if sc.Scan() {
			t.Errorf("n %v: unexpected item", n)
		}
		if sc.Err() == nil {
			t.Errorf("n %v: expected an error", n)
		}
		sc.Close()
	}
}

func TestScannerContext(t *testing.T) {
	for _, prefetch := range []int{0, 2} {
		ctx, cancel := context.WithCancel(t.Context())
		c := &counting{total: 1000000}
		sc := stream.NewScanner(ctx, c.popn, stream.WithPrefetch(prefetch), stream.WithBatchSize(8))
		for range 20 {
			if !sc.Scan() {
				t.Fatalf("prefetch %v: scan failed: %v", prefetch, sc.Err())
			}
		}
		cancel()
		// Whatever was fetched may still be delivered, but the scan ends.
		for sc.Scan() {
		}
		if err := sc.Err(); !errors.Is(err, context.Canceled) {
			t.Errorf("prefetch %v: got %v, want %v", prefetch, err, context.Canceled)
		}
		sc.Close()
	}
}

// TestScannerClose checks that Close stops a prefetch that is blocked, and
// that it is not an error.
func TestScannerClose(t *testing.T) {
	started := make(chan struct{})
	exited := make(chan struct{})
	var once atomic.Bool
	popn := func(ctx context.Context, items []int) (int, bool, error) {
		if once.CompareAndSwap(false, true) {
			items[0] = 1
			return 1, false, nil
		}
		close(started)
		<-ctx.Done()
		close(exited)
		return 0, false, ctx.Err()
	}
	sc := stream.NewScanner(t.Context(), popn, stream.WithPrefetch(1), stream.WithBatchSize(2))
	if !sc.Scan() || sc.Data() != 1 {
		t.Fatalf("expected the first item")
	}
	<-started
	if err := sc.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	select {
	case <-exited:
	default:
		t.Errorf("Close returned before the prefetch goroutine exited")
	}
	if sc.Scan() {
		t.Errorf("Scan returned true after Close")
	}
	if err := sc.Err(); err != nil {
		t.Errorf("got %v after Close, want no error", err)
	}
	sc.Close()
}

func TestScannerCloseUnused(t *testing.T) {
	c := &counting{total: 10}
	sc := stream.NewScanner(t.Context(), c.popn)
	sc.Close()
	if sc.Scan() {
		t.Errorf("Scan returned true after Close")
	}
	if got := c.calls.Load(); got != 0 {
		t.Errorf("got %v calls, want none", got)
	}
}

// TestPrefetch checks that batches are fetched ahead of being consumed.
func TestPrefetch(t *testing.T) {
	c := &counting{total: 1000}
	sc := stream.NewScanner(t.Context(), c.popn, stream.WithPrefetch(3), stream.WithBatchSize(10))
	defer sc.Close()
	if !sc.Scan() {
		t.Fatal(sc.Err())
	}
	// One batch is being consumed, and up to three more, and another that is
	// being filled, may be ahead of it.
	deadline := time.Now().Add(10 * time.Second)
	for c.calls.Load() < 4 {
		if time.Now().After(deadline) {
			t.Fatalf("got %v calls, want at least 4 without consuming", c.calls.Load())
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	if got := c.calls.Load(); got > 5 {
		t.Errorf("got %v calls, prefetching is not bounded", got)
	}
}

// TestNoPrefetch checks that without prefetching the PopN is only called when
// more items are needed.
func TestNoPrefetch(t *testing.T) {
	c := &counting{total: 1000}
	sc := stream.NewScanner(t.Context(), c.popn, stream.WithPrefetch(0), stream.WithBatchSize(10))
	defer sc.Close()
	for range 10 {
		sc.Scan()
	}
	time.Sleep(20 * time.Millisecond)
	if got := c.calls.Load(); got != 1 {
		t.Errorf("got %v calls, want 1", got)
	}
	sc.Scan()
	if got := c.calls.Load(); got != 2 {
		t.Errorf("got %v calls, want 2", got)
	}
}

func TestPopNFrom(t *testing.T) {
	next := 0
	errFail := errors.New("failed")
	pop := func(context.Context) (int, bool, error) {
		switch next {
		case 7:
			return 0, true, nil
		case 20:
			return 0, false, errFail
		}
		next++
		return next - 1, false, nil
	}
	popn := stream.PopNFrom(pop)
	buf := make([]int, 4)
	for _, want := range []struct {
		n     int
		empty bool
	}{{4, false}, {3, true}} {
		n, empty, err := popn(t.Context(), buf)
		if err != nil || n != want.n || empty != want.empty {
			t.Errorf("got %v, %v, %v, want %+v", n, empty, err, want)
		}
	}
	if got := buf[:3]; !slices.Equal(got, []int{4, 5, 6}) {
		t.Errorf("got %v", got)
	}

	next = 18
	n, empty, err := popn(t.Context(), buf)
	if n != 2 || empty || !errors.Is(err, errFail) {
		t.Errorf("got %v, %v, %v", n, empty, err)
	}

	// A scanner over a Pop.
	next = 0
	sc := stream.NewScanner(t.Context(), popn)
	defer sc.Close()
	if got := scanAll(sc); !slices.Equal(got, seq(7)) {
		t.Errorf("got %v", got)
	}
}

func TestIter(t *testing.T) {
	for _, prefetch := range []int{0, 2} {
		c := &counting{total: 100}
		var got []int
		for v, err := range stream.Iter(t.Context(), c.popn, stream.WithPrefetch(prefetch), stream.WithBatchSize(7)) {
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, v)
		}
		if !slices.Equal(got, seq(100)) {
			t.Errorf("prefetch %v: got %v", prefetch, got)
		}
	}
}

func TestIterError(t *testing.T) {
	errFail := errors.New("failed")
	c := &counting{total: 100, failAt: 5, failWith: errFail}
	var got []int
	var gotErr error
	for v, err := range stream.Iter(t.Context(), c.popn, stream.WithBatchSize(2)) {
		if err != nil {
			gotErr = err
			continue
		}
		got = append(got, v)
	}
	if !slices.Equal(got, seq(5)) || !errors.Is(gotErr, errFail) {
		t.Errorf("got %v, %v", got, gotErr)
	}
}

// TestIterBreak checks that leaving the iteration early stops the prefetching,
// and waits for it to stop.
func TestIterBreak(t *testing.T) {
	blocked := make(chan struct{})
	exited := make(chan struct{})
	var calls atomic.Int64
	popn := func(ctx context.Context, items []int) (int, bool, error) {
		if calls.Add(1) == 1 {
			items[0] = 1
			return 1, false, nil
		}
		close(blocked)
		<-ctx.Done()
		close(exited)
		return 0, false, ctx.Err()
	}
	for v, err := range stream.Iter(t.Context(), popn, stream.WithPrefetch(1), stream.WithBatchSize(2)) {
		if err != nil || v != 1 {
			t.Fatalf("got %v, %v", v, err)
		}
		<-blocked
		break
	}
	select {
	case <-exited:
	default:
		t.Errorf("the iteration returned before the prefetching stopped")
	}
}
