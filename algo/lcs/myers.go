// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package lcs

// Myers represents an implementation of Myer's longest common subsequence
// and shortest edit script algorithm as as documented in:
// An O(ND) Difference Algorithm and Its Variations, 1986.
type Myers[T comparable] struct {
	a, b []T
}

// NewMyers returns a new instance of Myers. The implementation supports slices
// of bytes/uint8, rune/int32 and int64s.
func NewMyers[T comparable](a, b []T) *Myers[T] {
	return &Myers[T]{a: a, b: b}
}

// Details on the implementation and details of the algorithms can be found
// here:
// http://xmailserver.org/diff2.pdf
// http://simplygenius.net/Article/DiffTutorial1
// https://blog.robertelder.org/diff-algorithm/

func forwardSearch[T comparable](a, b []T, d int32, forward, reverse []int32, offset int32) (nd, mx, my, x, y int32, ok bool) {
	na, nb := int32(len(a)), int32(len(b))
	delta := na - nb
	odd := delta%2 != 0
	for k := -d; k <= d; k += 2 {
		// Edge cases are:
		// k == -d    - move down
		// k == d * 2 - move right
		// Normal case:
		// move down or right depending on how far the move would be.
		if k == -d || k != d && forward[offset+k-1] < forward[offset+k+1] {
			x = forward[offset+k+1]
		} else {
			x = forward[offset+k-1] + 1
		}
		y = x - k
		mx, my = x, y
		for x < na && y < nb && a[x] == b[y] {
			x++
			y++
		}
		forward[offset+k] = x
		// Can this snake potentially overlap with one of the reverse ones?
		// Going forward only odd paths can be the longest ones.
		if odd && (-(k - delta)) >= -(d-1) && (-(k - delta)) <= (d-1) {
			// Doe this snake overlap with one of the reverse ones? If so,
			// the last snake is the longest one.
			if forward[offset+k]+reverse[offset-(k-delta)] >= na {
				return 2*d - 1, mx, my, x, y, true
			}
		}
	}
	return 0, 0, 0, 0, 0, false
}

func reverseSearch[T comparable](a, b []T, d int32, forward, reverse []int32, offset int32) (nd, mx, my, x, y int32, ok bool) {
	na, nb := int32(len(a)), int32(len(b))
	delta := na - nb
	even := delta%2 == 0
	for k := -d; k <= d; k += 2 {
		// Edge cases as per forward search, but looking at the reverse
		// stored values.
		if k == -d || k != d && reverse[offset+k-1] < reverse[offset+k+1] {
			x = reverse[offset+k+1]
		} else {
			x = reverse[offset+k-1] + 1
		}
		y = x - k
		mx, my = x, y
		for x < na && y < nb && a[na-x-1] == b[nb-y-1] {
			x++
			y++
		}
		reverse[offset+k] = x
		// Can this snake potentially overlap with one of the forward ones?
		// Going backward only even paths can be the longest ones.
		if even && (-(k - delta)) >= -d && (-(k - delta)) <= d {
			// Doe this snake overlap with one of the forward ones? If so,
			// the last snake is the longest one.
			if reverse[offset+k]+forward[offset-(k-delta)] >= na {
				return 2 * d, na - x, nb - y, na - mx, nb - my, true
			}
		}
	}
	return 0, 0, 0, 0, 0, false
}

func middleSnake[T comparable](a, b []T) (d, x1, y1, x2, y2 int32) {
	maxe := int32(len(a) + len(b)) // max # edits (delete all a, insert all of b)

	// forward and reverse are accessed using k which is in the
	// range -d .. +d, hence offset must be added to k.
	forward := make([]int32, maxe+2)
	reverse := make([]int32, maxe+2)
	offset := int32(len(forward) / 2)

	// Only need to search for D halfway through the table.
	halfway := maxe / 2
	if maxe%2 != 0 {
		halfway++
	}
	for d := int32(0); d <= halfway; d++ {
		if nd, mx, my, x, y, ok := forwardSearch(a, b, d, forward, reverse, offset); ok {
			return nd, mx, my, x, y
		}
		if nd, mx, my, x, y, ok := reverseSearch(a, b, d, forward, reverse, offset); ok {
			return nd, mx, my, x, y
		}
	}
	panic("unreachable")
}

func myersLCS[T comparable](a, b []T) []T {
	if len(a) == 0 || len(b) == 0 {
		return []T{}
	}
	d, x, y, u, v := middleSnake(a, b)
	if d > 1 {
		nd := myersLCS(a[:x], b[:y])
		nd = append(nd, a[x:u]...)
		nd = append(nd, myersLCS(a[u:], b[v:])...)
		return nd
	}
	if len(b) > len(a) {
		return append([]T{}, a...)
	}
	return append([]T{}, b...)
}

// LCS returns the longest common subsquence.
func (m *Myers[T]) LCS() []T {
	return myersLCS(m.a, m.b)
}

func (m *Myers[T]) ses(idx int, a, b []T, na, nb, cx, cy int32) []Edit[T] {
	var ses []Edit[T]
	if na > 0 && nb > 0 {
		d, x, y, u, v := middleSnake(a, b)
		if d > 1 || (x != u && y != v) {
			ses = append(ses,
				m.ses(idx+1, a[0:x], b[0:y], x, y, cx, cy)...)
			if x != u && y != v {
				// middle snake is part of the lcs.
				ses = append(ses, m.edits(a[x:u], Identical, int(cx+x), int(cy+y))...)
			}
			return append(ses,
				m.ses(idx+1, a[u:na], b[v:nb], na-u, nb-v, cx+u, cy+v)...)
		}
		if nb > na {
			// a is part of the LCS.
			ses = append(ses, m.edits(a[0:na], Identical, int(cx), int(cy))...)
			return append(ses,
				m.ses(idx+1, nil, b[na:nb], 0, nb-na, cx+na, cy+na)...)
		}
		if na > nb {
			// b is part of the LCS.
			ses = append(ses, m.edits(b[0:nb], Identical, int(cx), int(cy))...)
			return append(ses,
				m.ses(idx+1, a[nb:na], nil, na-nb, 0, cx+nb, cy+nb)...)
		}
		return ses
	}
	if na > 0 {
		return m.edits(a, Delete, int(cx), int(cy))
	}
	return m.edits(b, Insert, int(cx), int(cy))
}

func (m *Myers[T]) edits(vals []T, op EditOp, cx, cy int) []Edit[T] {
	createEdit := func(cx, cy, i int, op EditOp, val T) Edit[T] {
		atx := cx + i
		if op == Insert {
			atx = cx
		}
		return Edit[T]{op, atx, cy + i, val}
	}
	var edits []Edit[T]
	for i, v := range vals {
		edits = append(edits, createEdit(cx, cy, i, op, v))
	}
	return edits
}

// SES returns the shortest edit script.
func (m *Myers[T]) SES() *EditScript[T] {
	var es EditScript[T] = m.ses(0, m.a, m.b, int32(len(m.a)), int32(len(m.b)), 0, 0)
	return &es
}
