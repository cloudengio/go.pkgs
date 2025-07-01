// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package list

import "iter"

// Double provides a doubly linked list.
type Single[T any] struct {
	sentinel singleItem[T] // sentinel to avoid having to handle head/tail corner cases.
	tail     *singleItem[T]
	len      int
}

type singleItem[T any] struct {
	next *singleItem[T]
	T    T
}

func NewSingle[T any]() *Single[T] {
	dl := &Single[T]{}
	dl.Reset()
	return dl
}

func (dl *Single[T]) Reset() {
	dl.len = 0
	dl.sentinel.next = &dl.sentinel
	dl.tail = &dl.sentinel
}

func (dl *Single[T]) Len() int {
	return dl.len
}

func (dl *Single[T]) Forward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := dl.sentinel.next; n != &dl.sentinel; n = n.next {
			if !yield(n.T) {
				break
			}
		}
	}
}

func (dl *Single[T]) insertAfterItem(val T, it *singleItem[T]) *singleItem[T] {
	n := &singleItem[T]{T: val}
	n.next = it.next
	it.next = n
	dl.len++
	return n
}

func (dl *Single[T]) Head() T {
	if dl.len == 0 {
		return dl.sentinel.T
	}
	return dl.sentinel.next.T
}

func (dl *Single[T]) Append(val T) SingleID[T] {
	n := dl.insertAfterItem(val, dl.tail)
	dl.tail = n
	return n
}

func (dl *Single[T]) Prepend(val T) SingleID[T] {
	n := dl.insertAfterItem(val, &dl.sentinel)
	if dl.len == 1 {
		dl.tail = n
	}
	return n
}

func (dl *Single[T]) removeItem(prev, it *singleItem[T]) {
	dl.len--
	prev.next = it.next
	if dl.tail == it {
		dl.tail = prev
	}
	*it = singleItem[T]{}
}

func (dl *Single[T]) findPrev(it *singleItem[T]) *singleItem[T] {
	prev := &dl.sentinel
	for n := dl.sentinel.next; n != &dl.sentinel; n = n.next {
		if n == it {
			return prev
		}
		prev = n
	}
	return nil
}

type SingleID[T any] *singleItem[T]

func (dl *Single[T]) RemoveItem(id SingleID[T]) {
	if prev := dl.findPrev(id); prev != nil {
		dl.removeItem(prev, id)
	}
}

func (dl *Single[T]) Remove(val T, cmp func(a, b T) bool) {
	prev := &dl.sentinel
	for n := dl.sentinel.next; n != &dl.sentinel; n = n.next {
		if cmp(n.T, val) {
			dl.removeItem(prev, n)
			return
		}
		prev = n
	}
}
