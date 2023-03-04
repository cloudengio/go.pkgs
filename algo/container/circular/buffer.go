// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package circular provides 'circular' data structures,
package circular

// Buffer provides a circular buffer that grows as needed.
type Buffer[T any] struct {
	storage []T
	// NOTE, if head==tail then the buffer is empty or full,
	// and used == 0 must be used to distinguish between these two cases.
	used int
	head int // index of the first data element.
	tail int // index of the last data element.
}

// NewBuffer creates a new buffer with the specified initial size.
func NewBuffer[T any](size int) *Buffer[T] {
	if size == 0 {
		size = 1
	}
	return &Buffer[T]{
		storage: make([]T, size),
	}
}

// Len returns the current number of elements in the buffer.
func (b *Buffer[T]) Len() int {
	return b.used
}

// Cap returns the current capacity of the buffer.
func (b *Buffer[T]) Cap() int {
	return cap(b.storage)
}

func (b *Buffer[T]) grow(size int) {
	n := make([]T, size)
	switch {
	case b.head <= b.tail:
		b.tail = copy(n, b.storage[b.head:b.tail+1]) - 1
	default:
		c := copy(n, b.storage[b.head:])
		b.tail = c + copy(n[c:], b.storage[:b.tail+1]) - 1
	}
	b.head = 0
	b.storage = n
}

// Append appends the specified values to the buffer, growing the
// buffer as needed.
func (b *Buffer[T]) Append(v []T) {
	if total := b.used + len(v); total > len(b.storage) {
		b.grow(total)
	}
	switch {
	case b.used == 0:
		// empty
		b.head = 0
		b.tail = copy(b.storage[0:], v) - 1
	case b.head <= b.tail:
		// May need to use two copies to fill the buffer.
		c := copy(b.storage[b.tail+1:], v)
		copy(b.storage[0:], v[c:])
		b.tail = (b.tail + len(v)) % len(b.storage)
	default:
		// wrapped around, only copy is needed.
		copy(b.storage[b.tail+1:], v)
		b.tail += len(v)
	}
	b.used += len(v)
}

// Head returns the first n elements of the buffer, removing them
// from the buffer. If n is greater than the number of elements in
// the buffer then all elements are returned. The values returned
// are not zeroed out and hence if pointers will not be GC'd until
// the buffer itself is released or Compact is called.
func (b *Buffer[T]) Head(n int) []T {
	if n == 0 || b.used == 0 {
		return nil
	}
	if n > b.used {
		n = b.used
	}
	o := make([]T, n)
	if b.head < b.tail {
		copy(o, b.storage[b.head:])
		b.head += n
		b.used -= n
		return o
	}
	c := copy(o, b.storage[b.head:])
	copy(o[c:], b.storage[0:])
	b.head = (b.head + n) % len(b.storage)
	b.used -= n
	return o
}

// Compact reduces the storage used by the buffer to the minimum
// necessary to store its current contents. This also has the effect of
// freeing any pointers that are no longer accessible via the buffer and
// hence may be GC'd.
func (b *Buffer[T]) Compact() {
	if b.used == 0 {
		b.storage = make([]T, 1)
		b.head, b.tail = 0, 0
		return
	}
	n := make([]T, b.used)
	switch {
	case b.head <= b.tail:
		b.tail = copy(n, b.storage[b.head:b.tail+1]) - 1
	default:
		c := copy(n, b.storage[b.head:])
		b.tail = c + copy(n[c:], b.storage[:b.tail+1]) - 1
	}
	b.head = 0
	b.storage = n
}
