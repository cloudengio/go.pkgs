// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package patterns

// BufCap returns the capacity of the ring-buffer backing array.
// Only safe to call after Stop() has returned.
func (b *FIFO[T]) BufCap() int {
	return cap(b.buf)
}

// BufItemCount returns the number of items currently in the buffer.
// Only safe to call after Stop() has returned.
func (b *FIFO[T]) BufItemCount() int {
	return b.count
}
