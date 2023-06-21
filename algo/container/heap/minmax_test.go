// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"cloudeng.io/algo/container/heap"
)

func ExampleNewMinMax() {
	h := heap.NewMinMax[int, string]()
	for _, i := range []int{12, 32, 25, 36, 13, 23, 26, 42, 49, 7, 15, 63, 92, 5} {
		h.Push(i, strconv.Itoa(i))
	}
	for h.Len() > 0 {
		k, _ := h.PopMin()
		fmt.Printf("%v ", k)
		k, _ = h.PopMax()
		fmt.Printf("%v ", k)
	}
	fmt.Println()
	// Output:
	// 5 92 7 63 12 49 13 42 15 36 23 32 25 26
}

func popMin(t *testing.T, h *heap.MinMax[int, int]) []int {
	return popMinMax(t, h, h.PopMin)
}

func popMax(t *testing.T, h *heap.MinMax[int, int]) []int {
	return popMinMax(t, h, h.PopMax)
}

func popMinMax(t *testing.T, h *heap.MinMax[int, int], pop func() (int, int)) []int {
	output := make([]int, 0)
	for h.Len() > 0 {
		k, v := pop()
		if got, want := k, v; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		h.Verify(t)
		output = append(output, v)
	}
	return output
}

func popAltMinMax(t *testing.T, h *heap.MinMax[int, int]) []int {
	output := make([]int, 0)
	for h.Len() > 0 {
		k, v := h.PopMin()
		if got, want := k, v; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		h.Verify(t)
		output = append(output, v)
		if h.Len() > 0 {
			k, v := h.PopMax()
			if got, want := k, v; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			h.Verify(t)
			output = append(output, v)
		}
	}
	return output
}

func pushMinMax(t *testing.T, h *heap.MinMax[int, int], input []int) {
	for _, k := range input {
		h.Push(k, k)
		h.Verify(t)
	}
}

func TestMinMaxDups(t *testing.T) {
	h := heap.NewMinMax[uint32, int]()
	for i := 0; i < 20; i++ {
		h.Push(0, i)
		// The new duplicate will always be left at the end of the heap.
		if got, want := h.Vals[len(h.Vals)-1], i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		h.Verify(t)
	}

	var vals []int
	for i := 0; h.Len() > 0; i++ {
		k, v := h.PopMin()
		h.Verify(t)
		if got, want := k, uint32(0); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		vals = append(vals, v)
	}
	checkDupValues(t, vals)
	checkUnique(t, vals)
}

