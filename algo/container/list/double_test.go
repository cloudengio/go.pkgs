// Copyright 2024 loudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package list_test

import (
	"slices"
	"testing"

	"cloudeng.io/algo/container/list"
)

func forward[T any](dl *list.Double[T]) []T {
	var res []T
	for g := range dl.Forward() {
		res = append(res, g)
	}
	return res
}

func reverse[T any](dl *list.Double[T]) []T {
	var res []T
	for g := range dl.Reverse() {
		res = append(res, g)
	}
	return res
}

func testDL[T comparable](t *testing.T, dl *list.Double[T], fwd []T) {
	if got, want := forward(dl), fwd; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := dl.Len(), len(fwd); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(fwd) > 0 {
		if got, want := dl.Head(), fwd[0]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := dl.Tail(), fwd[len(fwd)-1]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	rev := slices.Clone(fwd)
	slices.Reverse(rev)
	if got, want := reverse(dl), rev; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDL(t *testing.T) {
	dl := list.NewDouble[int]()
	testDL(t, dl, []int{})
	if got, want := dl.Head(), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := dl.Tail(), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	dl.Append(1)

	testDL(t, dl, []int{1})
	dl.Append(2)
	testDL(t, dl, []int{1, 2})
	dl.Append(3)
	testDL(t, dl, []int{1, 2, 3})
	i4 := dl.Append(4)
	dl.Append(50)
	dl.Append(6)
	testDL(t, dl, []int{1, 2, 3, 4, 50, 6})

	cmp := func(a, b int) bool {
		return a == b
	}
	dl.Remove(1, cmp)
	testDL(t, dl, []int{2, 3, 4, 50, 6})
	dl.RemoveReverse(6, cmp)
	testDL(t, dl, []int{2, 3, 4, 50})
	dl.Remove(3, cmp)
	testDL(t, dl, []int{2, 4, 50})
	dl.RemoveItem(i4)
	testDL(t, dl, []int{2, 50})
	i0 := dl.Prepend(34)
	testDL(t, dl, []int{34, 2, 50})
	dl.RemoveItem(i0)
	testDL(t, dl, []int{2, 50})
	dl.Reset()
	dl.Prepend(1)
	dl.Prepend(3)
	testDL(t, dl, []int{3, 1})

}
