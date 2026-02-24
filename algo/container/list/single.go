// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package list //nolint:revive // intentional shadowing

import "iter"

// Single provides a singly linked list.
type Single[T any] struct {
	sentinel singleItem[T] // sentinel to avoid having to handle head/tail corner cases.
	tail     *singleItem[T]
	len      int
}

type singleItem[T any] struct {
	next *singleItem[T]
	T    T
}

// NewSingle creates a new instance of Single[T] with an initial empty state.
func NewSingle[T any]() *Single[T] {
	sl := &Single[T]{}
	sl.Reset()
	return sl
}

// Reset resets the singly linked list to its initial empty state.
func (sl *Single[T]) Reset() {
	sl.len = 0
	sl.sentinel.next = &sl.sentinel
	sl.tail = &sl.sentinel
}

// Len returns the number of items in the singly linked list.
func (sl *Single[T]) Len() int {
	return sl.len
}

// Forward returns an iterator over the list.
func (sl *Single[T]) Forward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := sl.sentinel.next; n != &sl.sentinel; n = n.next {
			if !yield(n.T) {
				break
			}
		}
	}
}

func (sl *Single[T]) insertAfterItem(val T, it *singleItem[T]) *singleItem[T] {
	n := &singleItem[T]{T: val}
	n.next = it.next
	it.next = n
	sl.len++
	return n
}

// Head returns the first item in the list.
func (sl *Single[T]) Head() T {
	if sl.len == 0 {
		return sl.sentinel.T
	}
	return sl.sentinel.next.T
}

// Append adds a new item to the end of the list and returns its ID.
func (sl *Single[T]) Append(val T) SingleID[T] {
	n := sl.insertAfterItem(val, sl.tail)
	sl.tail = n
	return n
}

// Prepend adds a new item to the beginning of the list and returns its ID.
func (sl *Single[T]) Prepend(val T) SingleID[T] {
	n := sl.insertAfterItem(val, &sl.sentinel)
	if sl.len == 1 {
		sl.tail = n
	}
	return n
}

func (sl *Single[T]) removeItem(prev, it *singleItem[T]) {
	sl.len--
	prev.next = it.next
	if sl.tail == it {
		sl.tail = prev
	}
	*it = singleItem[T]{}
}

func (sl *Single[T]) findPrev(it *singleItem[T]) *singleItem[T] {
	prev := &sl.sentinel
	for n := sl.sentinel.next; n != &sl.sentinel; n = n.next {
		if n == it {
			return prev
		}
		prev = n
	}
	return nil
}

type SingleID[T any] *singleItem[T]

// RemoveItem removes the item with the specified ID from the list.
func (sl *Single[T]) RemoveItem(id SingleID[T]) {
	if prev := sl.findPrev(id); prev != nil {
		sl.removeItem(prev, id)
	}
}

// Remove removes the first occurrence of the specified value from the list.
func (sl *Single[T]) Remove(val T, cmp func(a, b T) bool) {
	prev := &sl.sentinel
	for n := sl.sentinel.next; n != &sl.sentinel; n = n.next {
		if cmp(n.T, val) {
			sl.removeItem(prev, n)
			return
		}
		prev = n
	}
}
