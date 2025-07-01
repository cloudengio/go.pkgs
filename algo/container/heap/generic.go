// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

// Value represents the interface that must be implemented by types that
// can be used as values in the generic Heap implementation.
// It requires a Less method that compares the current instance with another
// instance of the same type and returns true if the current instance is less
// than the other instance.
type Value[T any] interface {
	Less(x T) bool
}

// Heap is a generic heap implementation that can be used with any type
// that implements the Value interface. It provides methods to push, pop,
// remove, and fix elements in the heap. The heap is implemented as a slice
// in the same manner as the standard library's heap package, but it is generic
// and can work with any type that satisfies the Value interface.
type Heap[T Value[T]] []T

func (h Heap[T]) Len() int {
	return len(h)
}

func (h Heap[T]) Init() {
	// heapify
	n := len(h)
	for i := n/2 - 1; i >= 0; i-- {
		down(h, i, n)
	}
}

// Push is like heap.Push.
func (h *Heap[T]) Push(x T) {
	*h = append(*h, x)
	up(*h, len(*h)-1)
}

// Pop is like heap.Pop.
func (h *Heap[T]) Pop() T {
	n := len(*h) - 1
	swap(*h, 0, n)
	down(*h, 0, n)
	t, x := pop(*h)
	*h = t
	return x
}

// Remove is like heap.Remove.
func (h *Heap[T]) Remove(i int) any {
	n := len(*h) - 1
	if n != i {
		swap(*h, i, n)
		if !down(*h, i, n) {
			up(*h, i)
		}
	}
	t, x := pop(*h)
	*h = t
	return x
}

// Fix is like heap.Fix.
func Fix[T Value[T]](h []T, i int) {
	if !down(h, i, len(h)) {
		up(h, i)
	}
}

func up[T Value[T]](h []T, j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !less(h, j, i) {
			break
		}
		swap(h, i, j)
		j = i
	}
}

func down[T Value[T]](h []T, i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && less(h, j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !less(h, j, i) {
			break
		}
		swap(h, i, j)
		i = j
	}
	return i > i0
}

func swap[T Value[T]](h []T, i, j int) {
	h[i], h[j] = h[j], h[i]
}

func less[T Value[T]](h []T, i, j int) bool {
	return h[i].Less(h[j])
}

func pop[T Value[T]](h []T) ([]T, T) {
	old := h
	n := len(old)
	x := old[n-1]
	h = old[0 : n-1]
	return h, x
}
