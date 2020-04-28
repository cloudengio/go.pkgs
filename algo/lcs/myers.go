package lcs

type Myers struct {
	a, b  []int32
	table []int
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// NewMyers returns a solver using the linear time/space variant of Myer's
// original algorithm as documented in:
// An O(ND) Difference Algorithm and Its Variations, 1986.
func NewMyers(a, b []byte, decoder Decoder) *Myers {
	da, db := decode(a, 0, decoder), decode(b, 0, decoder)
	return &Myers{
		a: da,
		b: db,
	}
}

// Details on the implementation and details of the algorithms can be
// found here:
// http://xmailserver.org/diff2.pdf
// http://simplygenius.net/Article/DiffTutorial1
// https://blog.robertelder.org/diff-algorithm/

// myersMiddle finds the middle snake.
func myersMiddle(a, b []int32) (d, x1, y1, x2, y2 int32) {
	na, nb := int32(len(a)), int32(len(b))
	max := na + nb // max # edits (delete all a, insert all of b)
	delta := na - nb

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
					return 2 * d, na - x, nb - y, na - mx, nb - my
				}
			}
		}
	}
	panic("unreachable")
}

// myersLCS returns the LCS
func myersLCS(a, b []int32) []int32 {
	na, nb := len(a), len(b)
	if na == 0 || nb == 0 {
		return nil
	}
	d, x, y, u, v := myersMiddle(a, b)
	if d > 1 {
		nd := myersLCS(a[:x], b[:y])
		nd = append(nd, a[x:u]...)
		nd = append(nd, myersLCS(a[u:na], b[v:nb])...)
		return nd
	}
	if nb > na {
		return a
	}
	return b
}

func (differ *Myers) LCS() []int32 {
	return myersLCS(differ.a, differ.b)
}

func (differ *Myers) SESLen() int {
	return len(myersLCS(differ.a, differ.b))
}

/*
func (differ *Myers) GreedySESLen() int {
	a, b := differ.a, differ.b
	na, nb := len(differ.a), len(differ.b)
	ndiag := (na + nb)
	contour := make([]int, (ndiag*2)+3)
	offset := (len(contour) / 2)
	for d := 0; d <= ndiag; d++ {
		for k := -d; k <= d; k += 2 {
			var x int
			// Edge cases are:
			// k == -d    - move down
			// k == d * 2 - move right
			// Normal case:
			// move down or right depending on how far the move would be.
			if k == -d || k != d*2 && contour[offset+k-1] < contour[offset+k+1] {
				x = contour[offset+k+1] // move down
			} else {
				x = contour[offset+k-1] + 1 // move right
			}
			y := x - k
			for x < na && y < nb && a[x] == b[y] {
				x++
				y++
			}
			contour[offset+k] = x
			if x >= na && y >= nb {
				return d
			}
		}
	}
	return 0
}
*/
