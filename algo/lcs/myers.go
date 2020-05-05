package lcs

import "fmt"

// Myers represents an implementation of Myer's longest common subsequence
// and shortest edit script algorithm as as documented in:
// An O(ND) Difference Algorithm and Its Variations, 1986.
type Myers struct {
	a, b       interface{}
	na, nb     int
	cmp        comparator
	slicer     func(v interface{}, from, to int32) interface{}
	deletions  func(v interface{}, cx int) []Edit
	insertions func(v interface{}, cx, cy int) []Edit
}

// NewMyers returns a new instance of Myers. The implementation supports slices
// of bytes/uint8, rune/int32 and int64s.
func NewMyers(a, b interface{}) *Myers {
	na, nb, err := configureAndValidate(a, b)
	if err != nil {
		panic(err)
	}
	return &Myers{
		a:  a,
		b:  b,
		na: na,
		nb: nb,
	}
}

// Details on the implementation and details of the algorithms can be found
// here:
// http://xmailserver.org/diff2.pdf
// http://simplygenius.net/Article/DiffTutorial1
// https://blog.robertelder.org/diff-algorithm/

func middleSnake(cmp comparator, na, nb int32) (d, x1, y1, x2, y2 int32) {
	max := na + nb // max # edits (delete all a, insert all of b)
	delta := int32(na - nb)

	odd := delta%2 != 0
	even := !odd

	// forward and reverse are accessed using k which is in the
	// range -d .. +d, hence offset must be added to k.
	forward := make([]int32, max+2)
	reverse := make([]int32, max+2)
	offset := int32(len(forward) / 2)

	// Only need to search for D halfway through the table.
	halfway := max / 2
	if max%2 != 0 {
		halfway += 1
	}
	for d := int32(0); d <= halfway; d++ {
		var x, y, mx, my int32
		// Forward search.
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
			for x < na && y < nb && cmp(int(x), int(y)) {
				x++
				y++
			}
			forward[offset+k] = x
			// Can this snake potentially overlap with one of the reverse ones?
			// Going forward only odd paths can be the longest ones.
			if odd && (-(k - delta)) >= -(d-1) && (-(k - delta)) <= (d-1) {
				// Doe this snake overlap with one of the reverse ones? If so,
				// the last snake is the longest one.
				if forward[offset+k]+reverse[offset-(k-delta)] >= int32(na) {
					return 2*d - 1, mx, my, x, y
				}
			}
		}

		// Reverse search.
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
			for x < na && y < nb && cmp(int(na-x-1), int(nb-y-1)) {
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
					return 2 * d, na - x, nb - y, na - mx, nb - my
				}
			}
		}
	}
	panic("unreachable")
}

func idxSlice(sa, na int32) []int {
	idx := make([]int, na)
	for i := range idx {
		idx[i] = int(sa) + i
	}
	return idx
}

func myersLCS64(a, b []int64) []int64 {
	na, nb := int32(len(a)), int32(len(b))
	if na == 0 || nb == 0 {
		return nil
	}
	d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
	if d > 1 {
		nd := myersLCS64(a[:x], b[:y])
		nd = append(nd, a[x:u]...)
		nd = append(nd, myersLCS64(a[u:na], b[v:nb])...)
		return nd
	}
	if nb > na {
		return a
	}
	return b
}

func myersLCS32(a, b []int32) []int32 {
	na, nb := int32(len(a)), int32(len(b))
	if na == 0 || nb == 0 {
		return nil
	}
	d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
	if d > 1 {
		nd := myersLCS32(a[:x], b[:y])
		nd = append(nd, a[x:u]...)
		nd = append(nd, myersLCS32(a[u:na], b[v:nb])...)
		return nd
	}
	if nb > na {
		return a
	}
	return b
}

func myersLCS8(a, b []uint8) []uint8 {
	na, nb := int32(len(a)), int32(len(b))
	if na == 0 || nb == 0 {
		return nil
	}
	d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
	if d > 1 {
		nd := myersLCS8(a[:x], b[:y])
		nd = append(nd, a[x:u]...)
		nd = append(nd, myersLCS8(a[u:na], b[v:nb])...)
		return nd
	}
	if nb > na {
		return a
	}
	return b
}

// LCS returns the longest common subsquence.
func (m *Myers) LCS() interface{} {
	switch av := m.a.(type) {
	case []int64:
		return myersLCS64(av, m.b.([]int64))
	case []int32:
		return myersLCS32(av, m.b.([]int32))
	case []uint8:
		return myersLCS8(av, m.b.([]uint8))
	}
	panic(fmt.Sprintf("unreachable: wrong type: %T", m.a))
}

