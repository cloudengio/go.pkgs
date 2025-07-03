// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"encoding/json"
	"fmt"
	"iter"
	"strconv"
	"sync"

	"cloudeng.io/algo/container/bitmap"
)

type byteRanges struct {
	contentSize int64
	bitmapSize  int
	blockSize   int
	bitmap      bitmap.T
}

func newByteRanges(contentSize int64, blockSize int) byteRanges {
	nb := NumBlocks(contentSize, blockSize)
	bm := bitmap.New(nb)
	return byteRanges{
		contentSize: contentSize,
		blockSize:   blockSize,
		bitmapSize:  nb,
		bitmap:      bm,
	}
}

// NumBlocks returns the number of blocks required to cover the byte ranges
// represented by this ByteRanges instance.
func (br byteRanges) NumBlocks() int {
	// NumBlocks returns the number of blocks in the byte ranges.
	return br.bitmapSize
}

// ContentLength returns the total size of the content in bytes.
func (br byteRanges) ContentLength() int64 {
	return br.contentSize
}

// BlockSize returns the size of each block in bytes.
func (br byteRanges) BlockSize() int {
	return br.blockSize
}

func (br *byteRanges) MarshalJSON() ([]byte, error) {
	jr, err := br.bitmap.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ranges := struct {
		ContentSize string          `json:"content_size"`
		BlockSize   int             `json:"block_size"`
		Ranges      json.RawMessage `json:"ranges"`
	}{
		ContentSize: strconv.FormatInt(br.contentSize, 10),
		BlockSize:   br.blockSize,
		Ranges:      jr,
	}
	return json.Marshal(ranges)
}

func (br *byteRanges) UnmarshalJSON(data []byte) error {
	var ranges struct {
		ContentSize string          `json:"content_size"`
		BlockSize   int             `json:"block_size"`
		Ranges      json.RawMessage `json:"ranges"`
	}
	if err := json.Unmarshal(data, &ranges); err != nil {
		return err
	}
	if ranges.ContentSize == "" || ranges.BlockSize <= 0 || len(ranges.Ranges) == 0 {
		return fmt.Errorf("invalid content size, block size, or empty ranges")
	}
	var err error
	br.contentSize, err = strconv.ParseInt(ranges.ContentSize, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid content size: %w", err)
	}
	br.blockSize = ranges.BlockSize
	br.bitmapSize = NumBlocks(br.contentSize, ranges.BlockSize)
	err = br.bitmap.UnmarshalJSON(ranges.Ranges)
	if err != nil {
		return fmt.Errorf("invalid bitmap data: %w", err)
	}
	return nil
}

// NextClear returns the next clear byte range starting from 'start'.
// It starts searching from the specified start index and returns the
// index of the next outstanding range which can be used to continue
// searching for the next outstanding range. The index will be -1
// if there are no more outstanding ranges.
//
//	for start := NextClear(0, &br); start >= 0; start = NextClear(start, &br) {
//	    // Do something with the byte range br.
//	}
func (br byteRanges) NextClear(start int, nbr *ByteRange) int {
	i := br.bitmap.NextClear(start, br.bitmapSize)
	if i < 0 {
		return -1
	}
	*nbr = RangeForIndex(i, br.contentSize, br.blockSize)
	return i + 1
}

// NextSet returns the next set byte range starting from 'start' and
// behaves similarly to NextClear.
func (br byteRanges) NextSet(start int, nbr *ByteRange) int {
	i := br.bitmap.NextSet(start, br.bitmapSize)
	if i < 0 {
		return -1
	}
	*nbr = RangeForIndex(i, br.contentSize, br.blockSize)
	return i + 1
}

// Block returns the block index for the specified position.
// It returns -1 if the position is out of bounds.
func (br byteRanges) Block(pos int64) int {
	// Check if the position is out of bounds.
	if pos < 0 || pos >= br.contentSize {
		return -1
	}
	blockIndex := int(pos) / br.blockSize
	if blockIndex < 0 || blockIndex >= br.bitmapSize {
		return -1
	}
	return blockIndex
}

// Set marks the byte range for the specified position as set.
// It has no effect if the position is out of bounds.
func (br *byteRanges) Set(pos int64) {
	if blockIndex := br.Block(pos); blockIndex >= 0 {
		br.bitmap.SetUnsafe(blockIndex)
	}
}

func (br *byteRanges) clear(pos int64) {
	if blockIndex := br.Block(pos); blockIndex >= 0 {
		br.bitmap.Clear(blockIndex)
	}
}

// IsSet checks if the byte range for the specified position is set.
func (br byteRanges) IsSet(pos int64) bool {
	if blockIndex := br.Block(pos); blockIndex >= 0 {
		return br.bitmap.IsSetUnsafe(blockIndex)
	}
	return false
}

// IsClear checks if the byte range for the specified position is clear.
func (br byteRanges) IsClear(pos int64) bool {
	if blockIndex := br.Block(pos); blockIndex >= 0 {
		return !br.bitmap.IsSet(blockIndex)
	}
	return false
}

// ByteRanges represents a collection of equally sized (apart from the last
// range), contiguous, byte ranges that can be used to track which parts of
// a file have or have not been 'processed', e.g downloaded, cached, uploaded
// etc. The ranges are represented as a bitmap, where each bit corresponds to
// a block of bytes of the specified size. The bitmap is used to efficiently
// track which byte ranges are set (processed) and which are clear (not processed).
// ByteRanges also allows for the contiguous head of the byte ranges to be
// tracked asynchronously. ByteRanges is thread-safe and can be used
// concurrently by multiple goroutines.
type ByteRanges struct {
	mu sync.RWMutex
	byteRanges
	contiguous *bitmap.Contiguous
}

