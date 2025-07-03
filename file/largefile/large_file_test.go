// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"cloudeng.io/file/largefile"
)

// GenAI comment: gemini 2.5 TestNewByteRanges, TestByteRanges_MarshalUnmarshalJSON
// but not the other tests

func TestNewByteRanges(t *testing.T) {
	tests := []struct {
		name            string
		contentSize     int64
		blockSize       int
		wantContentSize int64
		wantBlockSize   int
		wantBitmapSize  int
		wantNilBitmap   bool
	}{
		{"exact blocks", 1000, 100, 1000, 100, 10, false},
		{"partial block", 1005, 100, 1005, 100, 11, false},
		{"small content, one block", 99, 100, 99, 100, 1, false},
		{"content size zero", 0, 100, 0, 100, 0, true},
		{"block size one", 10, 1, 10, 1, 10, false},
		{"content size equals block size", 100, 100, 100, 100, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := largefile.NewByteRanges(tt.contentSize, tt.blockSize)
			if br.ContentLength() != tt.wantContentSize {
				t.Errorf("NewByteRanges() contentSize = %v, want %v", br.ContentLength(), tt.wantContentSize)
			}
			if br.BlockSize() != tt.wantBlockSize {
				t.Errorf("NewByteRanges() blockSize = %v, want %v", br.BlockSize(), tt.wantBlockSize)
			}
		})
	}
}

func TestByteRanges_MarshalUnmarshalJSON(t *testing.T) {
	originalBR := largefile.NewByteRanges(1024, 128) // 8 blocks
	originalBR.Set(0)
	originalBR.Set(512)
	originalBR.Set(513)
	originalBR.Set(1023)
	originalBR.Set(-1)
	originalBR.Set(1025)

	originalSet, originalClear := isSetAndClear(originalBR)

	jsonData, err := json.Marshal(originalBR)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var unmarshaledBR largefile.ByteRanges
	if err := json.Unmarshal(jsonData, &unmarshaledBR); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if unmarshaledBR.ContentLength() != originalBR.ContentLength() {
		t.Errorf("UnmarshalJSON() contentLength = %v, want %v", unmarshaledBR.ContentLength(), originalBR.ContentLength())
	}
	if unmarshaledBR.BlockSize() != originalBR.BlockSize() {
		t.Errorf("UnmarshalJSON() blockSize = %v, want %v", unmarshaledBR.BlockSize(), originalBR.BlockSize())
	}

	unmarshaledSet, unmarshaledClear := isSetAndClear(&unmarshaledBR)

	if !slices.Equal(unmarshaledSet, originalSet) {
		t.Errorf("UnmarshalJSON() NextSet(0) = %v, want %v", unmarshaledSet, originalSet)
	}
	if !slices.Equal(unmarshaledClear, originalClear) {
		t.Errorf("UnmarshalJSON() NextClear(0) = %v, want %v", unmarshaledClear, originalClear)
	}

	t.Run("unmarshal error - bad content size", func(t *testing.T) {
		badJSON := `{"content_size":"not-a-number","block_size":128,"bitmap_size":8,"ranges":null}`
		var br largefile.ByteRanges
		err := json.Unmarshal([]byte(badJSON), &br)
		if err == nil {
			t.Error("UnmarshalJSON() expected error for bad content_size, got nil")
		}
		var numErr *strconv.NumError
		if !errors.As(err, &numErr) {
			t.Errorf("UnmarshalJSON() expected strconv.NumError, got %T", err)
		}
	})

	t.Run("unmarshal error - bad bitmap data", func(t *testing.T) {
		// Assuming bitmap.UnmarshalJSON returns an error for malformed bitmap JSON.
		// This depends on the implementation of bitmap.UnmarshalJSON.
		// Let's simulate malformed JSON for the bitmap part.
		// A valid bitmap JSON might be like `["AQAAAAAAAAA="]` for one word with bit 0 set.
		// An invalid one could be just a string or a malformed array.
		badBitmapJSON := `{"content_size":"1024","block_size":128,"ranges":"this is not valid bitmap json"}`
		var br largefile.ByteRanges
		err := json.Unmarshal([]byte(badBitmapJSON), &br)
		if err == nil {
			t.Error("UnmarshalJSON() expected error for bad bitmap ranges, got nil")
		}
		// The exact error type depends on bitmap.UnmarshalJSON.
		// We can check if it's non-nil.
	})

	t.Run("unmarshal with nil bitmap (content size 0)", func(t *testing.T) {
		brZero := largefile.NewByteRanges(0, 100)
		jsonDataZero, err := json.Marshal(brZero)
		if err != nil {
			t.Fatalf("MarshalJSON() for zero content size error = %v", err)
		}
		// Expected JSON for bitmap part might be "null" or an empty array "[]"
		// depending on bitmap.T.MarshalJSON() for a nil bitmap.
		// Let's assume bitmap.UnmarshalJSON can handle this.

		var unmarshaledZeroBR largefile.ByteRanges
		if err := json.Unmarshal(jsonDataZero, &unmarshaledZeroBR); err != nil {
			t.Fatalf("UnmarshalJSON() for zero content size error = %v", err)
		}
		if unmarshaledZeroBR.ContentLength() != 0 {
			t.Errorf("UnmarshalJSON() zero contentLength = %v, want 0", unmarshaledZeroBR.ContentLength())
		}
	})
}

