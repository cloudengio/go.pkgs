// Package lcs provides an implementation of the longest common subsequence
// algorithm suitable for use with unicode/utf8 and other alphabets.
package lcs

import (
	"fmt"
	"io"
	"os"
)

type Decoder func([]byte) (int32, int)

type Pair struct {
	decoder      Decoder
	a, b         []int32
	table        [][]uint32
	directions   [][]uint8
	prevA, prevB []int32
}

// decode includes a leading 'blank' slot to avoud the need for special casing
// the initial comparisons when filling the table.
func decode(input []byte, decoder Decoder) []int32 {
	li := len(input)
	tmp := make([]int32, li+1)
	cursor := 0
	i := 1
	for {
		tok, n := decoder(input[cursor:])
		tmp[i] = tok
		i++
		cursor += n
		if cursor >= li {
			break
		}
	}
	return tmp[:i]
}

func New(a, b []byte, decoder Decoder) *Pair {
	da, db := decode(a, decoder), decode(b, decoder)
	if len(da) > MaxDecodedLength {
		panic(fmt.Sprintf("a is too large: %v > %v", da, MaxDecodedLength))
	}
	if len(db) > MaxDecodedLength {
		panic(fmt.Sprintf("b is too large: %v > %v", db, MaxDecodedLength))
	}
	table := make([][]uint32, len(da))
	for i := range table {
		table[i] = make([]uint32, len(db))
	}

	directions := make([][]int8)

	return &Pair{
		a:     da,
		b:     db,
		table: table,
		/*		prevA: make([]int32, len(da)),
				prevB: make([]int32, len(db)),*/
	}
}

const (
	//  MaxDecodedLength represents the maxium size of the decoded
	// 'strings' that can be compared.
	MaxDecodedLength = 1 << 27

	// store the 'direction' in the top 4 bits of an int32 and keep the
	// lower 28 bits for the length.
	lengthMask  uint32 = 0x0fffffff
	arrowMask   uint32 = 0xf0000000
	arrowOffset int    = 28
	up          uint32 = 0x1 << arrowOffset
	left        uint32 = 0x2 << arrowOffset
	diagonal    uint32 = 0x4 << arrowOffset

	upArrow       rune = 0x2191 // utf8 up arrow
	leftArrow     rune = 0x2190 // utf8 left arrow
	diagonalArrow rune = 0x2196 // utf8 diagonal arrow
	space         rune = 0x20   // utf8 space
)

func firstArrow(v uint32) rune {
	if v&left != 0 {
		return leftArrow
	}
	return space
}

func secondArrow(v uint32) rune {
	if v&up != 0 {
		return upArrow
	}
	if v&diagonal != 0 {
		return diagonalArrow
	}
	return space
}

func seqlen(v uint32) uint32 {
	return v & lengthMask
}

func (p *Pair) Find() [][]int32 {
	p.fill()
	p.print(os.Stdout)
	return p.backtrack(len(p.a)-1, len(p.b)-1)
}

func (p *Pair) print(out io.Writer) {
	out.Write([]byte("    "))
	for _, c := range p.b[1:] {
		out.Write([]byte(fmt.Sprintf("%10c  ", c)))
	}
	out.Write([]byte("\n"))
	for a := 1; a < len(p.table); a++ {
		out.Write([]byte(fmt.Sprintf("%3c", p.a[a])))
		for b := 1; b < len(p.table[a]); b++ {
			v := p.table[a][b]
			str := fmt.Sprintf("% 10d%c%c", seqlen(v), firstArrow(v), secondArrow(v))
			out.Write([]byte(str))
		}
		out.Write([]byte("\n"))
	}
}

func (p *Pair) fill() {
	for a := 1; a < len(p.table); a++ {
		for b := 1; b < len(p.table[a]); b++ {
			va, vb := p.a[a], p.b[b]
			if va == vb {
				p.table[a][b] = (seqlen(p.table[a-1][b-1]) + 1) | diagonal
				continue
			}
			prevUp := seqlen(p.table[a-1][b])
			prevLeft := seqlen(p.table[a][b-1])
			switch {
			case prevLeft < prevUp:
				p.table[a][b] = prevUp | up
			case prevLeft > prevUp:
				p.table[a][b] = prevLeft | left
			default:
				p.table[a][b] = prevLeft | up | left
			}
		}
	}
}

func extend(paths [][]int32, tok int32) [][]int32 {
	if len(paths) == 0 {
		first := []int32{tok}
		return [][]int32{first}
	}
	for i, p := range paths {
		paths[i] = append(p, tok)
	}
	return paths
}

func (p *Pair) backtrack(i, j int) [][]int32 {
	if i == 0 || j == 0 {
		return nil
	}
	dir := p.table[i][j]
	if dir&diagonal != 0 {
		return extend(p.backtrack(i-1, j-1), p.a[i])
	}
	var paths [][]int32
	if dir&up != 0 {
		paths = p.backtrack(i-1, j)
	}
	if dir&left != 0 {
		paths = append(paths, p.backtrack(i, j-1)...)
	}
	return paths
}
