// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

/*
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

type MaxHeap[K Ordered, V any] struct {
	Index  []K
	Values []V
}

func (h *MaxHeap[K, V]) Len() int { return len(h.Index) }

func (h *MaxHeap[K, V]) Less(i, j int) bool {
	return h.Index[i] > h.Index[j]
}

func (h *MaxHeap[K, V]) Swap(i, j int) {
	h.Index[i], h.Index[j] = h.Index[j], h.Index[i]
	h.Values[i], h.Values[j] = h.Values[j], h.Values[i]
}

func (h *MaxHeap[K, V]) Push(x any) {
	h.Index = append(h.Index, x.(K))
	h.Values = append(h.Values, x.(V))
}

func (h *MaxHeap[K, V]) Pop() any {
	old := h.Index
	n := len(old)
	x := old[n-1]
	h.Index = old[0 : n-1]
	h.Values = h.Values[0 : n-1]
	return x
}

type CompactUIntHeap struct {
	B8  []uint8
	B16 []uint16
	B32 []uint32
	B64 []uint64
}

// smallestBucketForValue returns the smallest bucket that can contain
// the specified value.
func (cih *CompactUIntHeap) smallestBucketForValue(v uint64) int {
	switch {
	case v < 0xff:
		return 8
	case v < 0xffff:
		return 16
	case v < 0xffffffff:
		return 32
	default:
		return 64
	}
}

type CompactMaxUIntHeap[V any] struct {
	B8  MaxHeap[uint8, V]
	B16 MaxHeap[uint16, V]
	B32 MaxHeap[uint32, V]
	B64 MaxHeap[uint64, V]
}

func (cih *CompactUIntHeap) Len() int {
	return len(cih.B8) + len(cih.B16) + len(cih.B32) + len(cih.B64)
}

func (cih *CompactMaxUIntHeap) bucket(i int) int {
	switch {
	case i < len(cih.B8):
		return 8
	case i < len(cih.B16):
		return 16
	case i < len(cih.B32):
		return 32
	default:
		return 64
	}
}

func (cih *CompactMaxUIntHeap) down(i, l int, v uint64) int {
	bucket := cih.smallestBucketForValue(v)
	for {
		if r := (2 * i) + 1; cih.bucket(r) == bucket {
			return r
		}
	}
}

func (cih *CompactMaxUIntHeap) Push(v uint64) {

}

/*
func (cih *CompactUIntHeap) Len() int {
	return len(cih.b0) + len(cih.b1) + len(cih.b2) + len(cih.b3)
}

func (cih *CompactUIntHeap) get(i int) uint64 {
	if i == cih.Len()-1 {
	}
	switch {
	case i < len(cih.b0):
		return uint64(cih.b0[i])
	case i < len(cih.b1):
		return uint64(cih.b1[i-len(cih.b0)])
	case i < len(cih.b2):
		return uint64(cih.b2[i-len(cih.b1)-len(cih.b0)])
	default:
		return uint64(cih.b3[i-len(cih.b2)-len(cih.b1)-len(cih.b0)])
	}
}

func (cih *CompactUIntHeap) set(i int, v uint64) {
	size := 0
	switch {
	case v < 0xff:
		size = 1
	case v < 0xffff:
		size = 2
	case v < 0xffffffff:
		size = 4
	default:
		size = 8
	}
	switch {
	case i < len(cih.b0):
		if size != 1 {
			panic(size)
		}
		cih.b0[i] = uint8(v & 0xff)
	case i < len(cih.b1):
		if size != 2 {
			panic(size)
		}
		cih.b1[i-len(cih.b0)] = uint16(v & 0xffff)
	case i < len(cih.b2):
		if size != 4 {
			panic(size)
		}
		cih.b2[i-len(cih.b1)-len(cih.b0)] = uint32(v & 0xffffffff)
	default:
		if size != 8 {
			panic(size)
		}
		cih.b3[i-len(cih.b2)-len(cih.b1)-len(cih.b0)] = v
	}
}

func (cih *CompactUIntHeap) Less(i, j int) bool {
	return cih.get(i) < cih.get(j)
}

func (cih *CompactUIntHeap) Swap(i, j int) {
	a, b := cih.get(i), cih.get(j)
	cih.set(i, b)
	cih.set(j, a)
}

func (cih *CompactUIntHeap) Push(v any) {
	//cih.set(cih.Len(), v.(uint64))
}

func (cih *CompactUIntHeap) Pop() any {
	return nil
}
*/
