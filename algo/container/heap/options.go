// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

type options[K Ordered, V any] struct {
	sliceCap int
	keys     []K
	vals     []V
	callback func(iv, jv V, i, j int)
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

// WithCallback provides a callback function that is called after every
// operation with the values and indices of the elements that have changed
// location. Note that is not sufficient to track removal of items and hence
// any applications that requires such tracking should do so explicitly.
func WithCallback[K Ordered, V any](fn func(iv, jv V, i, j int)) Option[K, V] {
	return func(o *options[K, V]) {
		o.callback = fn
	}
}
