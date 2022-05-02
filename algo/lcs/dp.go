// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package lcs

// DP represents a dynamic programming based implementation for finding
// the longest common subsequence and shortest edit script (LCS/SES) for
// transforming A to B.
// See https://en.wikipedia.org/wiki/Longest_common_subsequence_problem.
// This implementation can return all LCS and SES rather than just the first
// one found. If a single LCS or SES is sufficient then the Myer's algorithm
// implementation is lilkey a better choice.
type DP[T comparable] struct {
	a, b   []T
	filled bool

	// Store only the directions in this table. For now, use 8 bits
	// though 2 would suffice.
	directions [][]uint8
}

// NewDP creates a new instance of DP. The implementation supports slices
// of bytes/uint8, rune/int32 and int64s.
func NewDP[T comparable](a, b []T) *DP[T] {
	dp := &DP[T]{a: a, b: b}
	// a will be the X and b the Y axis.
	// The directions/score matrix has an extra 0th row/column to
	// simplify bounds checking.
	directions := make([][]uint8, len(a)+1)
	for i := range directions {
		directions[i] = make([]uint8, len(b)+1)
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
func (dp *DP[T]) LCS() []T {
	dp.fill()
	return dp.backtrack(len(dp.a), len(dp.b))
}

// AllLCS returns all of the the longest common subsquences.
func (dp *DP[T]) AllLCS() [][]T {
	dp.fill()
	return dp.backtrackAll(len(dp.a), len(dp.b))
}

// SES returns the shortest edit script.
func (dp *DP[T]) SES() *EditScript[T] {
	if dp.directions == nil {
		return &EditScript[T]{}
	}
	dp.fill()
	var es EditScript[T] = dp.diff(len(dp.a), len(dp.b))
	return &es
}

func (dp *DP[T]) fill() {
	if dp.filled {
		return
	}
	dp.filled = true

	table := make([][]int32, len(dp.a)+1)
	for i := range table {
		table[i] = make([]int32, len(dp.b)+1)
	}
	for y := 1; y <= len(dp.b); y++ {
		for x := 1; x <= len(dp.a); x++ {
			if dp.a[x-1] == dp.b[y-1] {
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

func (dp *DP[T]) backtrack(i, j int) []T {
	if i == 0 || j == 0 {
		return []T{}
	}
	switch dp.directions[i][j] {
	case diagonal:
		return append(dp.backtrack(i-1, j-1), dp.a[i-1])
	case up, upAndLeft:
		return dp.backtrack(i, j-1)
	}
	return dp.backtrack(i-1, j)
}

func (dp *DP[T]) extend(i int, bt [][]T) [][]T {
	if len(bt) == 0 {
		return [][]T{{dp.a[i]}}
	}
	for i, p := range bt {
		bt[i] = append(p, dp.a[i])
	}
	return bt
}

func (dp *DP[T]) backtrackAll(i, j int) [][]T {
	if i == 0 || j == 0 {
		return [][]T{}
	}
	dir := dp.directions[i][j]
	if dir == diagonal {
		return dp.extend(i-1, dp.backtrackAll(i-1, j-1))
	}
	paths := [][]T{}
	if dir == up || dir == upAndLeft {
		paths = dp.backtrackAll(i, j-1)
	}
	if dir == left || dir == upAndLeft {
		paths = append(paths, dp.backtrackAll(i-1, j)...)
	}
	return paths
}

func (dp *DP[T]) diff(i, j int) []Edit[T] {
	dir := dp.directions[i][j]
	if i > 0 && j > 0 && dir == diagonal {
		return append(dp.diff(i-1, j-1), Edit[T]{Identical, i - 1, j - 1, dp.b[j-1]})
	}
	if j > 0 && (i == 0 || dir == up || dir == upAndLeft) {
		return append(dp.diff(i, j-1), Edit[T]{Insert, i, j - 1, dp.b[j-1]})
	}
	if i > 0 && (j == 0 || dir == left) {
		return append(dp.diff(i-1, j), Edit[T]{Delete, i - 1, j, dp.a[i-1]})
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