func TestMinMaxHeap(t *testing.T) {
	for i := 0; i < 33; i++ {
		minmax := heap.NewMinMax[int, int]()
		pushMinMax(t, minmax, ascending(i))
		output := popMin(t, minmax)
		if got, want := output, ascending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		pushMinMax(t, minmax, ascending(i))
		output = popMax(t, minmax)
		if got, want := output, descending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	minmax := heap.NewMinMax[int, int]()
	rnd := uniformRand(0, 500)
	sorted := make([]int, len(rnd))
	copy(sorted, rnd)
	sort.Ints(sorted)
	pushMinMax(t, minmax, rnd)

	output := popMin(t, minmax)
	if got, want := output, sorted; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	pushMinMax(t, minmax, rnd)
	output = popMax(t, minmax)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	if got, want := output, sorted; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for i := 0; i < 33; i++ {
		rnd = uniformRand(0, i)
		pushMinMax(t, minmax, rnd)
		output = popAltMinMax(t, minmax)
		a, b := alternateData(rnd)
		for i, v := range output {
			w := a[i/2]
			if i%2 == 1 {
				w = b[i/2]
			}
			if got, want := v, w; got != want {
				t.Errorf("got %v, want %v", got, want)
				break
			}
		}
	}
}

func alternateData(data []int) ([]int, []int) {
	a, b := make([]int, len(data)), make([]int, len(data))
	copy(a, data)
	sort.Ints(a)
	copy(b, data)
	sort.Slice(b, func(i, j int) bool { return b[i] > b[j] })
	return a, b
}

func TestMinMaxBounded(t *testing.T) {
	for i := 0; i < 33; i++ {
		minmax := heap.NewMinMax[int, int]()
		n := i / 2
		if n == 0 {
			n = 1
		}
		for _, k := range ascending(i) {
			minmax.PushMaxN(k, k, n)
			minmax.Verify(t)
		}
		output := popMin(t, minmax)
		ln := n
		if i < n {
			ln = i
		}
		if got, want := len(output), ln; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		all := ascending(i)
		if n < len(all) {
			all = all[i-n:]
		}
		if got, want := output, all; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		for _, k := range descending(i) {
			minmax.PushMinN(k, k, n)
			minmax.Verify(t)
		}
		if got, want := len(output), ln; got != want {
			t.Errorf("got %v, want %v", got, want)
		}

		output = popMin(t, minmax)

		all = ascending(i)
		if n < len(all) {
			all = all[:n]
		}
		if got, want := output, all; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func testMinMaxRemove(t *testing.T, h *heap.MinMax[int, int], pos int, input []int) {
	size := len(input)

	pushMinMax(t, h, input)

	// Find the actual value to be removed and create a slice of the
	// remaining keys by removing that value from the original input.
	val := h.Keys[pos]
	sort.Ints(input)
	toBeRemoved := sort.SearchInts(input, val)
	remaining := append([]int{}, input[:toBeRemoved]...)
	remaining = append(remaining, input[toBeRemoved+1:]...)

	rk, _ := h.Remove(pos)
	if got, want := rk, val; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	h.VerifyQ(t)

	output := popMin(t, h)
	if got, want := output, remaining; !reflect.DeepEqual(got, want) {
		t.Errorf("remove item %v, len %v, got %v, want %v", pos, size, got, want)
	}
}

func TestMinMaxRemove(t *testing.T) {
	for i := 1; i < 33; i++ {
		minmax := heap.NewMinMax[int, int]()
		for r := 1; r <= i; r++ {
			testMinMaxRemove(t, minmax, r, ascending(i))
			testMinMaxRemove(t, minmax, r, descending(i))
			testMinMaxRemove(t, minmax, r, uniformRand(347, i))
		}
	}
}

func testMinMaxUpdate(t *testing.T, h *heap.MinMax[int, int], pos, delta int, input []int) {
	size := len(input)

	pushMinMax(t, h, input)

	// Find the actual value to be update and create a slice of the
	// with that updated value.
	val := h.Keys[pos]
	expected := make([]int, len(input))
	copy(expected, input)
	sort.Ints(expected)
	expected[sort.SearchInts(expected, val)] = val + delta
	sort.Ints(expected)

	h.Update(pos, val+delta, val+delta)
	if !h.VerifyQ(t) {
		heap.Pretty(h.Keys)
		t.Fatal("heap invariant violated")
	}

	output := popMin(t, h)
	if got, want := output, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("remove item %v, len %v, got %v, want %v", pos, size, got, want)
	}
}

func TestMinMaxUpdate(t *testing.T) {
	for i := 10; i < 33; i++ {
		minmax := heap.NewMinMax[int, int]()
		for r := 3; r <= i; r++ {
			testMinMaxUpdate(t, minmax, r, 2, ascending(i))
			testMinMaxUpdate(t, minmax, r, -2, ascending(i))
			testMinMaxUpdate(t, minmax, r, i/2, ascending(i))
			testMinMaxUpdate(t, minmax, r, -i/2, ascending(i))
			testMinMaxUpdate(t, minmax, r, 1000, ascending(i))
			testMinMaxUpdate(t, minmax, r, -1000, ascending(i))
		}
	}
}

func TestMinMaxOptions(t *testing.T) {
	h := heap.NewMinMax(heap.WithSliceCap[int, int](100))
	if got, want := cap(h.Keys), 100; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(h.Vals), 100; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(h.Keys), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(h.Vals), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func testMinMaxHeapify(t *testing.T, keys []int) {
	data := append([]int{0}, keys...)
	vals := append([]int{0}, keys...)
	h := heap.NewMinMax(heap.WithData(data, vals))
	h.Verify(t)
	if t.Failed() {
		heap.Pretty(h.Keys)
		return
	}
	output := popMin(t, h)
	expected := make([]int, len(keys))
	copy(expected, keys)
	sort.Ints(expected)
	if got, want := output, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMinMaxHeapify(t *testing.T) {
	testMinMaxHeapify(t, descending(36))
	testMinMaxHeapify(t, ascending(7))
	testMinMaxHeapify(t, uniformRand(32, 25))
}

func TestMinMaxCallback(t *testing.T) {
	locations := map[int]int{}
	data := ascending(13)

	h := heap.NewMinMax[int, int](heap.WithCallback[int, int](func(iv, jv int, i, j int) {
		locations[iv], locations[jv] = i, j
	}))

	pushMinMax(t, h, data)

	if got, want := len(locations), len(h.Keys)-1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for v, l := range locations {
		if got, want := h.Keys[l], v; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	h.Update(locations[10], 100, 100)

	output := popMin(t, h)
	// 10 is not present and 100 is the max.
	expected := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 100}
	if got, want := output, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	locations = map[int]int{}
	pushMinMax(t, h, data)
	for h.Len() > 0 {
		k, _ := h.PopMin()
		if got, want := locations[k], len(h.Keys); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		delete(locations, k)
	}

	if got, want := len(locations), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	locations = map[int]int{}
	pushMinMax(t, h, data)
	for h.Len() > 0 {
		k, _ := h.PopMax()
		if got, want := locations[k], len(h.Keys); got != want {
			t.Errorf("%v: got %v, want %v", k, got, want)
		}
		delete(locations, k)
	}

	if got, want := len(locations), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
