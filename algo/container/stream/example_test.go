// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stream_test

import (
	"context"
	"fmt"

	"cloudeng.io/algo/container/stream"
)

// lister stands in for a paginated API, such as a listing of files, whose
// results are fetched a page at a time.
type lister struct {
	names []string
	next  int
}

// popn is all that the stream's implementation need provide: it fills items
// with the next names, and says if there are no more.
func (l *lister) popn(_ context.Context, items []string) (int, bool, error) {
	n := copy(items, l.names[l.next:])
	l.next += n
	return n, l.next == len(l.names), nil
}

var names = []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt"}

func ExampleScanner() {
	ctx := context.Background()
	l := &lister{names: names}

	// Batches of two names are requested, and the next batches fetched
	// whilst the names of the current one are being used.
	sc := stream.NewScanner(ctx, l.popn, stream.WithBatchSize(2), stream.WithPrefetch(1))
	defer sc.Close()
	for sc.Scan() {
		fmt.Println(sc.Data())
	}
	if err := sc.Err(); err != nil {
		fmt.Println("error:", err)
	}
	// Output:
	// a.txt
	// b.txt
	// c.txt
	// d.txt
	// e.txt
}

func ExampleIter() {
	ctx := context.Background()
	l := &lister{names: names}

	for name, err := range stream.Iter(ctx, l.popn, stream.WithBatchSize(2)) {
		if err != nil {
			fmt.Println("error:", err)
			break
		}
		fmt.Println(name)
		if name == "c.txt" {
			// Stopping early also stops the prefetching.
			break
		}
	}
	// Output:
	// a.txt
	// b.txt
	// c.txt
}

func ExamplePopNFrom() {
	ctx := context.Background()
	next := 0

	// A stream that can only supply one item at a time is adapted to a PopN
	// to be used with a Scanner or Iter.
	pop := func(context.Context) (string, bool, error) {
		if next == len(names) {
			return "", true, nil
		}
		next++
		return names[next-1], false, nil
	}
	sc := stream.NewScanner(ctx, stream.PopNFrom(pop))
	defer sc.Close()
	for sc.Scan() {
		fmt.Println(sc.Data())
	}
	// Output:
	// a.txt
	// b.txt
	// c.txt
	// d.txt
	// e.txt
}
