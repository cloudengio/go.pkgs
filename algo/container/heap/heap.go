// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

/*
// Mirror heap.Interface but using type constraints rather than runtime
// types.
type Interface[T any] interface {
	sort.Interface
	Push(T)
	Pop() T
}

// Heapify reorders the elements of h into a heap.
func Heapify[T any](h Interface[T]) {
	n := h.Len()
	for i := n/2 - 1; i >= 0; i-- {
		siftDown(h, i, n)
	}
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (2 * i) + 1 }
func right(i int) int  { return (2 * i) + 2 }

func siftDown[T any](h Interface[T], p, n int) bool {
	i := p
	for {
		l := left(i)
		if l >= n || l < 0 { // l < 0 guards against integer overflow
			break
		}
		// chose the smaller of the left or right subtree.
		t := l
		if r := right(i); r < n && h.Less(r, l) {
			t = r
		}
		if !h.Less(t, i) {
			break
		}
		h.Swap(i, t)
		i = t
	}
	return i > p
}

/*
type Ordered interface {
	~string | ~byte | ~int8 | ~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64 | ~float32 | ~float64
}

type T[V Ordered] struct {
	values []V
	max    bool
}

// options:
// how much space to waste - cap() - len()
// dups

func newHeap[V Ordered](values []V, max bool) *T[V] {
	if values == nil {
		values = make([]V, 0)
	}
	return &T[V]{
		values: values[:0],
		max:    max,
	}
}

func NewMin[V Ordered](values []V) *T[V] {
	return newHeap(values, false)
}

func NewMax[V Ordered](values []V) *T[V] {
	return newHeap(values, true)
}

func (h *T[V]) Heapify() {
	h.heapify(0)
}

func (h *T[V]) Len() int { return len(h.values) }

func (h *T[V]) Cap() int { return cap(h.values) }

func (h *T[V]) Push(v V) {
	l := len(h.values)
	h.values = append(h.values, v)
	h.siftUp(l)
}

func (h *T[V]) Peek() V {
	return h.values[0]
}

func (h *T[V]) Pop() V {
	v := h.values[0]
	n := h.Len() - 1
	h.swap(0, n)
	h.siftDown(0)
	h.values = h.values[0 : n-1]
	return v
}

func (h *T[V]) Remove(i int) V {
	n := h.Len() - 1
	v := h.values[i]
	if n == i {
		h.values = h.values[0 : n-1]
		return v
	}
	h.swap(i, n)
	if !h.siftDown(i) {
		h.siftUp(i)
	}
	h.values = h.values[0 : len(h.values)-1]
	return v
}

func (h *T[V]) swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
}

func (h *T[V]) less(i, j int) bool {
	if h.max {
		return h.values[i] > h.values[j]
	}
	return h.values[i] < h.values[j]
}

func (h *T[V]) heapify(i int) {
	n := h.Len()
	for i := n/2 - 1; i > 0; i-- {
		h.siftDown(i)
	}
}



func (h *T[V]) siftUp(i int) {
	for {
		p := parent(i)
		if i == p || h.less(p, i) {
			//if h.values[p] == h.values[i] {
			//	fmt.Printf("duplicate: %v\n", h.values[p])
			//}
			break
		}
		h.swap(p, i)
		i = p
	}
}

func (h *T[V]) siftDown(p int) bool {
	i := p
	n := h.Len() - 1
	for {
		l := left(i)
		if l >= n || l < 0 { // overflow
			break
		}
		// chose either the left or right sub-tree, depending
		// on which is smaller.
		t := l
		if r := right(i); r < n && h.less(r, l) {
			t = r
		}
		if !h.less(t, i) {
			break
		}
		h.swap(i, t)
		i = t
	}
	return i > p
}

/*

// dups...

type Keyed[V comparable, D any] struct {
	values []V
	data   []D
}

func (h *Keyed[V, D]) Len() int { return len(h.data) }

func (h *Keyed[V, D]) Push(v V, d D) {
	h.values = append(h.values, v)
	h.data = append(h.data, d)

	// h.up(h.Len() - 1)
}

func (h *Keyed[V, D]) Pop() (V, D) {
	n := h.Len() - 1
	if n > 0 {
		//h.swap(0, n)
		//h.down()
	}
	v := h.values[n]
	d := h.data[n]
	h.values = h.values[0:n]
	h.data = h.data[0:n]
	return v, d
}

func (h *Keyed[V, D]) Peek() (V, D) {
	return h.values[0], h.data[0]
}

func (h *Keyed[V, D]) PeekN(n int) ([]V, []D) {
	vo := make([]V, n)
	do := make([]D, n)

	vo[0], do[0] = h.values[0], h.data[0]

	return vo, do
}

/*
type MapIndex[T comparable] map[T]int

func (mi MapIndex[T]) Insert(k T, v int) {
	mi[k] = v
}

func (mi MapIndex[T]) Lookup(k T) int {
	return mi[k]
}

type Index[T comparable] interface {
	Encode(T) int64
	Insert(k T, v int)
	Lookup(k T) (v int)
}

type Numeric[ValueT ArithmeticTypes, IndexT comparable] struct {
	order  Order
	total  ValueT
	values []ValueT
	index  Index[IndexT]
}

func NewNumericIndexed[ValueT ArithmeticTypes, IndexT comparable](order Order, index Index[IndexT]) *Numeric[ValueT, IndexT] {
	return &Numeric[ValueT, IndexT]{
		order:  order,
		values: make([]ValueT, 0),
		index:  index,
	}
}

/*
func NewNumeric[ValueT NumericTypes, DataT any](order Order) *Numeric[ValueT, DataT] {
	return &Heap[ValueT, DataT]{
		order:  order,
		values: make([]ValueT, 0),
		data:   make([]DataT, 0),
	}
}

func (h *Heap[V, D]) swap(i, j int) {
	h.values[i], h.values[j] = h.values[j], h.values[i]
	h.data[i], h.data[j] = h.data[j], h.data[i]
}

func (h *Heap[V, D]) Len() int { return len(h.data) }

func (h *Heap[V, D]) Push(v V, d D) {
	h.total += v
	h.values = append(h.values, v)
	h.data = append(h.data, d)
	h.up(h.Len() - 1)
}

func (h *Heap[V, D]) Pop() (V, D) {
	n := h.Len() - 1
	if n > 0 {
		h.swap(0, n)
		h.down()
	}
	v := h.values[n]
	d := h.data[n]
	h.values = h.values[0:n]
	h.data = h.data[0:n]
	return v, d
}

func (h *Heap[V, D]) Peek() (V, D) {
	return h.values[0], h.data[0]
}

func (h *Heap[V, D]) PeekN(n int) (V, D) {
	return h.values[0], h.data[0]
}

func (h *Heap[V, D]) up(jj int) {
	for {
		i := parent(jj)
		if i == jj || !h.comp(h.data[jj], h.data[i]) {
			break
		}
		h.swap(i, jj)
		jj = i
	}
}

func (h *Heap[V, D]) down() {
	n := h.Len() - 1
	i1 := 0
	for {
		j1 := left(i1)
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		j2 := right(i1)
		if j2 < n && h.comp(h.data[j2], h.data[j1]) {
			j = j2
		}
		if !h.comp(h.data[j], h.data[i1]) {
			break
		}
		h.swap(i1, j)
		i1 = j
	}
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return (i * 2) + 1 }
func right(i int) int  { return left(i) + 1 }
*/
