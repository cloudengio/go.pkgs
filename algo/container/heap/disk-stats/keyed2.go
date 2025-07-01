// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package heap contains various implementations of heap containers.
package heap

/*
import (
	"container/heap"
)

type MultiInt struct {
	heap8, heap16, heap32, heap64 heap.Interface
}

func (m *MultiInt) Fix(h heap.Interface, i int) {}
func (m *MultiInt) Init()                       {}

func (m *MultiInt) Pop() any           { return nil }
func (m *MultiInt) Peek() (int64, any) { return 0, nil }

func (m *MultiInt) Push(x any)       {}
func (m *MultiInt) Remove(i int) any { return nil }

type BucketTypes interface {
	~uint8 | ~uint32 | ~uint64
}

type Multi[B BucketTypes] interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
}

func (hb *heapBucket[B]) Len() int { return len(hb.values) }

func (hb *heapBucket[B]) Less(i, j int) bool {
	return hb.ifc.Less(hb.values[i], hb.values[j])
}

func newHeapInterface[B sizeBuckets](ifc Interface) *heapBucket[B] {
	return &heapBucket[B]{ifc: ifc}
}
*/
