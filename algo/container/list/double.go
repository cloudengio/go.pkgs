// Copyright 2024 loudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package list

import (
	"iter"
)

// Double provides a doubly linked list.
type Double[T any] struct {
	sentinel doubleItem[T] // sentinel to avoid having to handle head/tail corner cases.
	len      int
}

type doubleItem[T any] struct {
	next *doubleItem[T]
	prev *doubleItem[T]
	T    T
}

func NewDouble[T any]() *Double[T] {
	dl := &Double[T]{}
	dl.Reset()
	return dl
}

func (dl *Double[T]) Reset() {
	dl.len = 0
	dl.sentinel.next = &dl.sentinel
	dl.sentinel.prev = &dl.sentinel
}

func (dl *Double[T]) Len() int {
	return dl.len
}

func (dl *Double[T]) Forward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := dl.sentinel.next; n != &dl.sentinel; n = n.next {
			if !yield(n.T) {
				break
			}
		}
	}
}

func (dl *Double[T]) Reverse() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := dl.sentinel.prev; n != &dl.sentinel; n = n.prev {
			if !yield(n.T) {
				break
			}
		}
	}
}

func (dl *Double[T]) insertAfterItem(val T, it *doubleItem[T]) *doubleItem[T] {
	n := &doubleItem[T]{T: val}
	n.prev = it
	n.next = it.next
	n.prev.next = n
	n.next.prev = n
	dl.len++
	return n
}

func (dl *Double[T]) Head() T {
	if dl.len == 0 {
		return dl.sentinel.T
	}
	return dl.sentinel.next.T
}

func (dl *Double[T]) Tail() T {
	if dl.len == 0 {
		return dl.sentinel.T
	}
	return dl.sentinel.prev.T
}

func (dl *Double[T]) Append(val T) DoubleID[T] {
	return dl.insertAfterItem(val, dl.sentinel.prev)
}

func (dl *Double[T]) Prepend(val T) DoubleID[T] {
	return dl.insertAfterItem(val, &dl.sentinel)
}

func (dl *Double[T]) removeItem(it *doubleItem[T]) {
	dl.len--
	it.prev.next = it.next
	it.next.prev = it.prev
	*it = doubleItem[T]{}
}

type DoubleID[T any] *doubleItem[T]

func (dl *Double[T]) RemoveItem(id DoubleID[T]) {
	dl.removeItem(id)
}

func (dl *Double[T]) Remove(val T, cmp func(a, b T) bool) {
	for n := dl.sentinel.next; n != nil; n = n.next {
		if cmp(n.T, val) {
			dl.removeItem(n)
			return
		}
	}
}

func (dl *Double[T]) RemoveReverse(val T, cmp func(a, b T) bool) {
	for n := dl.sentinel.prev; n != nil; n = n.prev {
		if cmp(n.T, val) {
			dl.removeItem(n)
			return
		}
	}
}
