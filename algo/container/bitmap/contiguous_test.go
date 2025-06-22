// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap_test

import (
	"fmt"
	"testing"
	"time"

	"cloudeng.io/algo/container/bitmap"
)

// GenAI: gemini 2.5 wrote these tests, some comments were wrong

func TestNewContiguous(t *testing.T) {
	tests := []struct {
		name      string
		size      int
		start     int
		expectNil bool
	}{
		{"valid", 100, 10, false},
		{"valid start at 0", 10, 0, false},
		{"invalid size zero", 0, 10, true},
		{"invalid size negative", -1, 10, true},
		{"invalid start negative", 100, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := bitmap.NewContiguous(tt.size, tt.start)
			if (c == nil) != tt.expectNil {
				t.Errorf("NewContiguous() = %v, want nil: %v", c, tt.expectNil)
			}
		})
	}
}

func TestNewContiguousWithBitmap(t *testing.T) { //nolint:gocyclo
	t.Run("invalid parameters", func(t *testing.T) {
		bm := bitmap.New(100)
		testCases := []struct {
			name  string
			bm    bitmap.T
			size  int
			start int
		}{
			{"nil bitmap", nil, 100, 0},
			{"negative start", bm, 100, -1},
			{"start equals last (from size)", bm, 50, 50},
			{"start greater than last (from size)", bm, 50, 60},
			{"start equals last (from bitmap len)", bitmap.New(32), 100, 32},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := bitmap.NewContiguousWithBitmap(tc.bm, tc.size, tc.start)
				if c != nil {
					t.Errorf("NewContiguousWithBitmap with %v should have returned nil, but did not", tc.name)
				}
			})
		}
	})

	t.Run("valid initialization", func(t *testing.T) {
		// Case 1: Empty bitmap, start at 0.
		bm1 := bitmap.New(100)
		c1 := bitmap.NewContiguousWithBitmap(bm1, 100, 0)
		if c1 == nil {
			t.Fatal("c1 is nil")
		}
		if bitmap.NextValueForTesting(c1) != 0 {
			t.Errorf("Empty bitmap, start 0: got next %v, want 0", bitmap.NextValueForTesting(c1))
		}

		// Case 2: Empty bitmap, start at a non-zero value.
		bm2 := bitmap.New(100)
		c2 := bitmap.NewContiguousWithBitmap(bm2, 100, 10)
		if c2 == nil {
			t.Fatal("c2 is nil")
		}
		if bitmap.NextValueForTesting(c2) != 10 {
			t.Errorf("Empty bitmap, start 10: got next %v, want 10", bitmap.NextValueForTesting(c2))
		}

		// Case 3: Bitmap with existing contiguous block at start.
		bm3 := bitmap.New(100)
		bm3.Set(5)
		bm3.Set(6)
		c3 := bitmap.NewContiguousWithBitmap(bm3, 100, 5)
		if c3 == nil {
			t.Fatal("c3 is nil")
		}
		if bitmap.NextValueForTesting(c3) != 7 {
			t.Errorf("Existing block at start: got next %v, want 7", bitmap.NextValueForTesting(c3))
		}

		// Case 4: Bitmap with a gap at the specified start.
		bm4 := bitmap.New(100)
		bm4.Set(6) // Gap at 5.
		c4 := bitmap.NewContiguousWithBitmap(bm4, 100, 5)
		if c4 == nil {
			t.Fatal("c4 is nil")
		}
		if bitmap.NextValueForTesting(c4) != 5 {
			t.Errorf("Gap at start: got next %v, want 5", bitmap.NextValueForTesting(c4))
		}

		// Case 5: Bitmap where the tracked range is already full.
		bm5 := bitmap.New(100)
		for i := 10; i < 20; i++ {
			bm5.Set(i)
		}
		c5 := bitmap.NewContiguousWithBitmap(bm5, 20, 10) // Track from 10 up to 20.
		if c5 == nil {
			t.Fatal("c5 is nil")
		}
		if bitmap.NextValueForTesting(c5) != 20 {
			t.Errorf("Full range: got next %v, want 20", bitmap.NextValueForTesting(c5))
		}

		// Case 6: Size is smaller than bitmap capacity, limiting the tracked range.
		bm6 := bitmap.New(100)
		for i := 0; i < 50; i++ {
			bm6.Set(i)
		}
		c6 := bitmap.NewContiguousWithBitmap(bm6, 30, 0) // Track from 0 up to 30.
		if c6 == nil {
			t.Fatal("c6 is nil")
		}
		if bitmap.NextValueForTesting(c6) != 30 {
			t.Errorf("Limited size: got next %v, want 30", bitmap.NextValueForTesting(c6))
		}
		// Setting a bit within the original bitmap but outside the tracked range should have no effect on 'next'.
		c6.Set(35)
		if bitmap.NextValueForTesting(c6) != 30 {
			t.Errorf("Set outside range: got next %v, want 30", bitmap.NextValueForTesting(c6))
		}
	})
}
func TestContiguous_Set(t *testing.T) {
	c := bitmap.NewContiguous(10, 3) // Tracks bits from 3 to 10.

	// Set a bit before the start, should not advance next.
	c.Set(1)
	if bitmap.NextValueForTesting(c) != 3 {
		t.Errorf("got next %v, want 3", bitmap.NextValueForTesting(c))
	}

	// Set a bit at the start, should advance next.
	c.Set(3)
	if bitmap.NextValueForTesting(c) != 4 {
		t.Errorf("got next %v, want 4", bitmap.NextValueForTesting(c))
	}

	// Set a bit further on, creating a gap. next should not advance.
	c.Set(5)
	if bitmap.NextValueForTesting(c) != 4 {
		t.Errorf("got next %v, want 4", bitmap.NextValueForTesting(c))
	}

	// Fill the gap. next should advance to the new contiguous end.
	c.Set(4)
	if bitmap.NextValueForTesting(c) != 6 {
		t.Errorf("got next %v, want 6", bitmap.NextValueForTesting(c))
	}

	// Set invalid indices, should have no effect.
	c.Set(-1)
	c.Set(13) // size is 10, so last is 10.
	c.Set(100)
	if bitmap.NextValueForTesting(c) != 6 {
		t.Errorf("got next %v, want 6", bitmap.NextValueForTesting(c))
	}

	// Fill up to the end.
	c.Set(6)
	c.Set(7)
	c.Set(8)
	c.Set(9)
	if bitmap.NextValueForTesting(c) != 10 {
		t.Errorf("got next %v, want 10", bitmap.NextValueForTesting(c))
	}

	// Set past the end, should have no effect.
	c.Set(10)
	if bitmap.NextValueForTesting(c) != 10 {
		t.Errorf("got next %v, want 10", bitmap.NextValueForTesting(c))
	}
}