// NewByteRanges creates a new ByteRanges instance with the specified content size
// and block size. The content size is the total size of the file in bytes, and
// the block size is the size of each byte range in bytes.
func NewByteRanges(contentSize int64, blockSize int) *ByteRanges {
	br := newByteRanges(contentSize, blockSize)
	nbr := &ByteRanges{
		byteRanges: br,
		contiguous: bitmap.NewContiguousWithBitmap(br.bitmap, 0, br.bitmapSize),
	}
	return nbr
}

// MarshalJSON implements the json.Marshaler interface for ByteRanges.
func (br *ByteRanges) MarshalJSON() ([]byte, error) {
	return br.byteRanges.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for ByteRanges.
func (br *ByteRanges) UnmarshalJSON(data []byte) error {
	if err := br.byteRanges.UnmarshalJSON(data); err != nil {
		return err
	}
	br.contiguous = bitmap.NewContiguousWithBitmap(br.bitmap, 0, br.bitmapSize)
	return nil
}

// NextClear returns the next clear byte range starting from 'start'.
// It starts searching from the specified start index and returns the
// index of the next outstanding range which can be used to continue
// searching for the next outstanding range. The index will be -1
// if there are no more outstanding ranges.
//
//	for start := NextClear(0, &br); start >= 0; start = NextClear(start, &br) {
//	    // Do something with the byte range br.
//	}
func (br *ByteRanges) NextClear(start int, nbr *ByteRange) int {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.byteRanges.NextClear(start, nbr)
}

// NextSet returns the next set byte range starting from 'start' and
// behaves similarly to NextClear.
func (br *ByteRanges) NextSet(start int, nbr *ByteRange) int {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.byteRanges.NextSet(start, nbr)
}

// AllClear returns an iterator for all clear byte ranges starting from 'start'.
// A read lock is held while iterating over the byte ranges, hence
// calling any other method, such as Set, which takes a write lock will
// block until the iteration is complete. Use NextClear if finer-grained
// control is needed.
func (br *ByteRanges) AllClear(start int) iter.Seq[ByteRange] {
	return func(yield func(ByteRange) bool) {
		br.mu.RLock()
		defer br.mu.RUnlock()
		for i := range br.bitmap.AllClear(start, br.bitmapSize) {
			if !yield(RangeForIndex(i, br.contentSize, br.blockSize)) {
				return
			}
		}
	}
}

// AllSet returns an iterator for all set byte ranges starting from 'start'.
// A read lock is held while iterating over the byte ranges, hence
// calling any other method, such as Set, which takes a write lock will
// block until the iteration is complete. Use NextSet if finer-grained
// control is needed.
func (br *ByteRanges) AllSet(start int) iter.Seq[ByteRange] {
	return func(yield func(ByteRange) bool) {
		br.mu.RLock()
		defer br.mu.RUnlock()
		for i := range br.bitmap.AllSet(start, br.bitmapSize) {
			if !yield(RangeForIndex(i, br.contentSize, br.blockSize)) {
				return
			}
		}
	}
}

// Set marks the byte range for the specified position as set.
// It has no effect if the position is out of bounds.
func (br *ByteRanges) Set(pos int64) {
	br.mu.Lock()
	defer br.mu.Unlock()
	if blockIndex := br.Block(pos); blockIndex >= 0 {
		br.contiguous.Set(blockIndex)
	}
}

// IsSet checks if the byte range for the specified position is set.
func (br *ByteRanges) IsSet(pos int64) bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.byteRanges.IsSet(pos)
}

// IsClear checks if the byte range for the specified position is clear.
func (br *ByteRanges) IsClear(pos int64) bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.byteRanges.IsClear(pos)
}

// Notify returns a channel that is closed when the contiguous byte ranges
// starting at the first byte (ie. 0) are extended.
func (br *ByteRanges) Notify() <-chan struct{} {
	br.mu.Lock()
	defer br.mu.Unlock()
	return br.contiguous.Notify()
}

// Tail returns the contiguous byte ranage that starts at the first byte
// and extends to the last contiguous byte from the start that has been set.
// It returns false if no ranges have been set.
func (br *ByteRanges) Tail() (ByteRange, bool) {
	br.mu.RLock()
	blk := br.contiguous.Tail()
	br.mu.RUnlock()
	if blk < 0 {
		return ByteRange{}, false // No ranges have been set.
	}
	to := min(int64(blk+1)*int64(br.blockSize), br.contentSize) - 1
	return ByteRange{
		From: 0,
		To:   to,
	}, true
}

// ByteRangesTracker tracks byte ranges but is not thread safe and does not
// support tracking the contiguous head of the byte ranges.
type ByteRangesTracker struct {
	byteRanges
}

// NewByteRangesTracker creates a new ByteRangesTracker instance with the
// specified content size and block size.
func NewByteRangesTracker(contentSize int64, blockSize int) *ByteRangesTracker {
	nbr := &ByteRangesTracker{
		byteRanges: newByteRanges(contentSize, blockSize),
	}
	return nbr
}

// Clear clears the byte range for the specified position.
func (br *ByteRangesTracker) Clear(pos int64) {
	br.clear(pos)
}
