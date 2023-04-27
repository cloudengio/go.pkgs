// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

// ArithmeticTypes represents the set of types that can be used in a heap that
// keeps a running total of the items it contains. They must be both comparable
// and support addition and subtraction.
type ArithmeticTypes interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

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
