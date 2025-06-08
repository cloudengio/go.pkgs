// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap_test

import (
	"reflect"
	"slices"
	"testing"

	"cloudeng.io/algo/container/bitmap"
)

// gemini 2.5 wrote these tests, but failed to understand iterators
// and those tests needed signifant changes to work correctly.

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		size int
		want bitmap.T
	}{
		{"size 0", 0, nil},
		{"size -1", -1, nil},
		{"size 1", 1, make(bitmap.T, 1)},
		{"size 63", 63, make(bitmap.T, 1)},
		{"size 64", 64, make(bitmap.T, 1)},
		{"size 65", 65, make(bitmap.T, 2)},
		{"size 128", 128, make(bitmap.T, 2)},
		{"size 129", 129, make(bitmap.T, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bitmap.New(tt.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBitmap_Set(t *testing.T) {
	tests := []struct {
		name    string
		initial bitmap.T
		setIdx  int
		want    bitmap.T
	}{
		{"set 0 in 64bit", bitmap.New(64), 0, bitmap.T{1}},
		{"set 1 in 64bit", bitmap.New(64), 1, bitmap.T{2}},
		{"set 63 in 64bit", bitmap.New(64), 63, bitmap.T{1 << 63}},
		{"set 64 in 128bit", bitmap.New(128), 64, bitmap.T{0, 1}},
		{"set 127 in 128bit", bitmap.New(128), 127, bitmap.T{0, 1 << 63}},
		{"set existing", bitmap.T{1}, 0, bitmap.T{1}},
		{"set out of bounds negative", bitmap.New(64), -1, bitmap.New(64)}, // Should not change
		{"set out of bounds positive", bitmap.New(64), 64, bitmap.New(64)}, // Should not change
		{"set in nil bitmap", nil, 0, bitmap.T{}},                          // Should not panic
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clone initial to avoid modifying it across tests if it's not from New()
			b := make(bitmap.T, len(tt.initial))
			copy(b, tt.initial)
			b.Set(tt.setIdx)
			if !reflect.DeepEqual(b, tt.want) {
				t.Errorf("Bitmap.Set() test %d got %v, want %v", i, b, tt.want)
			}
		})
	}
}

func TestBitmap_Clear(t *testing.T) {
	tests := []struct {
		name     string
		initial  bitmap.T
		clearIdx int
		want     bitmap.T
	}{
		{"clear 0 in 64bit", bitmap.T{1}, 0, bitmap.T{0}},
		{"clear 1 in 64bit", bitmap.T{2}, 1, bitmap.T{0}},
		{"clear 63 in 64bit", bitmap.T{1 << 63}, 63, bitmap.T{0}},
		{"clear 64 in 128bit", bitmap.T{0, 1}, 64, bitmap.T{0, 0}},
		{"clear 127 in 128bit", bitmap.T{0, 1 << 63}, 127, bitmap.T{0, 0}},
		{"clear non-existing", bitmap.T{1}, 1, bitmap.T{1}}, // Clearing bit 1 when only bit 0 is set
		{"clear out of bounds negative", bitmap.T{1}, -1, bitmap.T{1}},
		{"clear out of bounds positive", bitmap.T{1}, 64, bitmap.T{1}}, // Assuming bitmap is size 64
		{"clear in nil bitmap", nil, 0, bitmap.T{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := make(bitmap.T, len(tt.initial))
			copy(b, tt.initial)
			b.Clear(tt.clearIdx)
			if !reflect.DeepEqual(b, tt.want) {
				t.Errorf("Bitmap.Clear() got %v, want %v", b, tt.want)
			}
		})
	}
}

func TestBitmap_IsSet(t *testing.T) {
	tests := []struct {
		name    string
		initial bitmap.T
		idx     int
		want    bool
	}{
		{"check set 0", bitmap.T{1}, 0, true},
		{"check set 63", bitmap.T{1 << 63}, 63, true},
		{"check set 64", bitmap.T{0, 1}, 64, true},
		{"check clear 0", bitmap.T{0}, 0, false},
		{"check clear 1 (when 0 is set)", bitmap.T{1}, 1, false},
		{"check out of bounds negative", bitmap.T{1}, -1, false},
		{"check out of bounds positive (size 64)", bitmap.T{1}, 64, false},
		{"check out of bounds positive (size 128, idx 128)", bitmap.T{0, 1}, 128, false},
		{"check in nil bitmap", nil, 0, false},
		{"check large index in small bitmap", bitmap.New(10), 65, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No need to copy initial for IsSet
			if got := tt.initial.IsSet(tt.idx); got != tt.want {
				t.Errorf("Bitmap.IsSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBitmap_Operations(t *testing.T) { //nolint:gocyclo
	size := 130 // Will require 3 uint64
	bm := bitmap.New(size)

	// Test initial state (all clear)
	for i := range size {
		if bm.IsSet(i) {
			t.Errorf("New bitmap: bit %d should be clear, but is set", i)
		}
	}
	if bm.IsSet(size) {
		t.Errorf("New bitmap: bit %d (out of bounds) should be clear, but is set", size)
	}

	// Set some bits
	bitsToSet := []int{0, 10, 63, 64, 65, 127, 129}
	for _, bit := range bitsToSet {
		bm.Set(bit)
	}

	// Verify set bits
	for i := range size {
		expectedSet := slices.Contains(bitsToSet, i)
		if bm.IsSet(i) != expectedSet {
			t.Errorf("After Set: bit %d, IsSet() = %v, want %v", i, bm.IsSet(i), expectedSet)
		}
	}

	// Set an already set bit
	bm.Set(10)
	if !bm.IsSet(10) {
		t.Errorf("After setting already set bit 10: IsSet(10) = false, want true")
	}

	// Clear some bits
	bitsToClear := []int{10, 64, 129}
	for _, bit := range bitsToClear {
		bm.Clear(bit)
	}

	// Verify cleared bits and remaining set bits
	remainingSetBits := []int{0, 63, 65, 127}
	for i := range size {
		expectedSet := slices.Contains(remainingSetBits, i)
		if bm.IsSet(i) != expectedSet {
			t.Errorf("After Clear: bit %d, IsSet() = %v, want %v", i, bm.IsSet(i), expectedSet)
		}
	}

	// Clear an already clear bit
	bm.Clear(10) // Was already cleared
	if bm.IsSet(10) {
		t.Errorf("After clearing already clear bit 10: IsSet(10) = true, want false")
	}

	// Test out of bounds operations again
	bm.Set(size + 10)        // Should do nothing
	bm.Clear(size + 10)      // Should do nothing
	if bm.IsSet(size + 10) { // Should be false
		t.Errorf("IsSet for out of bounds index %d returned true", size+10)
	}

	// Check specific word values if needed for deeper debugging
	// Example: bm[0] should have bits 0 and 63 set.
	expectedWord0 := (uint64(1) << 0) | (uint64(1) << 63)
	if bm[0] != expectedWord0 {
		t.Errorf("bm[0] = %064b, want %064b", bm[0], expectedWord0)
	}
	// Example: bm[1] should have bits 65 (relative 1) and 127 (relative 63) set.
	expectedWord1 := (uint64(1) << (65 - 64)) | (uint64(1) << (127 - 64))
	if bm[1] != expectedWord1 {
		t.Errorf("bm[1] = %064b, want %064b", bm[1], expectedWord1)
	}
	// Example: bm[2] should be 0 as bit 129 was cleared.
	expectedWord2 := uint64(0)
	if bm[2] != expectedWord2 {
		t.Errorf("bm[2] = %064b, want %064b", bm[2], expectedWord2)
	}
}

// Helper for generating sequences like 0, 1, 2, ..., endInclusive
func iotaSlice(start, endInclusive int) []int {
	if start > endInclusive {
		return []int{}
	}
	s := make([]int, 0, endInclusive-start+1)
	for i := start; i <= endInclusive; i++ {
		s = append(s, i)
	}
	return s
}

func TestBitmap_NextClear(t *testing.T) {
	tests := []struct {
		name       string
		bm         bitmap.T
		startIndex int
		size       int
		wantSeq    []int // Expected sequence of values from iterator, ending with -1
	}{
		{"nil bitmap", nil, 0, 64, []int{}},
		{"empty bitmap (len 0)", bitmap.T{}, 0, 64, []int{}},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 0, 64, []int{}},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 63, 64, []int{}},
		{"all set (1 word) (OOB)", bitmap.T{^uint64(0)}, 64, 64, []int{}},
		{"all set (2 words)", bitmap.T{^uint64(0), ^uint64(0)}, 0, 128, []int{}},
		{"all set (2 words) 120", bitmap.T{^uint64(0), ^uint64(0)}, 120, 128, []int{}},
		{"all clear (1 word)", bitmap.T{0}, 0, 64, iotaSlice(0, 63)},
		{"all clear (1 word)", bitmap.T{0}, 0, 1, []int{0}},
		{"all clear (1 word)", bitmap.T{0}, 0, 27, iotaSlice(0, 26)},
		{"all clear (1 word)", bitmap.T{0}, 60, 64, iotaSlice(60, 63)},
		{"all clear (1 word)", bitmap.T{0}, 63, 64, []int{63}},
		{"all clear (1 word) (OOB)", bitmap.T{0}, 64, 64, []int{}},
		{"all clear (2 words) 0", bitmap.T{0, 0}, 0, 128, iotaSlice(0, 127)},
		{"all clear (2 words) 125", bitmap.T{0, 0}, 125, 128, []int{125, 126, 127}},
		{"all clear, start negative", bitmap.T{0}, -5, 64, []int{}},
		{"first bit clear in full word", bitmap.T{^uint64(0) - 1}, 0, 64, []int{0}}, // ...110
		{"second bit clear", bitmap.T{^uint64(0) - 2}, 0, 64, []int{1}},             // ...101
		{"second bit clear", bitmap.T{^uint64(0) - 2}, 1, 64, []int{1}},
		{"second bit clear", bitmap.T{^uint64(0) - 2}, 2, 64, []int{}},
		{"63rd bit clear", bitmap.T{^(uint64(1) << 63)}, 0, 64, []int{63}},
		{"63rd bit clear", bitmap.T{^(uint64(1) << 63)}, 63, 64, []int{63}},
		{"64th bit clear (word 1, bit 0)", bitmap.T{^uint64(0), ^uint64(0) - 1}, 0, 64, []int{}},
		{"64th bit clear (word 1, bit 0)", bitmap.T{^uint64(0), ^uint64(0) - 1}, 0, 65, []int{64}},
		{"64th bit clear", bitmap.T{^uint64(0), ^uint64(0) - 1}, 60, 65, []int{64}},
		{"64th bit clear", bitmap.T{^uint64(0), ^uint64(0) - 1}, 64, 64, []int{}},
		{"0xF0 pattern (bits 0-3 clear per nibble)", bitmap.T{0xF0F0F0F0F0F0F0F0}, 0, 64, []int{0, 1, 2, 3, 8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"0xF0 pattern", bitmap.T{0xF0F0F0F0F0F0F0F0}, 2, 64, []int{2, 3, 8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"0xF0 pattern 4 (finds 8)", bitmap.T{0xF0F0F0F0F0F0F0F0}, 4, 64, []int{8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"two words, second word all clear", bitmap.T{^uint64(0), 0}, 0, 128, iotaSlice(64, 127)},
		{"two words, second word all clear", bitmap.T{^uint64(0), 0}, 60, 128, iotaSlice(64, 127)},
		{"two words, second word all clear", bitmap.T{^uint64(0), 0}, 64, 128, iotaSlice(64, 127)},
		{"start index beyond capacity", bitmap.New(64), 100, 128, []int{}}, // New(64) is 1 word.
		{"bitmap New(0)", bitmap.New(0), 0, 64, []int{}},                   // New(0) results in nil bitmap
		{"bitmap New(1), all clear", bitmap.New(1), 0, 64, iotaSlice(0, 63)},
		{"bitmap New(65), all clear", bitmap.New(65), 64, 128, iotaSlice(64, 127)},
		{"bitmap New(65), all clear, OOB", bitmap.New(65), 65, 128, iotaSlice(65, 127)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSeq []int
			for v := range tt.bm.NextClear(tt.startIndex, tt.size) {
				gotSeq = append(gotSeq, v)
			}
			if !slices.Equal(gotSeq, tt.wantSeq) {
				t.Errorf("Bitmap.NextClear(test %s, start %d, size %d) sequence = %v, want %v (bm: %064b)", tt.name, tt.startIndex, tt.size, gotSeq, tt.wantSeq, tt.bm)
			}

			buf, err := tt.bm.MarshalJSON()
			if err != nil {
				t.Errorf("Bitmap.MarshalJSON(test %s) = %v", tt.name, err)
			}
			var unmarshaled bitmap.T
			err = unmarshaled.UnmarshalJSON(buf)
			if err != nil {
				t.Errorf("Bitmap.UnmarshalJSON(test %s) = %v", tt.name, err)
			}
			if !slices.Equal(tt.bm, unmarshaled) {
				t.Errorf("Bitmap.UnmarshalJSON(test %s) = %064b, want %064b", tt.name, unmarshaled, tt.bm)
			}

		})
	}
}

func TestBitmap_NextSet(t *testing.T) {
	tests := []struct {
		name       string
		bm         bitmap.T
		startIndex int
		size       int
		wantSeq    []int // Expected sequence of values from iterator, ending with -1
	}{
		{"nil bitmap", nil, 0, 64, []int{}},
		{"empty bitmap (len 0)", bitmap.T{}, 0, 64, []int{}},
		{"all clear (1 word)", bitmap.T{0}, 0, 64, []int{}},
		{"all clear (1 word)", bitmap.T{0}, 63, 64, []int{}},
		{"all clear (1 word) (OOB)", bitmap.T{0}, 64, 64, []int{}},
		{"all clear (2 words)", bitmap.T{0, 0}, 0, 128, []int{}},
		{"all clear (2 words)", bitmap.T{0, 0}, 120, 128, []int{}},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 0, 64, iotaSlice(0, 63)},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 0, 1, []int{0}},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 60, 63, iotaSlice(60, 62)},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 60, 64, iotaSlice(60, 63)},
		{"all set (1 word)", bitmap.T{^uint64(0)}, 63, 64, []int{63}},
		{"all set (1 word) (OOB)", bitmap.T{^uint64(0)}, 64, 64, []int{}},
		{"all set (2 words)", bitmap.T{^uint64(0), ^uint64(0)}, 0, 128, iotaSlice(0, 127)},
		{"all set (2 words)", bitmap.T{^uint64(0), ^uint64(0)}, 0, 1024, iotaSlice(0, 127)},
		{"all set (2 words)", bitmap.T{^uint64(0), ^uint64(0)}, 0, 121, iotaSlice(0, 120)},
		{"all set (2 words)", bitmap.T{^uint64(0), ^uint64(0)}, 125, 128, []int{125, 126, 127}},
		{"all set, start negative", bitmap.T{^uint64(0)}, -5, 128, []int{}},
		{"first bit set", bitmap.T{1}, 0, 128, []int{0}},
		{"second bit set", bitmap.T{2}, 0, 128, []int{1}},
		{"second bit set", bitmap.T{2}, 1, 128, []int{1}},
		{"63rd bit set", bitmap.T{uint64(1) << 63}, 0, 128, []int{63}},
		{"63rd bit set", bitmap.T{uint64(1) << 63}, 63, 128, []int{63}},
		{"64th bit set (word 1, bit 0)", bitmap.T{0, 1}, 0, 128, []int{64}},
		{"64th bit set", bitmap.T{0, 1}, 60, 128, []int{64}},
		{"64th bit set", bitmap.T{0, 1}, 64, 128, []int{64}},
		{"0x0F pattern (bits 0-3 set per nibble)", bitmap.T{0x0F0F0F0F0F0F0F0F}, 0, 128, []int{0, 1, 2, 3, 8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"0x0F pattern", bitmap.T{0x0F0F0F0F0F0F0F0F}, 2, 128, []int{2, 3, 8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"0x0F pattern (finds 8)", bitmap.T{0x0F0F0F0F0F0F0F0F}, 4, 128, []int{8, 9, 10, 11, 16, 17, 18, 19, 24, 25, 26, 27, 32, 33, 34, 35, 40, 41, 42, 43, 48, 49, 50, 51, 56, 57, 58, 59}},
		{"two words, second word all set", bitmap.T{0, ^uint64(0)}, 0, 128, iotaSlice(64, 127)},
		{"two words, second word all set", bitmap.T{0, ^uint64(0)}, 60, 128, iotaSlice(64, 127)},
		{"two words, second word all set", bitmap.T{0, ^uint64(0)}, 64, 128, iotaSlice(64, 127)},
		{"start index beyond capacity", bitmap.New(64), 100, 128, []int{}},
		{"bitmap New(0)", bitmap.New(0), 0, 64, []int{}},
		{"bitmap New(1), all set", func() bitmap.T { b := bitmap.New(1); b.Set(0); return b }(), 0, 64, []int{0}},
		{"bitmap New(65), all set", func() bitmap.T { b := bitmap.New(65); b.Set(64); return b }(), 64, 128, []int{64}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSeq []int
			for v := range tt.bm.NextSet(tt.startIndex, tt.size) {
				gotSeq = append(gotSeq, v)
			}
			if !slices.Equal(gotSeq, tt.wantSeq) {
				t.Errorf("Bitmap.NextSet(test: %v, start %d, size %d) sequence = %v, want %v (bm: %064b)", tt.name, tt.startIndex, tt.size, gotSeq, tt.wantSeq, tt.bm)
			}
		})
	}
}
