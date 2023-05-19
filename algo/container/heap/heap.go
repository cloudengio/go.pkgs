// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

import "fmt"

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | string
}

func NewMin[K Ordered, V any]() *T[K, V] {
	h := newT[K, V](0)
	h.less = func(a, b K) bool { return a < b }
	return h
}

func NewMax[K Ordered, V any]() *T[K, V] {
	h := newT[K, V](0)
	h.less = func(a, b K) bool { return a > b }
	return h
}

func newT[K Ordered, V any](size int) *T[K, V] {
	return &T[K, V]{
		Keys: make([]K, 0, size),
		Vals: make([]V, 0, size),
	}
}

type T[K Ordered, V any] struct {
	Keys []K
	Vals []V
	less func(a, b K) bool
}

func (h *T[K, V]) Len() int {
	return len(h.Keys)
}

func (h *T[K, V]) Push(k K, v V) {
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	h.siftUp(len(h.Keys) - 1)
}

func swap[K Ordered, V any](keys []K, vals []V, i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
	vals[i], vals[j] = vals[j], vals[i]
}

func (h *T[K, V]) siftUp(i int) int {
	for {
		p := parent(i)
		if i == p || h.less(h.Keys[p], h.Keys[i]) {
			// Special case duplicates?
			return i
		}
		swap(h.Keys, h.Vals, p, i)
		i = p
	}
}

func (h *T[K, V]) Pop() (K, V) {
	k, v := h.Keys[0], h.Vals[0]
	n := len(h.Keys) - 1
	swap(h.Keys, h.Vals, 0, n)
	h.siftDown(0)
	// pop must come last so that there is room to move the last key all
	// of the way back down to where it came from - ie. the special case
	// where the last key needs to be sifted down to the exact same spot
	// it came from.
	h.Keys = h.Keys[:n]
	h.Vals = h.Vals[:n]
	return k, v
}

func (h *T[K, V]) siftDown(parent int) bool {
	p := parent
	n := len(h.Keys) - 1
	for {
		l := left(p)
		if l >= n || l < 0 {
			break
		}
		// If there are two subtrees to choose from, pick the "smaller"
		// to compare against the value being sifted down.
		s := l
		if r := right(p); r < n && h.less(h.Keys[r], h.Keys[l]) {
			s = r
		}
		if !h.less(h.Keys[s], h.Keys[p]) {
			// Neither subtree is "smaller", so we're done.
			break
		}
		swap(h.Keys, h.Vals, p, s)
		p = s
	}
	return p > parent
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (2 * i) + 1 }
func right(i int) int  { return (2 * i) + 2 }

type Bounded[K Ordered, V any] struct {
	*T[K, V]
	n        int
	leastKey K
	leastPos int
}

func newBounded[K Ordered, V any](size, n int) *Bounded[K, V] {
	return &Bounded[K, V]{
		T: &T[K, V]{
			Keys: make([]K, 0, size),
			Vals: make([]V, 0, size),
		},
		n: n,
	}
}

func NewMinBounded[K Ordered, V any](n int) *Bounded[K, V] {
	h := newBounded[K, V](0, n)
	h.less = func(a, b K) bool { return a < b }
	return h
}

func NewMaxBounded[K Ordered, V any](n int) *Bounded[K, V] {
	h := newBounded[K, V](0, n)
	h.less = func(a, b K) bool { return a > b }
	return h
}

func (h *Bounded[K, V]) swap(i, j int) {
	h.Keys[i], h.Keys[j] = h.Keys[j], h.Keys[i]
	h.Vals[i], h.Vals[j] = h.Vals[j], h.Vals[i]
	if j == h.leastPos {
		h.leastPos = i
		panic("x")
	}
}

func (h *Bounded[K, V]) Push(k K, v V) {
	switch {
	case len(h.Keys) == 0:
		h.leastKey = k
		h.leastPos = 0
	case len(h.Keys) >= h.n:
		// Heap is full.
		if h.less(h.leastKey, k) {
			// Have a new 'least' key.
			fmt.Printf("N0: %v at %v for %v: %v\n", k, h.leastPos, h.leastKey, h.Keys)
			h.Keys[h.leastPos] = k
			h.Vals[h.leastPos] = v
			fmt.Printf("N: %v at %v for %v\n", k, h.leastPos, h.leastKey)
			h.leastKey = k
			fmt.Printf("%v\n", h.Keys)
		}
		return
	}
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	at := h.siftUp(len(h.Keys) - 1)
	if len(h.Keys) > 1 && h.less(h.leastKey, k) {
		fmt.Printf("L: %v < %v, at: %v (%v): %v\n", k, h.leastKey, at, len(h.Keys), h.Keys)
		h.leastKey = k
		h.leastPos = at
	}
}
