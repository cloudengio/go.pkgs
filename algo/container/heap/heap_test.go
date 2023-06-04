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

func ExampleNewMin() {
	h := heap.NewMin[int, string]()
	for _, i := range []int{100, 19, 36, 17, 3, 25, 1, 2, 7} {
		h.Push(i, strconv.Itoa(i))
	}
	for h.Len() > 0 {
		k, _ := h.Pop()
		fmt.Printf("%v ", k)
	}
	fmt.Println()
	// Output:
	// 1 2 3 7 17 19 25 36 100
}

func ExampleNewMax() {
	h := heap.NewMax[int, string]()
	for _, i := range []int{100, 19, 36, 17, 3, 25, 1, 2, 7} {
		h.Push(i, strconv.Itoa(i))
	}
	for h.Len() > 0 {
		k, _ := h.Pop()
		fmt.Printf("%v ", k)
	}
	fmt.Println()
	// Output:
	// 100 36 25 19 17 7 3 2 1
}

func checkDupValues(t *testing.T, vals []int) {
	if vals[0] != 0 {
		t.Errorf("first popped value should be 0")
	}
	// Subsequent values should be in reverse order.
	for i := 1; i < 20; i++ {
		if got, want := vals[i], 20-i; i > 0 && got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func checkUnique(t *testing.T, vals []int) {
	// Check for uniqueness of all vals.
	sort.IntSlice(vals).Sort()
	for i := 0; i < 20; i++ {
		if got, want := vals[i], i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestDups(t *testing.T) {
	h := heap.NewMin[uint32, int]()
	for i := 0; i < 20; i++ {
		h.Push(0, i)
		// The new duplicate will always be left at the end of the heap.
		if got, want := h.Vals[len(h.Vals)-1], i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	h.Verify(t)

	var vals []int
	for i := 0; h.Len() > 0; i++ {
		k, v := h.Pop()
		h.Verify(t)
		if got, want := k, uint32(0); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		vals = append(vals, v)
	}
	checkDupValues(t, vals)
	checkUnique(t, vals)
}

type heapIfc[K heap.Ordered, V any] interface {
	Push(K, V)
	Pop() (K, V)
	Len() int
	Verify(*testing.T)
}

func testRand(t *testing.T, h heapIfc[int, int]) (input, output []int) {
	input = uniformRand(0, 1000)
	push(t, h, input)
	return input, pop(t, h)
}

func TestRand(t *testing.T) {
	var in, out []int
	in, out = testRand(t, heap.NewMin[int, int]())
	sort.Ints(in)
	if got, want := out, in; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	in, out = testRand(t, heap.NewMax[int, int]())
	sort.Slice(in, func(i, j int) bool { return in[i] > in[j] })
	if got, want := out, in; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	input := uniformRand(0, 1000)
	keys, vals := make([]int, len(input)), make([]int, len(input))
	for i, k := range input {
		keys[i], vals[i] = k, k
	}
	h := heap.NewMin(heap.WithData(keys, vals))
	h.Verify(t)
}

func testData(t *testing.T, h heapIfc[int, int], input []int) []int {
	push(t, h, input)
	return pop(t, h)
}

func ascending(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func descending(n int) []int {
	out := make([]int, n)
	v := n - 1
	for i := range out {
		out[i] = v
		v--
	}
	return out
}

func push(t *testing.T, h heapIfc[int, int], input []int) {
	for _, k := range input {
		h.Push(k, k)
	}
	h.Verify(t)
}

func pop(t *testing.T, h heapIfc[int, int]) []int {
	output := make([]int, 0)
	for h.Len() > 0 {
		k, v := h.Pop()
		if got, want := k, v; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		h.Verify(t)
		output = append(output, v)
	}
	return output
}

func TestHeap(t *testing.T) {
	for i := 0; i < 33; i++ {
		min := heap.NewMin[int, int]()
		output := testData(t, min, ascending(i))
		if got, want := output, ascending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		output = testData(t, min, descending(i))
		if got, want := output, ascending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		min = heap.NewMin(heap.WithData(ascending(i), ascending(i)))
		min.Verify(t)
		if got, want := pop(t, min), ascending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		max := heap.NewMax[int, int]()
		output = testData(t, max, ascending(i))
		if got, want := output, descending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		output = testData(t, max, descending(i))
		if got, want := output, descending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		max = heap.NewMax(heap.WithData(ascending(i), ascending(i)))
		min.Verify(t)
		if got, want := pop(t, max), descending(i); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestOptions(t *testing.T) {
	h := heap.NewMax(heap.WithSliceCap[int, int](100))
	if got, want := cap(h.Keys), 100; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cap(h.Vals), 100; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(h.Keys), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(h.Vals), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