/*
func myersSES64(a, b []int64, cx int32) []Edit {
	na, nb := int32(len(a)), int32(len(b))
	var ses []Edit
	if na > 0 && nb > 0 {
		d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
		if d > 1 || (x != u && y != v) {
			ses = append(ses, myersSES64(a[:x], b[:y], cx)...)
			return append(ses, myersSES64(a[u:], b[v:], cx+u)...)
		}
		if nb > na {
			return append(ses, myersSES64(nil, b[na:], cx+na)...)
		}
		if na > nb {
			return append(ses, myersSES64(a[nb:], nil, cx+nb)...)
		}
		return ses
	}
	if na > 0 {
		for i, val := range a {
			ses = append(ses, Edit{Delete, int(cx) + i, val})
		}
		return ses
	}
	for _, val := range b {
		ses = append(ses, Edit{Insert, floor0(int(cx) - 1), val})
	}
	return ses
}

func myersSES32(a, b []int32, cx int32) []Edit {
	na, nb := int32(len(a)), int32(len(b))
	var ses []Edit
	if na > 0 && nb > 0 {
		d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
		if d > 1 || (x != u && y != v) {
			ses = append(ses, myersSES32(a[:x], b[:y], cx)...)
			return append(ses, myersSES32(a[u:], b[v:], cx+u)...)
		}
		if nb > na {
			return append(ses, myersSES32(nil, b[na:], cx+na)...)
		}
		if na > nb {
			return append(ses, myersSES32(a[nb:], nil, cx+nb)...)
		}
		return ses
	}
	if na > 0 {
		for i, val := range a {
			ses = append(ses, Edit{Delete, int(cx) + i, val})
		}
		return ses
	}
	for _, val := range b {
		ses = append(ses, Edit{Insert, floor0(int(cx) - 1), val})
	}
	return ses
}

func myersSES8(a, b []uint8, cx int32) []Edit {
	na, nb := int32(len(a)), int32(len(b))
	var ses []Edit
	if na > 0 && nb > 0 {
		d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
		if d > 1 || (x != u && y != v) {
			ses = append(ses, myersSES8(a[:x], b[:y], cx)...)
			return append(ses, myersSES8(a[u:], b[v:], cx+u)...)
		}
		if nb > na {
			return append(ses, myersSES8(nil, b[na:], cx+na)...)
		}
		if na > nb {
			return append(ses, myersSES8(a[nb:], nil, cx+nb)...)
		}
		return ses
	}
	if na > 0 {
		for i, val := range a {
			ses = append(ses, Edit{Delete, int(cx) + i, val})
		}
		return ses
	}
	for _, val := range b {
		ses = append(ses, Edit{Insert, floor0(int(cx) - 1), val})
	}
	return ses
}
*/

func (m *Myers) ses(a, b interface{}, na, nb, cx, cy int32) []Edit {
	var ses []Edit
	if na > 0 && nb > 0 {
		d, x, y, u, v := middleSnake(cmpFor(a, b), na, nb)
		if d > 1 || (x != u && y != v) {
			ses = append(ses,
				m.ses(m.slicer(a, 0, x), m.slicer(b, 0, y), x, y, cx, cy)...)
			return append(ses,
				m.ses(m.slicer(a, u, na), m.slicer(b, v, nb), na-u, nb-v, cx+u, cy+v)...)
		}
		if nb > na {
			return append(ses,
				m.ses(nil, m.slicer(b, na, nb), 0, nb-na, cx+na, cy)...)
		}
		if na > nb {
			return append(ses,
				m.ses(m.slicer(a, nb, na), nil, na-nb, 0, cx+nb, cy)...)
		}
		return ses
	}
	if na > 0 {
		return m.deletions(a, int(cx))
	}
	return m.insertions(b, int(cx), int(cy))
}

// SES returns the shortest edit script.
func (m *Myers) SES() EditScript {
	switch m.a.(type) {
	case []int64:
		m.slicer = func(v interface{}, from, to int32) interface{} {
			return v.([]int64)[from:to]
		}
		m.deletions = func(v interface{}, cx int) []Edit {
			var edits []Edit
			for i, val := range v.([]int64) {
				edits = append(edits, Edit{Delete, cx + i, 0, val})
			}
			return edits
		}
		m.insertions = func(v interface{}, cx, cy int) []Edit {
			var edits []Edit
			for i, val := range v.([]int64) {
				edits = append(edits, Edit{Insert, floor0(int(cx) - 1), int(cy) + i, val})
			}
			return edits
		}
	case []int32:
		m.slicer = func(v interface{}, from, to int32) interface{} {
			return v.([]int32)[from:to]
		}
		m.deletions = func(v interface{}, cx int) []Edit {
			var edits []Edit
			for i, val := range v.([]int32) {
				edits = append(edits, Edit{Delete, cx + i, 0, val})
			}
			return edits
		}
		m.insertions = func(v interface{}, cx, cy int) []Edit {
			var edits []Edit
			for i, val := range v.([]int32) {
				edits = append(edits, Edit{Insert, floor0(int(cx) - 1), int(cy) + i, val})
			}
			return edits
		}
	case []uint8:
		m.slicer = func(v interface{}, from, to int32) interface{} {
			return v.([]uint8)[from:to]
		}
		m.deletions = func(v interface{}, cx int) []Edit {
			var edits []Edit
			for i, val := range v.([]uint8) {
				edits = append(edits, Edit{Delete, cx + i, 0, val})
			}
			return edits
		}
		m.insertions = func(v interface{}, cx, cy int) []Edit {
			var edits []Edit
			for i, val := range v.([]uint8) {
				edits = append(edits, Edit{Insert, floor0(int(cx) - 1), int(cy) + i, val})
			}
			return edits
		}
	default:
		panic(fmt.Sprintf("unreachable: wrong type: %T", m.a))
	}
	return m.ses(m.a, m.b, int32(m.na), int32(m.nb), 0, 0)
}
