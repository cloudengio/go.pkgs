// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	"fmt"
	"math/rand"
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

func TestDups(t *testing.T) {
	h := heap.NewMin[uint32, int]()
	for i := 0; i < 20; i++ {
		h.Push(0, i)
		// The new duplicate will always be sifted to the top of the
		// heap.
		if got, want := h.Vals[0], i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	h.Verify(t, 0)

	var vals []int
	for i := 0; h.Len() > 0; i++ {
		k, v := h.Pop()
		h.Verify(t, 0)
		if got, want := k, uint32(0); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		vals = append(vals, v)
	}
	// Check for uniqueness of all vals.
	sort.IntSlice(vals).Sort()
	for i := 0; i < 20; i++ {
		if got, want := vals[i], i; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

type heapIfc[K heap.Ordered, V any] interface {
	Push(K, V)
	Pop() (K, V)
	Len() int
	Verify(*testing.T, int)
}

func testRand(t *testing.T, h heapIfc[int, int]) (input, output []int) {
	for i := 0; i < 1000; i++ {
		k := rand.Intn(1000000)
		input = append(input, k)
		h.Push(k, k)
	}
	h.Verify(t, 0)
	for h.Len() > 0 {
		_, v := h.Pop()
		h.Verify(t, 0)
		output = append(output, v)
	}
	return
}

func TestRand(t *testing.T) {
	in, out := testRand(t, heap.NewMin[int, int]())
	sort.Ints(in)
	if got, want := in, out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	in, out = testRand(t, heap.NewMax[int, int]())
	sort.Slice(in, func(i, j int) bool { return in[i] > in[j] })
	if got, want := in, out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	in, out = testRand(t, heap.NewMinBounded[int, int](10))
	sort.Ints(in)
	if got, want := in[:10], out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	in, out = testRand(t, heap.NewMaxBounded[int, int](10))
	sort.Slice(in, func(i, j int) bool { return in[i] > in[j] })
	if got, want := in[:10], out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func testData(t *testing.T, h heapIfc[int, int], input []int) []int {
	for _, k := range input {
		h.Push(k, k)
	}
	h.Verify(t, 0)
	output := make([]int, 0, len(input))
	for h.Len() > 0 {
		_, v := h.Pop()
		h.Verify(t, 0)
		output = append(output, v)
	}
	return output
}

func TestSimple(t *testing.T) {
	in := []int{1, 2, 3, 4, 5}
	out := testData(t, heap.NewMin[int, int](), in)
	if got, want := in, out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	in = []int{5, 4, 3, 2, 1}
	out = testData(t, heap.NewMin[int, int](), in)
	sort.Ints(in)
	if got, want := in, out; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}

	in = []int{1, 2, 3, 4, 5}
	out = testData(t, heap.NewMinBounded[int, int](3), in)
	if got, want := out, []int{1, 2, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	in = []int{5, 4, 3, 2, 1}
	out = testData(t, heap.NewMaxBounded[int, int](3), in)
	if got, want := out, []int{5, 4, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
