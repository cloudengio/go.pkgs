// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap_test

import (
	"testing"
	"time"

	"cloudeng.io/algo/container/bitmap"
)

func expectClosed(t *testing.T, cb *bitmap.Contiguous, end int, ch <-chan struct{}) <-chan struct{} {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("expected closed channel, but timed out instead")
	}
	if got, want := cb.Tail(), end; got != want {
		t.Errorf("expect closed = %v, want %v", got, want)
	}
	return cb.Notify()
}

func expectBlocked(t *testing.T, cb *bitmap.Contiguous, end int, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("expected blocked channel, but it was closed")
	default:
		// This is the expected case.
	}
	if got, want := cb.Tail(), end; got != want {
		t.Errorf("expect blocked = %v, want %v", got, want)
	}
}

func TestContiguousEmpty(t *testing.T) {

	contig := bitmap.NewContiguous(20, 23)

	ch := contig.Notify()
	ch1 := contig.Notify()
	if ch != ch1 {
		t.Errorf("expected same channel for Tail() calls, got %v and %v", ch, ch1)
	}

	expectBlocked(t, contig, -1, ch)
	contig.Set(0)
	expectBlocked(t, contig, -1, ch)
	contig.Set(19)
	expectBlocked(t, contig, -1, ch)

	// Don't expect a notification until now.
	contig.Set(20)
	ch = expectClosed(t, contig, 20, ch)

	contig.Set(22)
	expectBlocked(t, contig, 20, ch)
	contig.Set(21)
	expectClosed(t, contig, 22, ch)
	ch = contig.Notify()
	expectClosed(t, contig, 22, ch)
	ch = contig.Notify()
	expectClosed(t, contig, 22, ch)
}

func TestContiguousWithBitmap(t *testing.T) {
	bm := bitmap.New(10)
	bm.Set(0)
	bm.Set(1)
	contig := bitmap.NewContiguousWithBitmap(bm, 5, 10)
	ch := contig.Notify()
	expectBlocked(t, contig, -1, ch)
	contig.Set(4)
	expectBlocked(t, contig, -1, ch)

	// First notification.
	contig.Set(5)
	ch = expectClosed(t, contig, 5, ch)

	contig.Set(6)
	ch = expectClosed(t, contig, 6, ch)
	contig.Set(7)
	ch = expectClosed(t, contig, 7, ch)

	// No notification since there's a hole between 7 and 9.
	contig.Set(9)
	expectBlocked(t, contig, 7, ch)
	contig.Set(8)
	expectClosed(t, contig, 9, ch)

	// Test overlap between initial bitmap and contiguous range.
	bm = bitmap.New(10)
	for i := range 7 {
		bm.Set(i)
	}
	contig = bitmap.NewContiguousWithBitmap(bm, 5, 10)
	tail := contig.Tail()
	if tail != 6 {
		t.Fatalf("expected initial LastSet() to be 6, got %d", tail)
	}
	ch = contig.Notify()
	expectBlocked(t, contig, 6, ch)
	contig.Set(9)
	expectBlocked(t, contig, 6, ch)
	contig.Set(8)
	expectBlocked(t, contig, 6, ch)
	contig.Set(7)
	expectClosed(t, contig, 9, ch)

	bm = bitmap.New(10)
	for i := range 7 {
		bm.Set(i)
	}
	bm.Set(9)
	bm.Set(8)
	contig = bitmap.NewContiguousWithBitmap(bm, 5, 10)
	ch = contig.Notify()
	expectBlocked(t, contig, 6, ch)
	contig.Set(7)
	expectClosed(t, contig, 9, ch)

}

func TestContiguousInitialState(t *testing.T) {
	// Create a bitmap that is already contiguous from the start of the tracked range.
	bm := bitmap.New(20)
	bm.Set(10)
	bm.Set(11)
	// Track from index 10.
	contig := bitmap.NewContiguousWithBitmap(bm, 10, 20)

	// The constructor should have advanced firstClear to 12.
	// LastSet should therefore be 11.
	if got, want := contig.Tail(), 11; got != want {
		t.Fatalf("initial LastSet() = %v, want %v", got, want)
	}

	// The first call to Notify should return a new channel.
	ch := contig.Notify()

	// Since nothing has been set since creation, the channel should be blocked.
	expectBlocked(t, contig, 11, ch)

	// Now, set the next bit in the sequence.
	contig.Set(12)

	// This should have triggered an update and closed the channel.
	// The new LastSet should be 12.
	ch = expectClosed(t, contig, 12, ch)

	// The next channel should be blocked.
	expectBlocked(t, contig, 12, ch)
}

func TestContiguousAlreadyComplete(t *testing.T) {
	bm := bitmap.New(10)
	for i := range 10 {
		bm.Set(i)
	}

	// Track the full range 0..9
	contig := bitmap.NewContiguousWithBitmap(bm, 0, 10)

	// The constructor should have advanced firstClear to 10.
	// LastSet should be 9.
	if got, want := contig.Tail(), 9; got != want {
		t.Fatalf("initial LastSet() = %v, want %v", got, want)
	}

	// Get a channel.
	ch := contig.Notify()

	// Since the range is already complete, firstClear (10) is greater than
	// the last index of the tracked range (last=9). The Notify method will
	// therefore return an already closed channel.
	expectClosed(t, contig, 9, ch)
}
