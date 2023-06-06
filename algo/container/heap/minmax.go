// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap

import "fmt"

// MinMax represents a min-max heap as described in:
// https://liacs.leidenuniv.nl/~stefanovtp/courses/StudentenSeminarium/Papers/AL/SMMH.pdf.
// Note that this requires the use of a dummy root node in the key and value
// slices, ie. Keys[0] and Values[0] is always empty.
type MinMax[K Ordered, V any] struct {
	Keys []K
	Vals []V
}

// NewMinMax creates a new instance of MinMax.
func NewMinMax[K Ordered, V any]() *MinMax[K, V] {
	return &MinMax[K, V]{
		Keys: make([]K, 1),
		Vals: make([]V, 1),
	}
}

// Len returns the number of itesm stored in the heap, excluding the dummy
// root node.
func (h *MinMax[K, V]) Len() int {
	return len(h.Keys) - 1
}

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

func (h *MinMax[K, V]) Push(k K, v V) {
	n := len(h.Keys)
	h.Keys = append(h.Keys, k)
	h.Vals = append(h.Vals, v)
	if n == 1 {
		return
	}
	h.siftUp(k, n)
}

func (h *MinMax[K, V]) siftUp(k K, i int) {
	if i >= 2 && i%2 == 0 {
		// if k is less than it's left sibling, if it has one, swap them.
		if k < h.Keys[i-1] {
			h.swap(i, i-1)
			i--
		}
	}
	for {
		// The easiest way to find the lnode and rnodes is to calculate them
		// relative to current nodes grandparent.
		//            gp
		//       lnode rnode
		//      c1  c2   c3 c4
		gp := ((i + 1) / 4) - 1
		if gp < 0 {
			break
		}
		lnode := (2 * gp) + 1
		rnode := lnode + 1
		if k < h.Keys[lnode] {
			h.swap(lnode, i)
			i = lnode
			continue
		}
		if k > h.Keys[rnode] {
			h.swap(rnode, i)
			i = rnode
			continue
		}
		break
	}
}

func (h *MinMax[K, V]) Update(i int, k K, v V) {
	if i == 0 || i >= len(h.Keys) {
		return
	}
	if i%2 == 1 {
		// min node

	} else {
		// max node
	}
}

