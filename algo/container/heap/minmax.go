// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

import (
	"fmt"
	"strings"
)

// MinMax represents a min-max heap as described in:
// https://liacs.leidenuniv.nl/~stefanovtp/courses/StudentenSeminarium/Papers/AL/SMMH.pdf.
// Note that this requires the use of a dummy root node in the key and value
// slices, ie. Keys[0] and Values[0] is always empty.
type MinMax[K Ordered, V any] struct {
	Keys []K
	Vals []V
}

// NewMinMax creates a new instance of MinMax.
func NewMinMax[K Ordered, V any](opts ...Option[K, V]) *MinMax[K, V] {
	var o options[K, V]
	o.sliceCap = 1
	for _, fn := range opts {
		fn(&o)
	}
	if o.keys != nil && o.vals != nil {
		h := &MinMax[K, V]{
			Keys: o.keys,
			Vals: o.vals,
		}
		h.heapify()
		return h
	}
	mm := &MinMax[K, V]{
		Keys: make([]K, 1, o.sliceCap),
		Vals: make([]V, 1, o.sliceCap),
	}
	return mm
}

func (h *MinMax[K, V]) heapify() {
	// Modified Floyd's algorithm to take account of the SMM's properties.
	// The 'leaves' must be ordered into their sibling order, and rather
	// than sifting down n/2, all of the nodes at the last level that does
	// not contain a leaf must be considered and sifted down.
	n := len(h.Keys)
	li := 1
	nn := n
	for {
		if nn >>= 1; nn == 1 {
			break
		}
		li++
	}
	for i := 1 << li; i < n; i += 2 {
		if h.Keys[i] < h.Keys[i-1] {
			h.swap(i, i-1)
		}
	}
	last := n - 1
	for i := 1<<li - 1; i > 0; i-- {
		if i%2 == 1 {
			if i+1 <= last && h.Keys[i] > h.Keys[i+1] {
				h.swap(i, i+1)
			}
			h.siftDownMin(i, last)
			continue
		}
		h.siftDownMax(i, last)
	}
}

// Len returns the number of items stored in the heap, excluding the dummy
// root node.
func (h *MinMax[K, V]) Len() int {
	return len(h.Keys) - 1
}

// PushMaxN pushes the key/value pair onto the heap if the key is greater than
// than the current maximum whilst ensuring that the heap is no larger than n.
func (h *MinMax[K, V]) PushMaxN(k K, v V, n int) {
	l := len(h.Keys)
	if l < n+1 {
		h.Push(k, v)
		return
	}
	if l > 1 && k < h.Keys[1] {
		// less than the min.
		return
	}
	h.PopMin()
	h.Push(k, v)
}

// PushMinN pushes the key/value pair onto the heap if the key is less than
// the current minimum whilst ensuring that the heap is no larger than n.
func (h *MinMax[K, V]) PushMinN(k K, v V, n int) {
	l := len(h.Keys)
	if l < n+1 {
		h.Push(k, v)
		return
	}
	maxIdx := 2
	if l == 2 {
		maxIdx = 1
	}
	if k > h.Keys[maxIdx] {
		// greater than the max.
		return
	}
	h.PopMax()
	h.Push(k, v)
}

// Push pushes the key/value pair onto the heap.
func (h *MinMax[K, V]) Push(k K, v V) {
	n := len(h.Keys)
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	if n == 1 {
		return
	}
	h.siftUp(h.adjustSiblings(n))
}

func (h *MinMax[K, V]) adjustSiblings(i int) int {
	if i >= 2 && i%2 == 0 {
		// if k is less than it's left sibling, if it has one, swap them.
		if h.Keys[i] < h.Keys[i-1] {
			h.swap(i, i-1)
			return i - 1
		}
	}
	return i
}

func (h *MinMax[K, V]) siftUp(i int) {
	for {
		// The lnode and rnodes is to calculate them
		// relative to current nodes grandparent.
		//            gp
		//       lnode rnode
		//      c1  c2   c3 c4
		gp := (i - 3) / 4
		if gp < 0 {
			break
		}
		lnode := (2 * gp) + 1
		rnode := lnode + 1
		if h.Keys[i] < h.Keys[lnode] {
			h.swap(lnode, i)
			i = lnode
			continue
		}
		if h.Keys[i] > h.Keys[rnode] {
			h.swap(rnode, i)
			i = rnode
			continue
		}
		break
	}
}

// Remove removes the i'th item from the heap, note that i includes the dummy
// root, i.e. i == 0 is the dummy root, 1 is the min, 2 is the max etc.
// Deleting the dummy root has no effect.
func (h *MinMax[K, V]) Remove(i int) (k K, v V) {
	n := len(h.Keys) - 1
	if i > n || i == 0 {
		return
	}
	k, v = h.Keys[i], h.Vals[i]
	if i < n {
		h.swap(i, n)
		if n > 2 {
			if i%2 == 1 { // min node
				h.updateMin(i, n-1)
			} else {
				h.updateMax(i, n-1)
			}
		}
	} // i == n, ie. last node means we can just remove it.
	h.Keys = h.Keys[:n]
	h.Vals = h.Vals[:n]
	return
}

// Update updates the i'th item in the heap, note that i includes the dummy
// root element. This is more efficient than removing and adding an item.
func (h *MinMax[K, V]) Update(i int, k K, v V) {
	n := len(h.Keys)
	if i == 0 || i >= n {
		return
	}
	h.Keys[i] = k
	h.Vals[i] = v
	if n == 2 {
		return
	}
	if i%2 == 1 {
		h.updateMin(i, n-1)
		return
	}
	h.updateMax(i, n-1)
}

