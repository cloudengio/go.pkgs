// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package heap contains various implementations of heap containers.
package heap_test

import "cloudeng.io/algo/container/heap"

func NewNumericIndexedExample() {

	idx := heap.MapIndex[string]{}
	ni := heap.NewNumericIndexed[uint16, string](heap.Descending, idx)
	_ = ni

}
