// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"cloudeng.io/algo/container/heap"
)

type Example struct {
	idx int
	val string
}

func (e Example) Less(b Example) bool {
	return e.idx < b.idx
}

func ExampleHeap() {
	h := heap.Heap[Example]{
		{idx: 3, val: "three"},
		{idx: 1, val: "one"},
		{idx: 2, val: "two"},
		{idx: 0, val: "zero"},
	}
	h.Init()
	for h.Len() > 0 {
		v := h.Pop()
		fmt.Printf("%v %v ", v.idx, v.val)
	}
	fmt.Println()
	// Output:
	// 0 zero 1 one 2 two 3 three
}

// IntType implements the Less interface for integers
type IntType int

func (i IntType) Less(b IntType) bool {
	return i < b
}

// StringType implements the Less interface for strings
type StringType string

func (s StringType) Less(b StringType) bool {
	return s < b
}

// Custom type with custom comparison logic
type Person struct {
	Name string
	Age  int
}

func (p Person) Less(b Person) bool {
	// Sort by age
	return p.Age < b.Age
}

func TestGenericHeap_Int(t *testing.T) {
	// Initialize a heap with some values
	values := []IntType{5, 3, 7, 2, 8, 1, 6, 4}
	h := heap.Heap[IntType](values)

	// Build the heap
	h.Init()

	// Verify heap properties after initialization
	if h.Len() != len(values) {
		t.Errorf("heap.Len() = %d, want %d", h.Len(), len(values))
	}

	// Push a new value
	h.Push(IntType(0))

	// Verify the length increased
	if h.Len() != len(values)+1 {
		t.Errorf("heap.Len() after push = %d, want %d", h.Len(), len(values)+1)
	}

	// Pop all values and verify they come out in sorted order
	var result []int
	for h.Len() > 0 {
		result = append(result, int(h.Pop()))
	}

	// Verify the result is sorted
	if !sort.IntsAreSorted(result) {
		t.Errorf("heap did not produce sorted output: %v", result)
	}

	// Verify the heap is now empty
	if h.Len() != 0 {
		t.Errorf("heap.Len() after popping all elements = %d, want 0", h.Len())
	}
}

func TestGenericHeap_String(t *testing.T) {
	// Initialize a heap with some string values
	values := []StringType{"banana", "apple", "cherry", "date", "elderberry"}
	h := heap.Heap[StringType](values)

	// Build the heap
	h.Init()

	// Pop all values and verify they come out in sorted order
	var result []string
	for h.Len() > 0 {
		result = append(result, string(h.Pop()))
	}

	// Verify the result is sorted
	expected := []string{"apple", "banana", "cherry", "date", "elderberry"}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("heap result[%d] = %s, want %s", i, v, expected[i])
		}
	}
}

func TestGenericHeap_CustomType(t *testing.T) {
	// Initialize a heap with custom type
	people := []Person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
		{"Diana", 20},
		{"Edward", 40},
	}
	h := heap.Heap[Person](people)

	// Build the heap
	h.Init()

	// Pop all values and verify they come out sorted by age
	var ages []int
	for h.Len() > 0 {
		person := h.Pop()
		ages = append(ages, person.Age)
	}

	// Verify ages are in ascending order
	expectedAges := []int{20, 25, 30, 35, 40}
	for i, age := range ages {
		if age != expectedAges[i] {
			t.Errorf("person age[%d] = %d, want %d", i, age, expectedAges[i])
		}
	}
}

func TestGenericHeap_EmptyHeap(t *testing.T) {
	// Test operations on an empty heap
	var h heap.Heap[IntType]

	// Initialize empty heap
	h.Init()

	// Verify length is 0
	if h.Len() != 0 {
		t.Errorf("empty heap.Len() = %d, want 0", h.Len())
	}

	// Push an element
	h.Push(IntType(42))

	// Verify length is 1
	if h.Len() != 1 {
		t.Errorf("heap.Len() after push = %d, want 1", h.Len())
	}

	// Pop the element
	val := h.Pop()
	if val != 42 {
		t.Errorf("heap.Pop() = %v, want 42", val)
	}

	// Verify heap is empty again
	if h.Len() != 0 {
		t.Errorf("heap.Len() after pop = %d, want 0", h.Len())
	}
}

