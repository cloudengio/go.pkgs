// Copyright 2025 loudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package list_test

import (
	"slices"
	"testing"

	"cloudeng.io/algo/container/list"
)

func forwardS[T any](sl *list.Single[T]) []T {
	var res []T
	for g := range sl.Forward() {
		res = append(res, g)
	}
	return res
}

func testSL[T comparable](t *testing.T, sl *list.Single[T], fwd []T) {
	t.Helper()
	if got, want := forwardS(sl), fwd; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := sl.Len(), len(fwd); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(fwd) > 0 {
		if got, want := sl.Head(), fwd[0]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestSL(t *testing.T) {
	sl := list.NewSingle[int]()
	testSL(t, sl, []int{})
	if got, want := sl.Head(), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	sl.Append(1)
	testSL(t, sl, []int{1})
	sl.Append(2)
	testSL(t, sl, []int{1, 2})
	sl.Append(3)
	testSL(t, sl, []int{1, 2, 3})
	i4 := sl.Append(4)
	sl.Append(50)
	sl.Append(6)
	testSL(t, sl, []int{1, 2, 3, 4, 50, 6})

	cmp := func(a, b int) bool {
		return a == b
	}
	sl.Remove(1, cmp)
	testSL(t, sl, []int{2, 3, 4, 50, 6})

	sl.Remove(3, cmp)
	testSL(t, sl, []int{2, 4, 50, 6})

	sl.RemoveItem(i4)
	testSL(t, sl, []int{2, 50, 6})

	i0 := sl.Prepend(34)
	testSL(t, sl, []int{34, 2, 50, 6})

	sl.RemoveItem(i0)
	testSL(t, sl, []int{2, 50, 6})

	sl.Reset()
	sl.Prepend(1)
	sl.Prepend(3)
	testSL(t, sl, []int{3, 1})
}
