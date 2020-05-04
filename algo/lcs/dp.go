package lcs

import (
	"fmt"
)

// DP represents a new dynamic programming based implementation for finding
// the longest common subsequence and shortest edit script (LCS/SES) for
// transforming A to B.
// See https://en.wikipedia.org/wiki/Longest_common_subsequence_problem.
// This implementaion can return all LCS and SES rather than just the first
// one found. If a single LCS or SES is sufficient then the Myer's algorithm
// implementation is lilkey a better choice.
type DP struct {
	a, b    interface{}
	na, nb  int
	cmp     comparator
	lookupA accessor

	filled bool

	// Store only the directions in this table. For now, use 8 bits
	// though 2 would suffice.
	directions [][]uint8
}

// NewDP creates a new instance of DP.
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

func (dp *DP) LCS() interface{} {
	dp.fill()
	switch a := dp.a.(type) {
	case []int32:
		return dp.backtrack32(a, dp.na, dp.nb)
	case []uint8:
		return dp.backtrack8(a, dp.na, dp.nb)
	}
	panic(fmt.Sprintf("unsupported type %T\n", dp.a))
}

func (dp *DP) AllLCS() interface{} {
	dp.fill()
	switch a := dp.a.(type) {
	case []int32:
		return dp.backtrackAll32(a, dp.na, dp.nb)
	case []uint8:
		return dp.backtrackAll8(a, dp.na, dp.nb)
	}
	panic(fmt.Sprintf("unsupported type %T\n", dp.a))
}

// SES returns the shortest edit script to turn A into B.
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

func (dp *DP) backtrack32(a []int32, i, j int) []int32 {
	if i == 0 || j == 0 {
		return nil
	}
	switch dp.directions[i][j] {
	case diagonal:
		return append(dp.backtrack32(a, i-1, j-1), a[i-1])
	case up, upAndLeft:
		return dp.backtrack32(a, i, j-1)
	}
	return dp.backtrack32(a, i-1, j)
}

func (dp *DP) backtrack8(a []uint8, i, j int) []uint8 {
	if i == 0 || j == 0 {
		return nil
	}
	switch dp.directions[i][j] {
	case diagonal:
		return append(dp.backtrack8(a, i-1, j-1), a[i-1])
	case up, upAndLeft:
		return dp.backtrack8(a, i, j-1)
	}
	return dp.backtrack8(a, i-1, j)
}

func (dp *DP) backtrackAll32(a []int32, i, j int) [][]int32 {
	if i == 0 || j == 0 {
		return nil
	}
	dir := dp.directions[i][j]
	if dir == diagonal {
		val := a[i-1]
		paths := dp.backtrackAll32(a, i-1, j-1)
		if len(paths) == 0 {
			return [][]int32{{val}}
		}
		for i, path := range paths {
			paths[i] = append(path, val)
		}
		return paths
	}
	var paths [][]int32
	if dir == up || dir == upAndLeft {
		paths = dp.backtrackAll32(a, i, j-1)
	}
	if dir == left || dir == upAndLeft {
		paths = append(paths, dp.backtrackAll32(a, i-1, j)...)
	}
	return paths
}

func (dp *DP) backtrackAll8(a []uint8, i, j int) [][]uint8 {
	if i == 0 || j == 0 {
		return nil
	}
	dir := dp.directions[i][j]
	if dir == diagonal {
		val := a[i-1]
		paths := dp.backtrackAll8(a, i-1, j-1)
		if len(paths) == 0 {
			return [][]uint8{{val}}
		}
		for i, path := range paths {
			paths[i] = append(path, val)
		}
		return paths
	}
	var paths [][]uint8
	if dir == up || dir == upAndLeft {
		paths = dp.backtrackAll8(a, i, j-1)
	}
	if dir == left || dir == upAndLeft {
		paths = append(paths, dp.backtrackAll8(a, i-1, j)...)
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
		return dp.diff(b, i-1, j-1)
	}
	if j > 0 && (i == 0 || dir == up || dir == upAndLeft) {
		return append(dp.diff(b, i, j-1), Edit{Insert, floor0(i - 1), b(j - 1)})
	}
	if i > 0 && (j == 0 || dir == left) {
		return append(dp.diff(b, i-1, j), Edit{Delete, i - 1, 0})
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
