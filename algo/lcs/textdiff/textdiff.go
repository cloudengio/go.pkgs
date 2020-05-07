// Package textdiff providers support for diff'ing text.
package textdiff

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
	"unicode/utf8"

	"cloudeng.io/algo/codec"
	"cloudeng.io/algo/lcs"
)

// LineFNVHashDecoder decodes a byte slice into newline delimited blocks each
// of which is represented by a 64 bit hash obtained from fnv.New64a.
func LineFNVHashDecoder(data []byte) (string, int64, int) {
	if len(data) == 0 {
		return "", 0, 0
	}
	idx := bytes.Index(data, []byte{'\n'})
	if idx < 0 {
		idx = len(data) - 1
	}
	h := fnv.New64a()
	h.Write(data[:idx])
	sum := h.Sum64()
	return string(data[:idx]), int64(sum), idx + 1
}

// LineDecoder represents a decoder that can be used to split a byte stream
// into lines for use with the cloudeng.io/algo/lcs package.
type LineDecoder struct {
	lines  []string
	hashes []uint64
	fn     func([]byte) (string, int64, int)
}

func NewLineDecoder(fn func(data []byte) (string, int64, int)) *LineDecoder {
	return &LineDecoder{fn: fn}
}

func (ld *LineDecoder) Decode(data []byte) (int64, int) {
	line, sum, n := LineFNVHashDecoder(data)
	ld.lines = append(ld.lines, line)
	ld.hashes = append(ld.hashes, uint64(sum))
	return int64(sum), n
}

func (ld *LineDecoder) NumLines() int {
	return len(ld.lines)
}

func (ld *LineDecoder) Line(i int) (string, uint64) {
	return ld.lines[i], ld.hashes[i]
}

func text(orig []string, lines []int) string {
	out := strings.Builder{}
	for _, l := range lines {
		out.WriteString(orig[l])
		out.WriteString("\n")
	}
	return out.String()
}

func lineRange(lines []int) string {
	nl := len(lines)
	switch nl {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%d", lines[0]+1)
	default:
		return fmt.Sprintf("%d,%d", lines[0]+1, lines[len(lines)-1]+1)
	}
}

// getEditsForGroup splts edits into groups of contiguous of related edit
// operations.
// 1. groups split at an 'identical' boundary
// 2. multiple insertions at a single point
// 3. runs of contiguous deletions
func getEditsForGroup(edits lcs.EditScript) (group, script lcs.EditScript) {
	last := 0
	firstIns, firstDel := true, true
	prevIns, prevDel := 0, 0
	for i, edit := range edits {
		switch edit.Op {
		case lcs.Identical:
			goto done
		case lcs.Insert:
			// multiple insertions
			if firstIns {
				prevIns = edit.A
			}
			if prevIns != edit.A {
				last = i
				goto done
			}
			firstIns = false
		case lcs.Delete:
			if firstDel {
				prevDel = edit.A
			}
			if prevDel != edit.A {
				last = i
				goto done
			}
			prevDel++
			firstDel = false
		}

	}
done:
	return edits[:last+1], edits[last+1:]
}

/*
func getLinesForGroup(edits lcs.EditScript) (a, b []int) {
	for edit := range edits {
		switch edit.Op {
		case lcs.Delete:
			a = append(a, edit.A)
		case lcs.Insert:
			b = append(b, edit.B)
		}
		last = i
	}
	return a, b, edits[last+1:]
}*/

type Group struct {
	edits                 lcs.EditScript
	insertions, deletions lcs.EditScript

	insertedLines, deletedLines []int

	//linesA, linesB []int
	//textA, textB   string

	//	deletions, insertions []int
	//	decodedA, decodedB    []int32
}

/*
func linemap(a []int) map[int]bool {
	m := map[int]bool{}
	for _, l := range a {
		m[l] = true
	}
	return m
}
*/