func TestContiguous_Next(t *testing.T) {
	c := bitmap.NewContiguous(20, 0)

	// 1. Get a channel and then advance the contiguous block.
	ch1 := c.Next()
	c.Set(0)

	select {
	case nextVal := <-ch1:
		if nextVal != 1 {
			t.Errorf("got next value %v, want 1", nextVal)
		}
	case <-time.After(10 * time.Millisecond):
		t.Fatal("timed out waiting for channel send")
	}

	// Channel should be closed now.
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("channel should be closed")
		}
	default:
		t.Error("channel should be closed and readable")
	}

	// 2. Create a gap, get a channel, then fill the gap.
	c.Set(2) // next is still 1.

	ch2 := c.Next()

	// Setting this should not trigger the channel.
	c.Set(3)

	select {
	case val := <-ch2:
		t.Fatalf("received %v from channel unexpectedly", val)
	default:
		// As expected.
	}

	// Now, fill the gap. This should trigger the channel.
	c.Set(1)

	select {
	case nextVal := <-ch2:
		if nextVal != 4 { // Should advance past 1, 2, and 3.
			t.Errorf("got next value %v, want 4", nextVal)
		}
	case <-time.After(10 * time.Millisecond):
		t.Fatal("timed out waiting for channel send")
	}

	// 3. Calling Next() multiple times should return the same channel until it's used.
	ch3a := c.Next()
	ch3b := c.Next()
	if ch3a != ch3b {
		t.Fatal("calling Next() multiple times should return the same channel instance")
	}

	c.Set(4)
	val := <-ch3a
	if val != 5 {
		t.Errorf("got %v, want 5", val)
	}

	// After being used, a new channel should be created.
	ch4 := c.Next()
	if ch4 == ch3a {
		t.Fatal("a new channel should have been created after the previous one was used")
	}

	for i := 5; i < 20; i++ {
		ch := c.Next()
		c.Set(i)
		if got, want := <-ch, i+1; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	// Return -1 for the next value after the last set bit.
	ch := c.Next()
	if got, want := <-ch, -1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	ch = c.Next()
	if got, want := <-ch, -1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestContiguous_Next_Done(t *testing.T) {
	c := bitmap.NewContiguous(20, 0)
	for i := range 19 {
		c.Set(i)
	}
	ch := c.Next()
	x := <-ch
	fmt.Printf("X: %v\n", x)
	t.Fatalf("expected channel to be closed after all bits set, but it was not")

}