func TestGenericHeap_PushPop(t *testing.T) {
	// Test pushing and popping in various orders
	h := heap.Heap[IntType]{}
	h.Init()

	// Push values in non-sorted order
	h.Push(IntType(5))
	h.Push(IntType(2))
	h.Push(IntType(7))
	h.Push(IntType(3))

	// Pop one value
	val1 := h.Pop()
	if val1 != 2 {
		t.Errorf("first heap.Pop() = %v, want 2", val1)
	}

	// Push more values
	h.Push(IntType(1))
	h.Push(IntType(8))

	// Pop remaining values and verify order
	expected := []int{1, 3, 5, 7, 8}
	for i, want := range expected {
		if h.Len() == 0 {
			t.Fatalf("heap empty after %d pops, expected %d more elements", i, len(expected)-i)
		}
		got := int(h.Pop())
		if got != want {
			t.Errorf("heap.Pop()[%d] = %v, want %v", i, got, want)
		}
	}
}

func TestGenericHeap_Remove(t *testing.T) {
	// Test 1: Remove the root element
	h := heap.Heap[IntType]{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	h.Init()
	removed := h.Remove(0).(IntType)
	if removed != 0 {
		t.Errorf("Test 1: Remove(0) got %v, want 0", removed)
	}
	if h.Len() != 9 {
		t.Errorf("Test 1: Len after Remove(0) is %v, want 9", h.Len())
	}
	if m := h.Pop(); m != 1 {
		t.Errorf("Test 1: Pop after Remove(0) got %v, want 1", m)
	}

	// Test 2: Remove an element from the middle
	h = heap.Heap[IntType]{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	h.Init()
	removed = h.Remove(4).(IntType) // Remove value 4
	if removed != 4 {
		t.Errorf("Test 2: Remove(4) got %v, want 4", removed)
	}
	var result []int
	for h.Len() > 0 {
		result = append(result, int(h.Pop()))
	}
	expected := []int{0, 1, 2, 3, 5, 6, 7, 8, 9}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Test 2: Pop after Remove(4) got %v, want %v", result, expected)
	}

	// Test 3: Remove the last element
	h = heap.Heap[IntType]{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	h.Init()
	lastIdx := h.Len() - 1
	lastVal := h[lastIdx]
	removed = h.Remove(lastIdx).(IntType)
	if removed != lastVal {
		t.Errorf("Test 3: Remove(last) got %v, want %v", removed, lastVal)
	}
	result = nil
	for h.Len() > 0 {
		result = append(result, int(h.Pop()))
	}
	expected = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Test 3: Pop after Remove(last) got %v, want %v", result, expected)
	}
}

func TestGenericHeap_Fix(t *testing.T) {
	// Test 1: Increase a value, requiring a down-heap operation
	h := heap.Heap[IntType]{2, 4, 6, 8, 10}
	h.Init()
	h[0] = 12 // Violate heap property by making root largest
	heap.Fix(h, 0)
	if h[0] != 4 {
		t.Errorf("Test 1: Fix failed, root is %v, want 4", h[0])
	}
	var result []int
	for h.Len() > 0 {
		result = append(result, int(h.Pop()))
	}
	if !sort.IntsAreSorted(result) {
		t.Errorf("Test 1: heap not sorted after Fix and pop: %v", result)
	}

	// Test 2: Decrease a value, requiring an up-heap operation
	h = heap.Heap[IntType]{2, 4, 6, 8, 10}
	h.Init()
	h[h.Len()-1] = 1 // Violate heap property by making a leaf smallest
	heap.Fix(h, h.Len()-1)
	if h[0] != 1 {
		t.Errorf("Test 2: Fix failed, root is %v, want 1", h[0])
	}
}

func TestGenericHeap_Duplicates(t *testing.T) {
	values := []IntType{5, 3, 7, 2, 8, 1, 6, 4, 5, 3}
	h := heap.Heap[IntType](values)
	h.Init()
	h.Push(IntType(3))

	var result []int
	for h.Len() > 0 {
		result = append(result, int(h.Pop()))
	}

	expected := []int{1, 2, 3, 3, 3, 4, 5, 5, 6, 7, 8}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("heap with duplicates failed. got %v, want %v", result, expected)
	}
}

func BenchmarkPushPop(b *testing.B) {
	const size = 10000
	data := make([]IntType, size)
	for i := 0; i < size; i++ {
		data[i] = IntType(size - i) // Push in reverse sorted order
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var h heap.Heap[IntType]
		for _, v := range data {
			h.Push(v)
		}
		for h.Len() > 0 {
			h.Pop()
		}
	}
}