func (d *Diff) newGroup(edits lcs.EditScript) *Group {
	ins, dels := []lcs.Edit{}, []lcs.Edit{}
	for _, e := range edits {
		switch e.Op {
		case lcs.Insert:
			ins = append(ins, e)
		case lcs.Delete:
			dels = append(dels, e)
		}
	}
	return &Group{
		insertions: ins,
		deletions:  dels,
		edits:      edits,
	}
	/*
		la, lb := linesFromScript(edits)
		linesA: la,
			linesB: lb,
			textA:  text(d.linesA, la),
			textB:  text(d.linesB, lb),
		ma, mb := linemap(a), linemap(b)
		deletions, insertions := []int{}, []int{}
		for la := range ma {
			deletions = append(deletions, la)
		}
		for lb := range mb {
			insertions = append(insertions, lb)
		}
		textA := text(d.linesA, a)
		textB := text(d.linesB, b)
		//da, db := d.utf8Decoder.Decode([]byte(textA)), d.utf8Decoder.Decode([]byte(textB))
		//decodedA:   da.([]int32),
		//decodedB:   db.([]int32),
		//edits:      lcs.NewMyers(da, db).SES(),
		return &Group{
			linesA:     a,
			linesB:     b,
			textA:      textA,
			textB:      textB,
			deletions:  deletions,
			insertions: insertions,
		}*/
}

func (g *Group) Summary() string {
	/*	ins, dels := g.insertions, g.deletions
		ni, nd := len(ins), len(dels)
		switch {
		case ni == nd:
			return fmt.Sprintf("%sc%s", lineRange(dels), lineRange(ins))
		case ni > 0:
			return "a" + lineRange(ins)
		case nd > 0:
			return "d" + lineRange(dels)
		}*/
	return g.edits.String()
}

type Diff struct {
	utf8Decoder    codec.Decoder
	linesA, linesB []string
	groups         []*Group
}

// Same returns true if there were no diffs.
func (d *Diff) Same() bool {
	return len(d.groups) == 0
}

func (d *Diff) NumGroups() int {
	return len(d.groups)
}

func (d *Diff) Group(i int) *Group {
	return d.groups[i]
}

func DiffByLines(a, b []byte) *Diff {
	lda, ldb := NewLineDecoder(LineFNVHashDecoder), NewLineDecoder(LineFNVHashDecoder)
	decA, err := codec.NewDecoder(lda.Decode)
	if err != nil {
		panic(err)
	}
	decB, _ := codec.NewDecoder(ldb.Decode)
	da, db := decA.Decode([]byte(a)), decB.Decode([]byte(b))

	utf8Dec, err := codec.NewDecoder(utf8.DecodeRune)
	if err != nil {
		panic(err)
	}

	diff := &Diff{
		utf8Decoder: utf8Dec,
		linesA:      lda.lines,
		linesB:      ldb.lines,
	}

	lineDiffs := lcs.NewMyers(da, db).SES()
	script := lineDiffs
	for len(script) > 0 {
		var edits lcs.EditScript
		edits, script = getEditsForGroup(script)
		diff.groups = append(diff.groups, diff.newGroup(edits))
	}

	/*
		for i, g := range diff.groups {
			a, b := utf8Dec.Decode([]byte(g.textA)), utf8Dec.Decode([]byte(g.textB))
			diff.groups[i].edits = lcs.NewMyers(a, b).SES()
		}

		//	lcs.PrettyVertical(os.Stdout, da, lineDiffs)
		for _, g := range diff.groups {
			//fmt.Printf("EDITS: %v\n", g.edits.String())
			//fmt.Printf("EDITS: %v\n", g.edits[0])

			//	lcs.FormatVertical(os.Stdout, g.decodedA, g.edits)
			fmt.Printf("a%v\n", g.insertions)
			fmt.Printf("d%v\n", g.deletions)
			fmt.Printf("c%v\n", g.changes)
			fmt.Printf("a%v\n", lineRange(g.insertions))
			fmt.Printf("d%v\n", lineRange(g.deletions))
			fmt.Printf("c%v\n", lineRange(g.changes))

			fmt.Printf("%s\n", g.textA)
			fmt.Printf("------------------\n")
			fmt.Printf("%s\n", g.textB)
			//lcs.PrettyHorizontal(os.Stdout, []int32(g.textA), g.edits)
		}
	*/
	return diff
}
