// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap_test

import (
	stdheap "container/heap"
	"math/rand"
	"testing"

	"cloudeng.io/algo/container/heap"
)

type withValue[K heap.Ordered, V any] struct {
	k K
	v V
}

type withValueSlice[K heap.Ordered, V any] []withValue[K, V]

func (h *withValueSlice[K, V]) Less(i, j int) bool {
	return (*h)[i].k < (*h)[j].k
}

func (h *withValueSlice[K, V]) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *withValueSlice[K, V]) Len() int {
	return len(*h)
}

func (h *withValueSlice[K, V]) Pop() (v any) {
	old := *h
	n := len(old)
	v = (*h)[n-1]
	*h = old[:n-1]
	return
}

func (h *withValueSlice[K, V]) Push(v any) {
	*h = append(*h, v.(withValue[K, V]))
	// fmt.Printf("std push: %v: %v\n", h.Len(), *h)
}

func uniformRand(seed int64, n int) []int {
	rnd := rand.New(rand.NewSource(seed)) // #nosec: G404
	r := make([]int, n)
	for i := range r {
		r[i] = rnd.Intn(10000)
	}
	return r
}

func zipfRand(seed int64, n int) []uint64 {
	rnd := rand.New(rand.NewSource(seed))                // #nosec: G404
	gen := rand.NewZipf(rnd, 3.0, 1.1, 8*1024*1024*1024) // 8Gib
	r := make([]uint64, n)
	for i := range r {
		r[i] = gen.Uint64()
	}
	return r
}

func benchmarkStdeap[K heap.Ordered, V any](b *testing.B, h *withValueSlice[K, V], keys []K, v V) {
	for i := 0; i < b.N; i++ {
		for j := range keys {
			stdheap.Push(h, withValue[K, V]{k: keys[j], v: v})
		}
		for h.Len() > 0 {
			_ = stdheap.Pop(h).(withValue[K, V])

		}
	}
}

const BenchmarkInputSize = 10

func BenchmarkStdHeapDup_10000(b *testing.B) {
	b.ReportAllocs()
	keys := make([]int, BenchmarkInputSize)
	h := make(withValueSlice[int, int], 0, len(keys))
	b.ResetTimer()
	benchmarkStdeap(b, &h, keys, 0)
}

func BenchmarkStdHeapRand_10000(b *testing.B) {
	b.ReportAllocs()
	keys := uniformRand(0, BenchmarkInputSize)
	h := make(withValueSlice[int, int], 0, len(keys))
	b.ResetTimer()
	benchmarkStdeap(b, &h, keys, 0)
}

func BenchmarkStdHeapZipf_10000(b *testing.B) {
	b.ReportAllocs()
	keys := zipfRand(0, BenchmarkInputSize)
	h := make(withValueSlice[uint64, int], 0, len(keys))
	b.ResetTimer()
	benchmarkStdeap(b, &h, keys, 0)
}

func benchmarkGenHeap[K heap.Ordered, V any](b *testing.B, h *heap.T[K, V], keys []K, v V) {
	for i := 0; i < b.N; i++ {
		for j := range keys {
			h.Push(keys[j], v)
		}
		for h.Len() > 0 {
			h.Pop()
		}
	}
}

func BenchmarkGenHeapDup_10000(b *testing.B) {
	b.ReportAllocs()
	keys := make([]int, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMin[int, int]()
	benchmarkGenHeap(b, h, keys, 0)
}

func BenchmarkGenHeapRand_10000(b *testing.B) {
	b.ReportAllocs()
	keys := uniformRand(0, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMin[int, int]()
	benchmarkGenHeap(b, h, keys, 0)
}

func BenchmarkGenHeapZipf_10000(b *testing.B) {
	b.ReportAllocs()
	keys := zipfRand(0, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMin[uint64, int]()
	benchmarkGenHeap(b, h, keys, 0)
}

func benchmarkMinMax[K heap.Ordered, V any](b *testing.B, h *heap.MinMax[K, V], keys []K, v V) {
	for i := 0; i < b.N; i++ {
		for j := range keys {
			h.Push(keys[j], v)
		}
		for h.Len() > 0 {
			h.PopMin()
		}
	}
}

func BenchmarkMinMaxHeapDup_10000(b *testing.B) {
	b.ReportAllocs()
	keys := make([]int, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMinMax[int, int]()
	benchmarkMinMax(b, h, keys, 0)
}

func BenchmarkMinMaxHeapRand_10000(b *testing.B) {
	b.ReportAllocs()
	keys := uniformRand(0, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMinMax[int, int]()
	benchmarkMinMax(b, h, keys, 0)
}

func BenchmarkMinMaxHeapZipf_10000(b *testing.B) {
	b.ReportAllocs()
	keys := zipfRand(0, BenchmarkInputSize)
	b.ResetTimer()
	h := heap.NewMinMax[uint64, int]()
	benchmarkMinMax(b, h, keys, 0)
}
