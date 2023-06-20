// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

// Orderded represents the set of types that can be used as keys in a heap.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | string
}

type options[K Ordered, V any] struct {
	sliceCap int
	keys     []K
	vals     []V
}

// Option represents the options that can be passed to NewMin and NewMax.
type Option[K Ordered, V any] func(*options[K, V])

// WithSliceCap sets the initial capacity of the slices used to hold keys
// and values.
func WithSliceCap[K Ordered, V any](n int) Option[K, V] {
	return func(o *options[K, V]) {
		o.sliceCap = n
	}
}

// WithData sets the initial data for the heap.
func WithData[K Ordered, V any](keys []K, vals []V) Option[K, V] {
	return func(o *options[K, V]) {
		if len(keys) != len(vals) {
			panic("keys and vals must be the same length")
		}
		o.keys = keys
		o.vals = vals
	}
}

// NewMin creates a new heap with ascending order.
func NewMin[K Ordered, V any](opts ...Option[K, V]) *T[K, V] {
	return newT(false, opts)
}

// NewMax creates a new heap with descending order.
func NewMax[K Ordered, V any](opts ...Option[K, V]) *T[K, V] {
	return newT(true, opts)
}

func newT[K Ordered, V any](max bool, opts []Option[K, V]) *T[K, V] {
	var o options[K, V]
	for _, fn := range opts {
		fn(&o)
	}
	if o.keys != nil && o.vals != nil {
		h := &T[K, V]{
			Keys: o.keys,
			Vals: o.vals,
			max:  max,
		}
		h.heapify()
		return h
	}

	n := &T[K, V]{
		Keys: make([]K, 0, o.sliceCap),
		Vals: make([]V, 0, o.sliceCap),
		max:  max,
	}
	return n
}

// T represents a heap of keys and values.
type T[K Ordered, V any] struct {
	Keys []K
	Vals []V
	max  bool
}

func (h *T[K, V]) heapify() {
	n := len(h.Keys)
	// Use Floyd's algorithm, start at the mid point since the bottom half must
	// all be leaf nodes and hence are already heaps.
	for i := n/2 - 1; i >= 0; i-- {
		h.siftDown(i, n)
	}
}

func (h *T[K, V]) less(i, j int) bool {
	if h.max {
		return h.Keys[i] > h.Keys[j]
	}
	return h.Keys[i] < h.Keys[j]
}

// Len returns the number of elements in the heap.
func (h *T[K, V]) Len() int {
	return len(h.Keys)
}

// Push adds a new key and value to the heap.
func (h *T[K, V]) Push(k K, v V) {
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	h.siftUp(len(h.Keys) - 1)
}

// Pop removes and returns the top element from the heap.
func (h *T[K, V]) Pop() (K, V) {
	k, v := h.Keys[0], h.Vals[0]
	n := len(h.Keys) - 1
	h.set(0, n)
	h.siftDown(0, n)
	// pop must come last so that there is room to move the last key all
	// of the way back down to where it came from - ie. the special case
	// where the last key needs to be sifted down to the exact same spot
	// it came from.
	h.Keys, h.Vals = h.Keys[:n], h.Vals[:n]
	return k, v
}

// Remove removes the i'th element from the heap.
func (h *T[K, V]) Remove(i int) (K, V) {
	n := h.Len() - 1
	if n != i {
		h.swap(i, n)
		if !h.siftDown(i, n) {
			h.siftUp(i)
		}
	}
	k, v := h.Keys[n], h.Vals[n]
	h.Keys, h.Vals = h.Keys[:n], h.Vals[:n]
	return k, v
}

// Update updates the key and value for the i'th element in the
// heap. It is more efficient than Remove followed by Push.
func (h *T[K, V]) Update(pos int, k K, v V) {
	h.Keys[pos] = k
	h.Vals[pos] = v
	if !h.siftDown(pos, len(h.Keys)) {
		h.siftUp(pos)
	}
}

func (h *T[K, V]) swap(i, j int) {
	h.Keys[i], h.Keys[j] = h.Keys[j], h.Keys[i]
	h.Vals[i], h.Vals[j] = h.Vals[j], h.Vals[i]
}

func (h *T[K, V]) set(i, j int) {
	h.Keys[i] = h.Keys[j]
	h.Vals[i] = h.Vals[j]
}

func (h *T[K, V]) siftUp(from int) {
	i := from
	for {
		p := (i - 1) / 2 // parent
		if (p == i) || !h.less(i, p) {
			// The test above ensures that a duplicate key is left
			// at the last position in a run of deaps rather than
			// being pointlessly moved to the head of that run.
			// Consider an option for not keeping dups.
			break
		}
		h.swap(p, i)
		i = p
	}
}

// siftDown returns true if the value was moved down the heap, returning
// false means that it should be moved up the heap.
func (h *T[K, V]) siftDown(parent, limit int) bool {
	p := parent
	for {
		c := (p * 2) + 1 // left child
		if c >= limit {
			break
		}
		// If there are two subtrees to choose from, pick the "smaller"
		// to compare against the value being sifted down.
		if r := c + 1; r < limit && h.less(r, c) {
			c = r
		}
		if !h.less(c, p) {
			// Neither subtree is "smaller", so we're done.
			break
		}
		h.swap(p, c)
		p = c
	}
	return p > parent
}
