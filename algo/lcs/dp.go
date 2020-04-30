package lcs

import (
	"fmt"
	"io"
)

// DP represents an LCS/SES solver that uses dynamic programming.
// See https://en.wikipedia.org/wiki/Longest_common_subsequence_problem.
type DP struct {
	a, b     interface{}
	na, nb   int
	cmp      comparator
	lookupA  accessor
	append   appendor
	newSlice func() interface{}

	filled bool

	// Store only the directions in this table. For now, use 8 bits
	// though 2 would suffice.
	directions [][]uint8
}

// NewDP creates a new dynamic programming based implementation for finding
// the longest common subsequence and shortest edit script (LCS/SES) for
// transforming A to B. This implementaion can return all LCS and SES rather
// than just the first one found. If a single LCS or SES is sufficient then
// the Myer's algorithm implementation is lilkey a better choice.
func NewDP(a, b interface{}) *DP {
	na, nb, err := configureAndValidate(a, b)
	if err != nil {
		panic(err)
	}
	dp := &DP{
		a:        a,
		b:        b,
		na:       na,
		nb:       nb,
		cmp:      cmpFor(a, b),
		lookupA:  accessorFor(a),
		newSlice: newSliceFor(a),
		append:   appendorFor(a),
	}
	if na == 0 || nb == 0 {
		// Leave dp.directions as nil to indicate that one or other
		// input is empty.
		return dp
	}
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

	upArrow       rune = 0x2191 // utf8 up arrow
	leftArrow     rune = 0x2190 // utf8 left arrow
	diagonalArrow rune = 0x2196 // utf8 diagonal arrow
	space         rune = 0x20   // utf8 space
)

func (p *DP) LCS() interface{} {
	if p.directions == nil {
		return p.newSlice()
	}
	p.fill()
	return p.backtrack(p.directions, p.na-1, p.nb-1)
}

func (p *DP) All() []interface{} {
	if p.directions == nil {
		return nil
	}
	p.fill()
	return p.backtrackAll(p.directions, p.na-1, p.nb-1)
}

// SES returns the shortest edit script to turn A into B.
func (p *DP) SES() EditScript {
	if p.directions == nil {
		return nil
	}
	return EditScript(diff(p.directions, p.na-1, p.nb-1))
}

func (p *DP) fill() {
	if p.filled {
		return
	}
	p.filled = true
	table := make([][]int32, p.na+1)
	for i := range table {
		table[i] = make([]int32, p.nb+1)
	}
	for x := 1; x < len(p.directions); x++ {
		for y := 1; y < len(p.directions[x]); y++ {
			if p.cmp(x-1, y-1) {
				table[x][y] = (table[x-1][y-1]) + 1
				p.directions[x][y] = diagonal
				continue
			}
			prevUp := table[x-1][y]
			prevLeft := table[x][y-1]
			switch {
			case prevLeft < prevUp:
				table[x][y] = prevUp
				p.directions[x][y] = up
			case prevLeft > prevUp:
				table[x][y] = prevLeft
				p.directions[x][y] = left
			default:
				table[x][y] = prevLeft
				p.directions[x][y] = upAndLeft
			}
		}
	}
}

func (p *DP) backtrack(directions [][]uint8, i, j int) interface{} {
	if i == 0 || j == 0 {
		return p.newSlice()
	}
	var dir uint8
	dir = directions[i][j]
	if dir == diagonal {
		return p.append(p.backtrack(directions, i-1, j-1), p.lookupA(i))
	}
	if dir == up {
		return p.backtrack(directions, i-1, j)
	}
	return p.backtrack(directions, i, j-1)
}

func (p *DP) backtrackAll(directions [][]uint8, i, j int) []interface{} {
	fmt.Printf("%v %v\n", i, j)
	if i == 0 || j == 0 {
		return nil
	}
	var dir uint8
	dir = directions[i][j]
	if dir == diagonal {
		paths := p.backtrackAll(directions, i-1, j-1)
		val := p.lookupA(i)
		//fmt.Printf("V: %c\n", val)
		if len(paths) == 0 {
			np := p.append(p.newSlice(), val)
			//fmt.Printf("P 1: %c\n", np)
			return []interface{}{np}
		}
		for i, path := range paths {
			paths[i] = p.append(path, val)
		}
		//		fmt.Printf("P 2: %#v\n", paths)
		return paths
	}
	var paths []interface{}
	if dir == up || dir == upAndLeft {
		paths = p.backtrackAll(directions, i-1, j)
	}
	if dir == left || dir == upAndLeft {
		paths = append(paths, p.backtrackAll(directions, i, j-1)...)
	}
	return paths
}

func diff(directions [][]uint8, i, j int) []Edit {
	dir := directions[i][j]
	if i > 0 && j > 0 && dir == diagonal {
		return append(diff(directions, i-1, j-1), Edit{Same, i})
	}
	if j > 0 && (i == 0 || dir == up || dir == upAndLeft) {
		return append(diff(directions, i, j-1), Edit{Add, j})
	}
	if i > 0 && (j == 0 || dir == left || dir == upAndLeft) {
		return append(diff(directions, i-1, j), Edit{Delete, i})
	}
	return nil
}

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
	//out.Write([]byte("   "))
	//	for _, c := range p.b[1:] {
	//		out.Write([]byte(fmt.Sprintf(" %3v ", c)))
	//	}
	//out.Write([]byte(fmt.Sprintf("%v\n", p.a)))
	out.Write([]byte("\n"))
	for a := 1; a < len(p.directions); a++ {
		//out.Write([]byte(fmt.Sprintf("%vc ", p.a[a])))
		for b := 1; b < len(p.directions[a]); b++ {
			dir := p.directions[a][b]
			str := fmt.Sprintf("  %c%c ", firstArrow(dir), secondArrow(dir))
			out.Write([]byte(str))
		}
		out.Write([]byte("\n"))
	}
}
