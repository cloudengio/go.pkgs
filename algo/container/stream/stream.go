// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package stream provides generic building blocks for scanning and iterating
// over streams of items, such as the results of a paginated API, the rows of a
// query or the entries of a directory. A stream is implemented by providing
// a Pop or PopN function, and the Scanner and Iter built on it then fetch
// items in batches, optionally ahead of their consumption, so that the
// implementation need only say how to obtain the next items.
package stream

import "context"

// Pop returns the next item of a stream. The item is only valid if empty is
// false and err is nil; empty is true when the stream has no more items.
type Pop[T any] func(ctx context.Context) (item T, empty bool, err error)

// PopN fills items, from the start, with as many as len(items) of the next
// items of a stream, and returns the number that it stored in n. Empty is true
// if the stream has no more items beyond those n, which may be zero, so that an
// implementation that finds the end of the stream whilst filling items
// need not be called again. A PopN must make progress: it must not return zero
// items, with empty false and no error, since that is a stream that never ends.
// If err is not nil the n items are still valid, but the stream is over.
type PopN[T any] func(ctx context.Context, items []T) (n int, empty bool, err error)

// PopNFrom returns a PopN that is implemented using pop, for streams that
// can only supply one item at a time.
func PopNFrom[T any](pop Pop[T]) PopN[T] {
	return func(ctx context.Context, items []T) (int, bool, error) {
		for n := range items {
			item, empty, err := pop(ctx)
			if err != nil || empty {
				return n, empty, err
			}
			items[n] = item
		}
		return len(items), false, nil
	}
}
