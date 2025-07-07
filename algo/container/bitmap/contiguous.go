// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap

// Contiguous supports following the tail of contiguous sub-range of
// a bitmap as it is updated. Clients use the Notify method to obtain
// a channel that is closed whenever the tail of the the tracked contiguous
// sub-range is extended. Contiguous is not thread-safe and callers must
// ensure appropriate synchronization when using it concurrently.
type Contiguous struct {
	bits       T
	last       int
	start      int
	firstClear int
	ch         chan struct{}
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

// NewContiguous creates an instance of Contiguous of the given size
// with the tracked ranged starting at the given 'start' index.
func NewContiguous(start, size int) *Contiguous {
	return newContiguous(New(size), start, size)
}

// NewContiguousWithBitmap is like NewContiguous, but is initialized with
// the supplied bitmap. The bitmap is not copied. Updates must be made
// to the bitmap using the Set method of the Contiguous type.
func NewContiguousWithBitmap(bm T, start, size int) *Contiguous {
	if size > len(bm)*64 {
		return nil
	}
	return newContiguous(bm, start, size)
}

// Notify can be used to notify the caller of an extension in the contiguous
// sub-range of the bitmap. It returns a channel that will be closed whenever
// the tail of the contiguous sub-range is extended. Closing a channel
// is used as the notification mechanism (rather than sending updates
// on the channel) because it allows for multiple listeners and avoids
// the need for any synchronization between the contiguous bitmap
// implementation and the listeners. Expected usage is of the form:
//
//	ch := cb.Notify()
//	<-ch
//	end := cb.Tail()
//
// Note that if the range has already reached the end of the bitmap, the
// returned channel will have already been closed.
func (c *Contiguous) Notify() <-chan struct{} {
	if c.ch != nil {
		return c.ch
	}
	c.ch = make(chan struct{})
	if c.firstClear > c.last {
		close(c.ch)
	}
	return c.ch
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
	if c.ch == nil || c.firstClear <= c.start {
		return
	}
	close(c.ch)
	c.ch = nil
}

// Tail returns the last index in the contiguous subrange that has been set,
// or -1 if no bits have been set. The value of the returned value will be
// that of the last index in the bitmap if entire range has been set.
func (c *Contiguous) Tail() int {
	if c.firstClear <= c.start {
		return -1
	}
	return c.firstClear - 1
}