func (h *MinMax[K, V]) updateMin(i int, last int) {
	if i+1 <= last && h.Keys[i] > h.Keys[i+1] {
		h.swap(i, i+1)
		h.siftUp(i + 1)
		h.siftDownMin(i, last)
		return
	}
	if !h.siftDownMin(i, last) {
		h.siftUp(i)
	}
}

func (h *MinMax[K, V]) updateMax(i int, last int) {
	if h.Keys[i] < h.Keys[i-1] {
		h.swap(i, i-1)
		h.siftUp(i - 1)
		h.siftDownMax(i, last)
		return
	}
	if !h.siftDownMax(i, last) {
		h.siftUp(i)
	}
}

func (h *MinMax[K, V]) swap(i, j int) {
	//	fmt.Printf("swap [%d] %v, [%d] %v\n", i, h.Keys[i], j, h.Keys[j])
	h.Keys[i], h.Keys[j] = h.Keys[j], h.Keys[i]
	h.Vals[i], h.Vals[j] = h.Vals[j], h.Vals[i]
}

func (h *MinMax[K, V]) set(i, j int) {
	h.Keys[i] = h.Keys[j]
	h.Vals[i] = h.Vals[j]
}

// PopMin removes and returns the smallest key/value pair from the heap.
func (h *MinMax[K, V]) PopMin() (K, V) {
	i := len(h.Keys) - 1
	k, v := h.Keys[1], h.Vals[1]
	h.set(1, i)
	h.siftDownMin(1, i)
	h.Keys, h.Vals = h.Keys[:i], h.Vals[:i]
	return k, v
}

func (h *MinMax[K, V]) siftDownMin(i, last int) bool {
	root := i
	for {
		minMinTree := (2 * i) + 1 // min node in min tree.
		if minMinTree > last {
			// no children, so we're done.
			break
		}

		// test to see if the current node is larger than the smallest
		// of the max nodes in either of the min or max trees, if so, swap
		// and continue with the new max value in the same position.
		if maxMinTree := minMinTree + 1; maxMinTree <= last {
			maxIdx := maxMinTree
			if maxMaxTree := minMinTree + 3; maxMaxTree <= last && h.Keys[maxMaxTree] <= h.Keys[maxMinTree] {
				maxIdx = maxMaxTree
			}
			if h.Keys[i] > h.Keys[maxIdx] {
				h.swap(i, maxIdx)
				continue
			}
		}

		// test to see if the current node is greater than the smallest
		// of the min nodes, if so, swap and continue with the new min value.
		minIdx := minMinTree
		if minMaxTree := minIdx + 2; minMaxTree <= last && h.Keys[minMaxTree] <= h.Keys[minMinTree] {
			minIdx = minMaxTree
		}
		if h.Keys[minIdx] < h.Keys[i] {
			h.swap(i, minIdx)
			i = minIdx
			continue
		}

		break
	}
	return i > root
}

// PopMax removes and returns the largest key/value pair from the heap.
func (h *MinMax[K, V]) PopMax() (K, V) {
	i := len(h.Keys) - 1
	if i == 1 {
		// Max is in the min node position for a heap
		// with a single member.
		k, v := h.Keys[1], h.Vals[1]
		h.Keys, h.Vals = h.Keys[:1], h.Vals[:1]
		return k, v
	}
	k, v := h.Keys[2], h.Vals[2]
	h.set(2, i)
	h.siftDownMax(2, i)
	h.Keys, h.Vals = h.Keys[:i], h.Vals[:i]
	return k, v
}

func (h *MinMax[K, V]) siftDownMax(i, last int) bool {
	root := i
	for {

		minMinTree := (2 * i) - 1
		if minMinTree > last || minMinTree < 0 {
			break
		}

		// test to see if the current node is smaller than the largest
		// of the min nodes in either of the min or max trees, if so, swap
		// and continue with the new min value in the same position.
		minIdx := minMinTree
		if minMaxTree := minIdx + 2; minMaxTree <= last && h.Keys[minMaxTree] >= h.Keys[minIdx] {
			minIdx = minMaxTree
		}
		if h.Keys[i] < h.Keys[minIdx] {
			h.swap(i, minIdx)
			continue
		}

		// test to see if the current node is smaller than the largest
		// of the max nodes, if so, swap and continue with the new value.
		maxIdx := (2 * i)
		if maxIdx > last {
			break
		}
		if maxMaxTree := maxIdx + 2; maxMaxTree <= last && h.Keys[maxMaxTree] > h.Keys[maxIdx] {
			maxIdx = maxMaxTree
		}
		if h.Keys[i] < h.Keys[maxIdx] {
			h.swap(i, maxIdx)
			i = maxIdx
			continue
		}

		break
	}
	return i > root
}

func pretty[K Ordered](k []K) {
	l := 0
	nsp := (((len(k) - 1) / 2) * 10)
	for i, v := range k {
		if i+1 == (1 << l) {
			l++
			fmt.Printf("\n% 4v:", (1<<(l-1))-1)
			nsp >>= 1
		}
		fmt.Printf("%v%v%v", strings.Repeat(" ", nsp), v, strings.Repeat(" ", nsp))
	}
	fmt.Println()
}
