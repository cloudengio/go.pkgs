// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap

// Contiguous supports following the tail of contiguous sub-range of
// a bitmap as it is being updated.
type Contiguous struct {
	bits       T
	last       int
	start      int
	firstClear int
	ch         chan<- int
}

func newContiguous(bm T, start, size int) *Contiguous {
	if size <= 0 || start < 0 || start > size {
		return nil
	}
	c := &Contiguous{
		bits:       bm,
		last:       min(len(bm)*64, size) - 1,
		start:      start,
		firstClear: start,
	}
	c.extend(start)
	return c
}

// NewContiguous creates a new Contiguous instance that tracks a sub-range
// of a bitmap of the given size (in bits) starting at the given index.
// As the tail of the sub-range is extended, updates are sent on the
// provided channel `ch`. The updates specify the index of the last bit
// that is set in the contiguous range. The channel is closed when the
// tail of the sub-range extends beyond the last index of the bitmap.
// If size is less than or equal to zero, or start is negative, it returns nil.
func NewContiguous(start, size int) *Contiguous {
	return newContiguous(New(size), start, size)
}

// SetCh sets the channel on which updates are sent. If the channel is nil,
// no updates will be sent. If the channel is set, an update is sent immediately
// if the first clear bit is beyond the start index. The channel is closed
// when all bits in the sub-range have been set. It is up to the caller
// to ensure that the channel is deep enough and read frequently enough
// to avoid blocking the sender.
func (c *Contiguous) SetCh(ch chan<- int) *Contiguous {
	if c == nil {
		return nil
	}
	c.ch = ch
	c.sendUpdate()
	return c
}

// NewContiguousWithBitmap is like NewContiguous, but is initialized with
// the supplied bitmap. The bitmap is not copied. Updates must be made
// to the bitmap using the Set method of the Contiguous type. If the supplied
// bitmap has set bits that overlap with the specified start index an update
// will be sent on the channel immediately.
func NewContiguousWithBitmap(bm T, start, size int) *Contiguous {
	if size > len(bm)*64 {
		return nil
	}
	return newContiguous(bm, start, size)
}

func (c *Contiguous) extend(start int) {
	for nb := start; nb <= c.last; nb++ {
		if c.bits.IsSetUnsafe(nb) {
			c.firstClear++
			continue
		}
		break
	}
}

func (c *Contiguous) sendUpdate() {
	if c.ch == nil || c.firstClear <= c.start {
		return
	}
	c.ch <- c.firstClear - 1
	if c.firstClear > c.last {
		close(c.ch)
	}
}

// Set sets the bit at index i in the bitmap to 1. If i is out of bounds,
// the function does nothing.
func (c *Contiguous) Set(i int) {
	if i < 0 || i > c.last {
		return
	}
	c.bits.SetUnsafe(i)
	if i != c.firstClear {
		return
	}
	c.extend(c.firstClear)
	c.sendUpdate()
}

// Last returns the last index in the bitmap subrange that has been set,
// or -1 if no bits have been set.
func (c *Contiguous) LastSet() int {
	if c.firstClear <= c.start {
		return -1
	}
	return c.firstClear - 1
}
