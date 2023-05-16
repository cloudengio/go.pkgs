// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	"container/heap"
	"testing"

	cheap "cloudeng.io/algo/container/heap"
)

func TestCHeap(t *testing.T) {
	ch := &cheap.CompactUIntHeap{}
	heap.Init(ch)
}

/*
func TestHeap(t *testing.T) {
	var min heap.MinHeap[int, int]

	for i := 0; i < 20; i++ {
		min.Push(i, i*10)
	}
}

/*
func (h *T[V]) verify(t *testing.T, p int) {
	n := h.Len()
	if r := right(p); r < n {
		if h.less(r, p) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", p, h.values[p], r, h.values[r])
			return
		}
		h.verify(t, r)
	}
	if l := left(p); l < n {
		if h.less(l, p) {
			t.Errorf("heap invariant invalidated [%d] = %v > [%d] = %v", p, h.values[p], l, h.values[l])
			return
		}
		h.verify(t, l)
	}
}

func TestInit0(t *testing.T) {
	minh := NewMin(make([]int, 20))
	minh.Heapify()
	minh.verify(t, 0)

	for i := 1; minh.Len() > 0; i++ {
		x := minh.Pop()
		minh.verify(t, 0)
		if got, want := x, 0; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}

func TestInit1(t *testing.T) {
	minh := NewMax(make([]int, 20))
	minh.Heapify()
	minh.verify(t, 0)
	minh = NewMin[int](nil)
	for i := 0; i < 20; i++ {
		minh.Push(i)
	}
	minh.Heapify()
	minh.verify(t, 0)

	for i := 1; minh.Len() > 0; i++ {
		x := minh.Pop()
		minh.verify(t, 0)
		if got, want := x, i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func BenchmarkDup(b *testing.B) {
	const n = 10000
	h := NewMin(make([]int, 0, n))
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			h.Push(0) // all elements are the same
		}
		for h.Len() > 0 {
			h.Pop()
		}
	}
}
*/
