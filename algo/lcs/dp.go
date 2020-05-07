package lcs

import (
	"fmt"
)

// DP represents a dynamic programming based implementation for finding
// the longest common subsequence and shortest edit script (LCS/SES) for
// transforming A to B.
// See https://en.wikipedia.org/wiki/Longest_common_subsequence_problem.
// This implementation can return all LCS and SES rather than just the first
// one found. If a single LCS or SES is sufficient then the Myer's algorithm
// implementation is lilkey a better choice.
type DP struct {
	a, b      interface{}
	na, nb    int
	cmp       comparator
	lookupA   accessor
	emptyAll  func() interface{}
	appendAll func(a, b interface{}) interface{}
	extend    func(i int, b interface{}) interface{}
	empty     func() interface{}
	append    func(a, b interface{}) interface{}

	filled bool

	// Store only the directions in this table. For now, use 8 bits
	// though 2 would suffice.
	directions [][]uint8
}

// NewDP creates a new instance of DP. The implementation supports slices
// of bytes/uint8, rune/int32 and int64s.
func NewDP(a, b interface{}) *DP {
	na, nb, err := configureAndValidate(a, b)
	if err != nil {
		panic(err)
	}
	dp := &DP{
		a:       a,
		b:       b,
		na:      na,
		nb:      nb,
		cmp:     cmpFor(a, b),
		lookupA: accessorFor(a),
	}
	// A will be the X and b the Y axis.
	// The directions/score matrix has an extra 0th row/column to
	// simplify bounds checking.
	directions := make([][]uint8, na+1)
	for i := range directions {
		directions[i] = make([]uint8, nb+1)
	}
	dp.directions = directions
	return dp
}

const (
	diagonal  uint8 = 0x0
	up        uint8 = 0x1
	left      uint8 = 0x2
	upAndLeft uint8 = 0x3
)

// LCS returns the longest common subsquence.
func (dp *DP) LCS() interface{} {
	dp.fill()
	switch dp.a.(type) {
	case []int64:
		dp.empty = func() interface{} {
			return []int64{}
		}
		dp.append = func(a, b interface{}) interface{} {
			return append(a.([]int64), b.(int64))
		}
	case []int32:
		dp.empty = func() interface{} {
			return []int32{}
		}
		dp.append = func(a, b interface{}) interface{} {
			return append(a.([]int32), b.(int32))
		}
	case []uint8:
		dp.empty = func() interface{} {
			return []uint8{}
		}
		dp.append = func(a, b interface{}) interface{} {
			return append(a.([]uint8), b.(uint8))
		}
	default:
		panic(fmt.Sprintf("unsupported type %T\n", dp.a))

	}
	return dp.backtrack(dp.na, dp.nb)
}

// AllLCS returns all of the the longest common subsquences.
func (dp *DP) AllLCS() interface{} {
	dp.fill()
	switch a := dp.a.(type) {
	case []int64:
		dp.emptyAll = func() interface{} {
			return [][]int64{}
		}
		dp.appendAll = func(a, b interface{}) interface{} {
			return append(a.([][]int64), b.([][]int64)...)
		}
		dp.extend = func(i int, p interface{}) interface{} {
			sl := p.([][]int64)
			v := a[i]
			if len(sl) == 0 {
				return [][]int64{{v}}
			}
			for i, p := range sl {
				sl[i] = append(p, v)
			}
			return sl
		}
	case []int32:
		dp.emptyAll = func() interface{} {
			return [][]int32{}
		}
		dp.appendAll = func(a, b interface{}) interface{} {
			return append(a.([][]int32), b.([][]int32)...)
		}
		dp.extend = func(i int, p interface{}) interface{} {
			sl := p.([][]int32)
			v := a[i]
			if len(sl) == 0 {
				return [][]int32{{v}}
			}
			for i, p := range sl {
				sl[i] = append(p, v)
			}
			return sl
		}
	case []uint8:
		dp.emptyAll = func() interface{} {
			return [][]uint8{}
		}
		dp.appendAll = func(a, b interface{}) interface{} {
			return append(a.([][]uint8), b.([][]uint8)...)
		}
		dp.extend = func(i int, p interface{}) interface{} {
			sl := p.([][]uint8)
			v := a[i]
			if len(sl) == 0 {
				return [][]uint8{{v}}
			}
			for i, p := range sl {
				sl[i] = append(p, v)
			}
			return sl
		}

	default:
		panic(fmt.Sprintf("unsupported type %T\n", dp.a))
	}
	return dp.backtrackAll(dp.na, dp.nb)

}