func isSetAndClear(br *largefile.ByteRanges) (set, clr []largefile.ByteRange) {
	for s := range br.AllSet(0) {
		set = append(set, s)
	}
	for c := range br.AllClear(0) {
		clr = append(clr, c)
	}
	return
}

func expectedSetAndClear(size int64, blockSize int, positions ...int) (set, clr []largefile.ByteRange) {
	if size <= 0 || blockSize == 0 {
		return []largefile.ByteRange{}, []largefile.ByteRange{}
	}
	iset := map[int]struct{}{}
	for _, pos := range positions {
		b := pos / blockSize
		if _, ok := iset[b]; ok {
			continue // Skip if already set.
		}
		from := int64(b) * int64(blockSize)
		to := min(from+int64(blockSize), size)
		set = append(set, largefile.ByteRange{
			From: from,
			To:   to - 1,
		})
		iset[b] = struct{}{}
	}
	nb := largefile.NumBlocks(size, blockSize)
	for i := range nb {
		if _, ok := iset[i]; !ok {
			from := int64(i * blockSize)
			to := min(from+int64(blockSize), size)
			clr = append(clr, largefile.ByteRange{
				From: from,
				To:   to - 1,
			})
		}
	}
	return
}

func compareRanges(t *testing.T, br *largefile.ByteRanges, size int64, blockSize int, positions ...int) {
	t.Helper()
	gotSet, gotClear := isSetAndClear(br)
	wantSet, wantClear := expectedSetAndClear(size, blockSize, positions...)

	if !slices.Equal(gotSet, wantSet) {
		t.Errorf("NextSet(0) = %v, want %v", gotSet, wantSet)
	}
	if !slices.Equal(gotClear, wantClear) {
		t.Errorf("NextClear(0) = %v, want %v", gotClear, wantClear)
	}
}

func TestRanges(t *testing.T) {
	contentSize := int64(1010)
	blockSize := 250
	br := largefile.NewByteRanges(contentSize, blockSize)

	compareRanges(t, br, contentSize, blockSize)

	// Test setting and getting ranges
	br.Set(0)
	br.Set(500)
	br.Set(600)
	br.Set(1001)

	compareRanges(t, br, contentSize, blockSize, 0, 500, 600, 1001)

	allSet := []largefile.ByteRange{}
	for b := range br.AllSet(0) {
		allSet = append(allSet, b)
	}
	if allSet[0].Size() != 250 {
		t.Errorf("NextSet(0) size = %d, want 250", allSet[0].Size())
	}
	if allSet[2].Size() != 10 {
		t.Errorf("NextSet(1000) size = %d, want 10", allSet[2].Size())
	}

	allSet = []largefile.ByteRange{}
	var b largefile.ByteRange
	for n := br.NextSet(0, &b); n >= 0; n = br.NextSet(n+1, &b) {
		allSet = append(allSet, b)
	}

	if allSet[0].Size() != 250 {
		t.Errorf("NextSet(0) size = %d, want 250", allSet[0].Size())
	}
	if allSet[2].Size() != 10 {
		t.Errorf("NextSet(1000) size = %d, want 10", allSet[2].Size())
	}
}

