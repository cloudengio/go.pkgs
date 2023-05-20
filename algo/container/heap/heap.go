// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

import "fmt"

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | string
}

func swap[K Ordered, V any](keys []K, vals []V, i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
	vals[i], vals[j] = vals[j], vals[i]
}

func NewMin[K Ordered, V any]() *T[K, V] {
	h := newT[K, V](0)
	h.ops.less = func(a, b K) bool { return a < b }
	return h
}

func NewMax[K Ordered, V any]() *T[K, V] {
	h := newT[K, V](0)
	h.ops.less = func(a, b K) bool { return a > b }
	return h
}

func newT[K Ordered, V any](size int) *T[K, V] {
	n := &T[K, V]{
		Keys: make([]K, 0, size),
		Vals: make([]V, 0, size),
	}
	n.ops.swap = swap[K, V]
	return n
}

type T[K Ordered, V any] struct {
	Keys []K
	Vals []V
	ops  operations[K, V]
}

func (h *T[K, V]) Len() int {
	return len(h.Keys)
}

func (h *T[K, V]) Push(k K, v V) {
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	h.ops.siftUp(h.Keys, h.Vals, len(h.Keys)-1)
}

func (h *T[K, V]) Pop() (k K, v V) {
	k, v, h.Keys, h.Vals = h.ops.pop(h.Keys, h.Vals)
	return
}

type operations[K Ordered, V any] struct {
	less func(a, b K) bool
	swap func(keys []K, vals []V, i, j int)
}

func (o operations[K, V]) siftUp(keys []K, vals []V, i int) int {
	for {
		p := parent(i)
		if i == p || o.less(keys[p], keys[i]) {
			// Special case duplicates?
			return i
		}
		o.swap(keys, vals, p, i)
		i = p
	}
}

func (o operations[K, V]) minChild(keys []K, i int) int {
	n := len(keys) - 1
	l, r := left(i), right(i)
	if l >= n {
		return -1
	}
	if r >= n {
		return l
	}
	if o.less(keys[l], keys[r]) {
		return l
	}
	return r
}

func (o operations[K, V]) siftDown(keys []K, vals []V, parent int) bool {
	p := parent
	//	n := len(keys) - 1
	for {
		/*		l := left(p)
				if l >= n || l < 0 {
					break
				}*/
		mc := o.minChild(keys, p)
		if mc == -1 {
			break
		}
		// If there are two subtrees to choose from, pick the "smaller"
		// to compare against the value being sifted down.
		/*s := l
		if r := right(p); r < n && o.less(keys[r], keys[l]) {
			s = r
		}*/
		if !o.less(keys[mc], keys[p]) {
			// Neither subtree is "smaller", so we're done.
			break
		}
		swap(keys, vals, p, mc)
		p = mc
	}
	return p > parent
}

/*
func (o operations[K, V]) last(keys []K, l i int) int {
	for {
		mc := o.minChild(keys, i)
		if mc == -1 {
			return i
		}
		i = mc
	}
}*/

func (o operations[K, V]) pop(keys []K, vals []V) (K, V, []K, []V) {
	k, v := keys[0], vals[0]
	n := len(keys) - 1
	swap(keys, vals, 0, n)
	o.siftDown(keys, vals, 0)
	// pop must come last so that there is room to move the last key all
	// of the way back down to where it came from - ie. the special case
	// where the last key needs to be sifted down to the exact same spot
	// it came from.
	return k, v, keys[:n], vals[:n]
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (2 * i) + 1 }
func right(i int) int  { return (2 * i) + 2 }

type Bounded[K Ordered, V any] struct {
	Keys      []K
	Vals      []V
	ops       operations[K, V]
	n         int
	lowestKey K
	lowestPos int
}

func newBounded[K Ordered, V any](size, n int) *Bounded[K, V] {
	b := &Bounded[K, V]{
		Keys: make([]K, 0, size),
		Vals: make([]V, 0, size),
		n:    n,
	}
	b.ops.swap = func(keys []K, vals []V, i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
		vals[i], vals[j] = vals[j], vals[i]
		if i == b.lowestPos {
			fmt.Printf("leastPos: I swap: %v: %v to %v: %v\n", b.lowestKey, i, j, b.Keys)
			b.lowestPos = j
			return
		}
		if j == b.lowestPos {
			fmt.Printf("leastPos: J swap: %v: %v to %v: %v\n", b.lowestKey, j, i, b.Keys)
			b.lowestPos = i
		}
	}
	return b
}

func NewMinBounded[K Ordered, V any](n int) *Bounded[K, V] {
	h := newBounded[K, V](0, n)
	h.ops.less = func(a, b K) bool { return a < b }
	return h
}

func NewMaxBounded[K Ordered, V any](n int) *Bounded[K, V] {
	h := newBounded[K, V](0, n)
	h.ops.less = func(a, b K) bool { return a > b }
	return h
}

func (h *Bounded[K, V]) Len() int {
	return len(h.Keys)
}

func (h *Bounded[K, V]) Pop() (k K, v V) {
	k, v, h.Keys, h.Vals = h.ops.pop(h.Keys, h.Vals)
	return
}

// But you can make this branchless by calculating the child index as (2*i)+1+(child1<child2), and then this will be fast.

func (h *Bounded[K, V]) Push(k K, v V) {
	switch {
	case len(h.Keys) >= h.n:
		fmt.Printf("push: full: %v: %v (%v at %v)\n", k, h.Keys, h.lowestKey, h.lowestPos)
		// Heap is full.
		if h.ops.less(k, h.lowestKey) {
			fmt.Printf("new least: %v < %v\n", k, h.lowestKey)
			// Replace current least key.
			h.Keys[h.lowestPos] = k
			h.Vals[h.lowestPos] = v
			//at := h.ops.siftUp(h.Keys, h.Vals, h.lowestPos)
			//h.lowestPos = h.ops.last(h.Keys, at)
			fmt.Printf("NL: %v\n", h.lowestPos)
			h.lowestKey = h.Keys[h.lowestPos]
			//

			//--- heapify ---

			//find new least key & pos
			//			-- heapify

			//h.lowestKey = k
			//			.... err what about leastPos
		}
		return
	case len(h.Keys) == 0:
		h.lowestKey = k
		h.lowestPos = 0
	}
	fmt.Printf("push: %v: %v (%v at %v)\n", k, h.Keys, h.lowestKey, h.lowestPos)
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	at := h.ops.siftUp(h.Keys, h.Vals, len(h.Keys)-1)
	if len(h.Keys) > 1 && h.ops.less(h.lowestKey, k) {
		h.lowestKey = k
		h.lowestPos = at
	}
	fmt.Printf("push: done: %v: %v (%v at %v)\n", k, h.Keys, h.lowestKey, h.lowestPos)
}