func rightMostChild(l, r, limit int) int {
	for {
		lc := (2 * l) + 1
		rc := (2 * r) + 2
		//fmt.Printf("REM: % 3v % 3v % 3v % 3v\n", l, r, lc, rc)
		if rc <= limit {
			l = lc
			r = rc
			continue
		}
		if lc <= limit {
			return r
		}
		//fmt.Printf("R: %v\n", r)
		return r
	}
}
func leftMostChild(l, r, limit int) int {
	for {
		lc := (2 * l) + 1
		//fmt.Printf("REM: % 3v % 3v % 3v % 3v\n", l, r, lc, rc)
		if lc <= limit {
			l = lc
			continue
		}
		return l
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
	if i == 1 && (n == 1 || n == 2) {
		k, v := h.Keys[1], h.Vals[1]
		h.Keys = h.Keys[:n]
		h.Vals = h.Vals[:n]
		return k, v
	}
	if i == 2 && n == 2 {
		k, v := h.Keys[2], h.Vals[2]
		h.swap(1, 2)
		h.Keys = h.Keys[:n]
		h.Vals = h.Vals[:n]
		return k, v
	}

	k, v = h.Keys[i], h.Vals[i]
	if i < n {
		h.swap(i, n)
		if (i*2)+1 > n {
			fmt.Printf("i is leaf: %v\n", i)
			h.siftUp(h.Keys[i], n-1)
		} else {
			//		h.swap(i, n)
			//p := (i - 1) / 2
			//		h.swap(i, p)
			if i%2 == 1 { // min node
				//rc := rightMostChild(i, i, n-1)
				/*			if n <= (2*i)+3 {
								fmt.Printf("min swap: %v <= %v\n", n, (2*i)+3)
								h.swap(i, n)
							} else {
								fmt.Printf("min did nothing....\n")
								return
							}
							//			h.swap(i, rc)
							fmt.Printf("i: %v ... %v\n", i, n)*/
				h.siftDownMin(i, n-1)
				//h.siftUp()
			} else { // max node
				/*			if n <= (2*i)+1 {
								fmt.Printf("max swap: %v <= %v\n", n, (2*i)+1)
								h.swap(i, n)
							} else {
								fmt.Printf("max did nothing....\n")
								return
							}*/
				//			lc := leftMostChild(i, i, n-1)
				//			h.set(i, lc)
				h.siftDownMax(i, n-1)
				//h.siftUp(h.Keys[lc], lc)
			}
		}
	}
	h.Keys = h.Keys[:n]
	h.Vals = h.Vals[:n]
	return
}

func (h *MinMax[K, V]) swap(i, j int) {
	fmt.Printf("swap: [%v] %v, [%v] %v\n", i, h.Keys[i], j, h.Keys[j])
	h.Keys[i], h.Keys[j] = h.Keys[j], h.Keys[i]
	h.Vals[i], h.Vals[j] = h.Vals[j], h.Vals[i]
}

func (h *MinMax[K, V]) set(i, j int) {
	h.Keys[i] = h.Keys[j]
	h.Vals[i] = h.Vals[j]
}

func (h *MinMax[K, V]) PopMin() (K, V) {
	i := len(h.Keys) - 1
	k, v := h.Keys[1], h.Vals[1]
	if i == 1 {
		h.Keys, h.Vals = h.Keys[:1], h.Vals[:1]
		return k, v
	}
	if i == 2 {
		h.Keys[1], h.Vals[1] = h.Keys[2], h.Vals[2]
		h.Keys, h.Vals = h.Keys[:2], h.Vals[:2]
		return k, v
	}
	h.set(1, i)
	h.siftDownMin(1, i)
	h.Keys, h.Vals = h.Keys[:i], h.Vals[:i]
	return k, v
}

func (h *MinMax[K, V]) siftDownMin(i, limit int) {
	for {
		gp := ((i + 1) / 2) - 1      // grandparent
		minMin := ((gp + 1) * 4) - 1 // min node in min tree
		if minMin > limit {
			break
		}

		// test to see if the current node is larger than the smallest
		// of the max nodes in either of the min or max trees, if so, swap.
		if maxMin := minMin + 1; maxMin <= limit {
			maxIdx := maxMin
			if maxMax := minMin + 3; maxMax <= limit && h.Keys[maxMin] > h.Keys[maxMax] {
				maxIdx = maxMax
			}
			if h.Keys[i] > h.Keys[maxIdx] {
				// maxIdx is smaller than the current node, so swap.
				h.swap(i, maxIdx)
				continue
			}
		}

		// minIdx is the smallest of the min nodes.
		minMax := minMin + 2 // min node in the max tree
		minIdx := minMin
		if minMax <= limit && h.Keys[minMin] > h.Keys[minMax] {
			// compare against the smallest of the min nodes.
			minIdx = minMax
		}

		if h.Keys[minIdx] < h.Keys[i] {
			// min child is smaller than the current node, so swap.
			h.swap(i, minIdx)
			i = minIdx
			continue
		}
		break
	}
}

func (h *MinMax[K, V]) PopMax() (K, V) {
	i := len(h.Keys) - 1
	if i == 1 {
		k, v := h.Keys[1], h.Vals[1]
		h.Keys, h.Vals = h.Keys[:1], h.Vals[:1]
		return k, v
	}
	if i == 2 {
		k, v := h.Keys[2], h.Vals[2]
		h.Keys, h.Vals = h.Keys[:2], h.Vals[:2]
		return k, v
	}
	k, v := h.Keys[2], h.Vals[2]
	h.set(2, i)
	h.siftDownMax(2, i)
	h.Keys, h.Vals = h.Keys[:i], h.Vals[:i]
	return k, v
}

func (h *MinMax[K, V]) siftDownMax(i, limit int) {
	for {
		gp := (i / 2) - 1        // grandparent
		maxMin := ((gp + 1) * 4) // max node in min tree
		if maxMin > limit {
			break
		}

		// test to see if the current node is smaller than the largest
		// of the min nodes in either of the min or max trees, if so, swap.
		minMin := maxMin - 1
		minIdx := minMin
		if minMax := minMin + 2; minMax <= limit && h.Keys[minMin] < h.Keys[minMax] {
			minIdx = minMax
		}
		if h.Keys[i] < h.Keys[minIdx] {
			// maxIdx is smaller than the current node, so swap.
			h.swap(i, minIdx)
			continue
		}

		// maxIdx is the largest of the max nodes.
		maxMax := maxMin + 2 // max node in the max tree
		maxIdx := maxMin
		if maxMax <= limit && h.Keys[maxMax] > h.Keys[maxMin] {
			// compare against the smallest of the min nodes.
			maxIdx = maxMax
		}

		if h.Keys[maxIdx] > h.Keys[i] {
			// min child is smaller than the current node, so swap.
			h.swap(i, maxIdx)
			i = maxIdx
			continue
		}
		break
	}
}