// SES returns the shortest edit script.
func (dp *DP) SES() EditScript {
	if dp.directions == nil {
		return EditScript{}
	}
	dp.fill()
	return EditScript(dp.diff(accessorFor(dp.b), dp.na, dp.nb))
}

func (dp *DP) fill() {
	if dp.filled {
		return
	}
	dp.filled = true

	table := make([][]int32, dp.na+1)
	for i := range table {
		table[i] = make([]int32, dp.nb+1)
	}
	for y := 1; y <= dp.nb; y++ {
		for x := 1; x <= dp.na; x++ {
			if dp.cmp(x-1, y-1) {
				table[x][y] = table[x-1][y-1] + 1
				dp.directions[x][y] = diagonal
				continue
			}
			prevLeft := table[x-1][y]
			prevUp := table[x][y-1]
			switch {
			case prevUp > prevLeft:
				dp.directions[x][y] = up
				table[x][y] = prevUp
			case prevLeft > prevUp:
				dp.directions[x][y] = left
				table[x][y] = prevLeft
			default:
				dp.directions[x][y] = upAndLeft
				table[x][y] = prevLeft
			}
		}
	}
}

func (dp *DP) backtrack(i, j int) interface{} {
	if i == 0 || j == 0 {
		return dp.empty()
	}
	switch dp.directions[i][j] {
	case diagonal:
		return dp.append(dp.backtrack(i-1, j-1), dp.lookupA(i-1))
	case up, upAndLeft:
		return dp.backtrack(i, j-1)
	}
	return dp.backtrack(i-1, j)
}

func (dp *DP) backtrackAll(i, j int) interface{} {
	if i == 0 || j == 0 {
		return dp.emptyAll()
	}
	dir := dp.directions[i][j]
	if dir == diagonal {
		return dp.extend(i-1, dp.backtrackAll(i-1, j-1))
	}
	paths := dp.emptyAll()
	if dir == up || dir == upAndLeft {
		paths = dp.backtrackAll(i, j-1)
	}
	if dir == left || dir == upAndLeft {
		paths = dp.appendAll(paths, dp.backtrackAll(i-1, j))
	}
	return paths
}

func floor0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}

func (dp *DP) diff(b accessor, i, j int) []Edit {
	dir := dp.directions[i][j]
	if i > 0 && j > 0 && dir == diagonal {
		return append(dp.diff(b, i-1, j-1), Edit{Identical, i - 1, j - 1, b(j - 1)})
	}
	if j > 0 && (i == 0 || dir == up || dir == upAndLeft) {
		return append(dp.diff(b, i, j-1), Edit{Insert, floor0(i - 1), j - 1, b(j - 1)})
	}
	if i > 0 && (j == 0 || dir == left) {
		return append(dp.diff(b, i-1, j), Edit{Delete, i - 1, -1, 0})
	}
	return nil
}

/*
const (
	upArrow       rune = 0x2191 // utf8 up arrow
	leftArrow     rune = 0x2190 // utf8 left arrow
	diagonalArrow rune = 0x2196 // utf8 diagonal arrow
	space         rune = 0x20   // utf8 space
)

func firstArrow(v uint8) rune {
	if v == left || v == upAndLeft {
		return leftArrow
	}
	return space
}

func secondArrow(v uint8) rune {
	switch v {
	case up, upAndLeft:
		return upArrow
	case diagonal:
		return diagonalArrow
	default:
		return space
	}
}

func (p *DP) print(out io.Writer) {
	mx, my := p.na, p.nb
	row := &strings.Builder{}
	for y := 0; y < my; y++ {
		for x := 0; x < mx; x++ {
			dir := p.directions[x][y]
			row.WriteString(fmt.Sprintf("  %c%c ", firstArrow(dir), secondArrow(dir)))
		}
		row.WriteString("\n")
		out.Write([]byte(row.String()))
		row.Reset()
	}
}
*/
