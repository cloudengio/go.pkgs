// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap_test

import (
	"slices"
	"testing"

	"cloudeng.io/algo/container/bitmap"
)

func goReadall(ch <-chan int, res chan []int) {
	go func() {
		var values []int
		for val := range ch {
			values = append(values, val)
		}
		res <- values
		close(res)
	}()
}

func iseq(a ...int) []int {
	return a
}

func lastSet(t *testing.T, cb *bitmap.Contiguous, end int) {
	t.Helper()
	if got, want := cb.LastSet(), end; got != want {
		t.Errorf("LastSet() = %v, want %v", got, want)
	}
}

func TestContiguousEmpty(t *testing.T) {
	ch := make(chan int, 1)
	resCh := make(chan []int)
	contig := bitmap.NewContiguous(20, 23)
	contig.SetCh(ch) // Set the channel to receive updates.
	lastSet(t, contig, -1)

	goReadall(ch, resCh)

	contig.Set(0)
	lastSet(t, contig, -1)
	contig.Set(19)
	lastSet(t, contig, -1)
	contig.Set(20)
	lastSet(t, contig, 20)
	contig.Set(22)
	lastSet(t, contig, 20)
	contig.Set(21)
	lastSet(t, contig, 22)

	if got, want := <-resCh, iseq(20, 22); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContiguousWithBitmap(t *testing.T) {
	ch := make(chan int, 1)
	resCh := make(chan []int)
	bm := bitmap.New(10)
	bm.Set(0)
	bm.Set(1)
	contig := bitmap.NewContiguousWithBitmap(bm, 5, 10)
	contig.SetCh(ch)
	goReadall(ch, resCh)
	lastSet(t, contig, -1)
	contig.Set(4)
	lastSet(t, contig, -1)
	contig.Set(5)
	lastSet(t, contig, 5)
	contig.Set(6)
	lastSet(t, contig, 6)
	contig.Set(7)
	lastSet(t, contig, 7)
	contig.Set(9)
	lastSet(t, contig, 7)
	contig.Set(8)
	lastSet(t, contig, 9)

	if got, want := <-resCh, iseq(5, 6, 7, 9); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	ch = make(chan int, 1)
	resCh = make(chan []int)
	bm = bitmap.New(10)
	for i := range 7 {
		bm.Set(i)
	}
	contig = bitmap.NewContiguousWithBitmap(bm, 5, 10)
	contig.SetCh(ch)
	goReadall(ch, resCh)
	lastSet(t, contig, 6)
	contig.Set(9)
	lastSet(t, contig, 6)
	contig.Set(8)
	lastSet(t, contig, 6)
	contig.Set(7)
	contig.Set(6)
	lastSet(t, contig, 9)

	if got, want := <-resCh, iseq(6, 9); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContiguousInvalidCreation(t *testing.T) {
	bm := bitmap.New(100)

	testCases := []struct {
		name string
		c    *bitmap.Contiguous
	}{
		{"NewContiguous: zero size", bitmap.NewContiguous(0, 0)},
		{"NewContiguous: negative start", bitmap.NewContiguous(-1, 10)},
		{"NewContiguous: start > size", bitmap.NewContiguous(11, 10)},
		{"NewContiguousWithBitmap: nil bitmap", bitmap.NewContiguousWithBitmap(nil, 0, 10)},
		{"NewContiguousWithBitmap: size > bitmap capacity", bitmap.NewContiguousWithBitmap(bm, 0, 100*64)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.c != nil {
				t.Errorf("expected nil but got a valid instance")
			}
		})
	}
}

func TestContiguousCompletion(t *testing.T) {
	ch := make(chan int, 1)
	resCh := make(chan []int)
	contig := bitmap.NewContiguous(0, 3) // Range is 0, 1, 2.
	contig.SetCh(ch)                     // Set the channel to receive updates.

	goReadall(ch, resCh)

	contig.Set(0)
	lastSet(t, contig, 0)

	contig.Set(1)
	lastSet(t, contig, 1)

	contig.Set(2) // This should fill the range and close the channel.
	lastSet(t, contig, 2)

	// The channel should be closed, and the reader goroutine should finish.
	if got, want := <-resCh, iseq(0, 1, 2); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Setting again should have no effect.
	contig.Set(1)
	lastSet(t, contig, 2)
}

func TestContiguousExtendStopsAtGap(t *testing.T) {
	ch := make(chan int, 1)
	resCh := make(chan []int)
	contig := bitmap.NewContiguous(0, 5) // Tracks range 0..4
	contig.SetCh(ch)                     // Set the channel to receive updates.

	goReadall(ch, resCh)

	// Set bit 2, creating a gap at 0 and 1.
	contig.Set(2)
	lastSet(t, contig, -1) // No change, since firstClear is 0.

	// Set bit 0. This should trigger extend, which should stop at the gap (bit 1).
	contig.Set(0)
	lastSet(t, contig, 0) // firstClear should now be 1.

	// Set bit 3. This is after the gap, so it should not advance firstClear.
	contig.Set(3)
	lastSet(t, contig, 0) // No change.

	// Now, fill the gap by setting bit 1.
	// This should trigger extend, which will now advance past 1, 2, and 3.
	contig.Set(1)
	lastSet(t, contig, 3) // firstClear should now be 4.

	// Close the channel to signal the reader goroutine to finish.
	close(ch)

	// Verify the sequence of updates received on the channel.
	if got, want := <-resCh, iseq(0, 3); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContiguousDelayedSetCh(t *testing.T) {
	ch := make(chan int, 1)
	resCh := make(chan []int)
	contig := bitmap.NewContiguous(0, 5) // Tracks range 0..4
	goReadall(ch, resCh)

	contig.Set(0) // Set a bit before setting the channel.
	contig.Set(1)
	lastSet(t, contig, 1) // Should not trigger an update since channel is
	contig.SetCh(ch)      // Set the channel to receive updates.
	contig.Set(4)
	contig.Set(3)
	contig.Set(2)         // This should fill the range and trigger an update.
	lastSet(t, contig, 4) // Should not trigger an update since 5 is out of range.
	// Verify the sequence of updates received on the channel.
	if got, want := <-resCh, iseq(1, 4); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}
