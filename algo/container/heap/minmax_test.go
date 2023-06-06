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

func TestMinMaxRemove(t *testing.T) {
	for i := 1; i < 33; i++ {
		minmax := heap.NewMinMax[int, int]()
		for r := 1; r < i; r++ {
			input := ascending(i)
			pushMinMax(t, minmax, input)

			fmt.Printf("Before: %v, #keys %v\n", minmax.Keys, len(minmax.Keys))
			heap.Pretty(minmax.Keys)
			rk, _ := minmax.Remove(r)
			if !minmax.VerifyQ(t) {
				fmt.Printf("RM: remove %v, len %v, #keys %v, %v, rk: %v \n", r, i, len(minmax.Keys), minmax.Keys, rk)
			}
			heap.Pretty(minmax.Keys)
			//				break

			output := popMin(t, minmax)
			idx := sort.SearchInts(input, rk)
			expected := append(input[:idx], input[idx+1:]...)
			//fmt.Printf("i: %v, r %v, input %v\nG: %v\nW: %v\n", i, r, input, output, expected)

			if got, want := output, expected; !reflect.DeepEqual(got, want) {
				t.Errorf("remove item %v, got %v, want %v", i, got, want)
				//				break
			}
		}
	}
	//t.Fail()
}
