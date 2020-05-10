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
		idx = len(data)
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
	nextIns, nextDel := 0, 0
	for i, edit := range edits {
		switch edit.Op {
		case lcs.Identical:
			return group, edits[last+1:]
		case lcs.Insert:
			if firstIns {
				nextIns = edit.A
			}
			if edit.A > nextIns {
				goto done
			}
			nextIns = edit.A + 1
			group = append(group, edit)
			firstIns = false
		case lcs.Delete:
			if firstDel {
				nextDel = edit.A
			}
			if edit.A > nextDel {
				goto done
			}
			nextDel = edit.A + 1
			firstDel = false
			group = append(group, edit)
		}
		last = i
	}
done:
	return group, edits[last+1:]
}

type Group struct {
	edits                       lcs.EditScript
	insertions, deletions       map[int][]lcs.Edit
	insertedLines, deletedLines []int
	insertedText, deletedText   string
}

func (d *Diff) newGroup(edits lcs.EditScript) *Group {
	insertions, deletions := map[int][]lcs.Edit{}, map[int][]lcs.Edit{}
	insertedLines, deletedLines := []int{}, []int{}
	for _, edit := range edits {
		switch edit.Op {
		case lcs.Insert:
			insertions[edit.A] = append(insertions[edit.A], edit)
			insertedLines = append(insertedLines, edit.B)
		case lcs.Delete:
			deletions[edit.A] = append(deletions[edit.A], edit)
			deletedLines = append(deletedLines, edit.A)
		}
	}
	return &Group{
		insertions:    insertions,
		deletions:     deletions,
		insertedLines: insertedLines,
		deletedLines:  deletedLines,
		edits:         edits,
		insertedText:  text(d.linesB, insertedLines),
		deletedText:   text(d.linesA, deletedLines),
	}
}

func (g *Group) Summary() string {
	onlyKey := func(m map[int][]lcs.Edit) int {
		for k := range m {
			return k
		}
		panic("unreachable")
	}
	ins, dels := g.insertions, g.deletions
	ni, nd := len(ins), len(dels)
	switch {
	case ni == 1 && nd == 0:
		l := onlyKey(ins)
		return fmt.Sprintf("%da%s", ins[l][0].A, lineRange(g.insertedLines))
	case nd >= 1 && ni == 0:
		l := g.deletedLines[0]
		return fmt.Sprintf("%sd%v", lineRange(g.deletedLines), dels[l][0].B)
	default:
		return fmt.Sprintf("%sc%s", lineRange(g.deletedLines), lineRange(g.insertedLines))
	}
}

func (g *Group) Inserted() string {
	return g.insertedText
}

func (g *Group) Deleted() string {
	return g.deletedText
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
	return DiffByLinesUsing(a, b, Myers)
}

func Myers(a, b interface{}) lcs.EditScript {
	return lcs.NewMyers(a, b).SES()
}

func DP(a, b interface{}) lcs.EditScript {
	return lcs.NewDP(a, b).SES()
}

func DiffByLinesUsing(a, b []byte, engine func(a, b interface{}) lcs.EditScript) *Diff {
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

	lineDiffs := engine(da, db)
	//	lcs.FormatVertical(os.Stdout, da, lineDiffs)
	script := lineDiffs
	for len(script) > 0 {
		var edits lcs.EditScript
		edits, script = getEditsForGroup(script)
		if len(edits) == 0 {
			continue
		}
		diff.groups = append(diff.groups, diff.newGroup(edits))
	}
	return diff
}
