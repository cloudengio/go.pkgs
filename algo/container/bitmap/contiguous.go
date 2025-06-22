// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap

// Contiguous supports following the tail of contiguous sub-range of
// a bitmap as it is being updated.
type Contiguous struct {
	bits     T
	last     int
	next     int // next is the first bit that needs to be set to extend the contiguous sub-range.
	notified int // notified is the last value sent on the channel.
	ch       chan int
}

// NewContiguous creates a new Contiguous instance that tracks a sub-range
// of a bitmap of the given size (in bits) starting at the given index.
// If size is less than or equal to zero, or start is negative, it returns nil.
func NewContiguous(size, start int) *Contiguous {
	if size <= 0 || start < 0 || start >= size {
		return nil
	}
	bm := New(size)
	return &Contiguous{
		bits:     bm,
		last:     min(len(bm)*64, size),
		next:     start,
		notified: -1, // no notification has been sent yet.
	}
}

// NewContiguousWithBitmap creates a new Contiguous instance that tracks a sub-range
// of the provided bitmap `bm` of the given size (in bits) starting at the
// given index. If `bm` is nil, or start is negative, or size is less than or
// equal to start, it returns nil. The supplied bitmap is modified directly
// and is not copied.
func NewContiguousWithBitmap(bm T, size, start int) *Contiguous {

	if bm == nil || start < 0 || size > len(bm)*64 {
		return nil
	}
	last := min(len(bm)*64, size)
	if start >= last {
		return nil
	}
	c := &Contiguous{
		bits:     bm,
		last:     last,
		next:     start,
		notified: -1, // no notification has been sent yet.
	}
	if nb := c.extend(); nb > c.next {
		c.next = nb
	}
	return c
}

func (c *Contiguous) extend() int {
	nb := c.next
	for ; nb < c.last; nb++ {
		if !c.bits.IsSetUnsafe(nb) {
			break
		}
	}
	return nb
}

// Set sets the bit at index i in the bitmap to 1. If i is out of bounds,
// the function does nothing.
func (c *Contiguous) Set(i int) {
	if i < 0 || i >= c.last {
		return
	}
	c.bits.SetUnsafe(i)
	if i < c.next {
		return
	}
	if nb := c.extend(); nb > c.next {
		if c.ch != nil {
			c.ch <- nb
			c.notified = nb
			close(c.ch)
			c.ch = nil
		}
		c.next = nb
	}
}

// Next returns a channel that will receive the next update to the tail
// of the sub-range being tracked. The channel will be closed immediately
// after sending that update. Only one outstanding call to Next should be
// If the sub-range is already exhausted
// (i.e., `next` is greater than or equal to `last`), it returns a channel
// that immediately sends -1 and then closes.
func (c *Contiguous) Next() <-chan int {
	if c.ch != nil {
		return c.ch
	}
	ch := make(chan int, 1)
	if c.next >= c.last {
		ch <- -1
		close(ch)
		return ch
	}
	c.ch = ch
	return c.ch
}