func TestRangesFunction(t *testing.T) {
	tests := []struct {
		name       string
		from       int64
		to         int64
		blockSize  int
		wantRanges []largefile.ByteRange
	}{
		{
			name:      "exact multiple from zero",
			from:      0,
			to:        99,
			blockSize: 25,
			wantRanges: []largefile.ByteRange{
				{From: 0, To: 24},
				{From: 25, To: 49},
				{From: 50, To: 74},
				{From: 75, To: 99},
			},
		},
		{
			name:      "partial last block from zero",
			from:      0,
			to:        99,
			blockSize: 30,
			wantRanges: []largefile.ByteRange{
				{From: 0, To: 29},
				{From: 30, To: 59},
				{From: 60, To: 89},
				{From: 90, To: 99},
			},
		},
		{
			// This test case demonstrates the current behavior where the
			// yielded ranges are relative to 0, not the 'from' parameter.
			name:      "non-zero from",
			from:      50,
			to:        149,
			blockSize: 25,
			wantRanges: []largefile.ByteRange{
				{From: 50, To: 74},
				{From: 75, To: 99},
				{From: 100, To: 124},
				{From: 125, To: 149},
			},
		},
		{
			name:      "single block smaller than blocksize",
			from:      0,
			to:        49,
			blockSize: 100,
			wantRanges: []largefile.ByteRange{
				{From: 0, To: 49},
			},
		},
		{
			name:      "range size equals block size",
			from:      0,
			to:        99,
			blockSize: 100,
			wantRanges: []largefile.ByteRange{
				{From: 0, To: 99},
			},
		},
		{
			name:       "zero size range",
			from:       10,
			to:         9,
			blockSize:  100,
			wantRanges: []largefile.ByteRange{},
		},
		{
			name:       "zero block size",
			from:       0,
			to:         99,
			blockSize:  0,
			wantRanges: []largefile.ByteRange{},
		},
		{
			name:       "negative block size",
			from:       0,
			to:         99,
			blockSize:  -1,
			wantRanges: []largefile.ByteRange{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRanges []largefile.ByteRange
			for r := range largefile.Ranges(tt.from, tt.to, tt.blockSize) {
				gotRanges = append(gotRanges, r)
			}

			if !slices.Equal(gotRanges, tt.wantRanges) {
				t.Errorf("Ranges() got = %v, want %v", gotRanges, tt.wantRanges)
			}
		})
	}
}

func TestByteRanges_NotifyAndTail(t *testing.T) {
	const (
		contentSize = 4096
		blockSize   = 1024
	)

	t.Run("Notify closes when contiguous range is extended", func(t *testing.T) {
		br := largefile.NewByteRanges(contentSize, blockSize)
		ch := br.Notify()
		select {
		case <-ch:
			t.Fatal("channel should not be closed yet")
		default:
		}
		br.Set(0)
		select {
		case <-ch:
			// Success: channel closed after first block set
		case <-time.After(50 * time.Millisecond):
			t.Fatal("channel was not closed after contiguous extension")
		}
	})

	t.Run("Notify already closed if range is complete", func(t *testing.T) {
		br := largefile.NewByteRanges(contentSize, blockSize)
		for i := 0; i < br.NumBlocks(); i++ {
			br.Set(int64(i * blockSize))
		}
		ch := br.Notify()
		select {
		case <-ch:
			// Success: already closed
		default:
			t.Fatal("channel should be closed for complete range")
		}
	})

	t.Run("Tail returns false if nothing set", func(t *testing.T) {
		br := largefile.NewByteRanges(contentSize, blockSize)
		r, ok := br.Tail()
		if ok {
			t.Errorf("Tail() = %v, %v; want false", r, ok)
		}
	})

	t.Run("Tail returns correct range after Set", func(t *testing.T) {
		br := largefile.NewByteRanges(contentSize, blockSize)
		br.Set(0)
		r, ok := br.Tail()
		want := largefile.ByteRange{From: 0, To: 1023}
		if !ok {
			t.Fatal("Tail() returned false after Set")
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("Tail() = %v, want %v", r, want)
		}
	})

	t.Run("Tail returns full range when all set", func(t *testing.T) {
		br := largefile.NewByteRanges(contentSize, blockSize)
		for i := 0; i < br.NumBlocks(); i++ {
			br.Set(int64(i * blockSize))
		}
		r, ok := br.Tail()
		want := largefile.ByteRange{From: 0, To: contentSize - 1}
		if !ok {
			t.Fatal("Tail() returned false for complete range")
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("Tail() = %v, want %v", r, want)
		}
	})
}
