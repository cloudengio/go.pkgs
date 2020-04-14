// Package edit provides support for editing in-memory byte slices using
// insert, delete and replace operations.
package edit

import (
	"fmt"
	"sort"

	"cloudeng.io/errors"
)

type editOp int

const (
	insertOp editOp = iota
	deleteOp
	replaceOp
)

// comparisons encodes the sorting order for different operations at the
// same position; the stored value is returned from the sort.SliceStable less
// function. Note that since SliceStable is used, comparisons of the same
// operation type return false to allow for stable sorting; true is returned
// to provide precedence between ops, with the order being: delete, replace,
// insert.
var comparisons = [][]bool{
	// insert, delete, replace for j
	{false, false, false}, // insert for i
	{true, false, true},   // delete for i
	{true, false, false},  // replace for i
}

// Delta represents an insertion, deletion or replacement.
type Delta struct {
	op       editOp
	from, to int
	data     []byte
}

// String implements stringer. The format is as follows:
//   deletions:    < (from, to]
//   insertions:   > @pos#<num bytes>
//   replacements: ~ @pos#<num-bytes>/<num-bytes>
func (d Delta) String() string {
	switch d.op {
	case deleteOp:
		return fmt.Sprintf("< @%d#%d", d.from, d.to-d.from)
	case insertOp:
		return fmt.Sprintf("> @%d#%d", d.from, len(d.data))
	default:
		return fmt.Sprintf("~ @%d#%d/%d", d.from, d.to-d.from, len(d.data))
	}
}

func (d Delta) Text() string {
	return string(d.data)
}

// Insert creates a Delta to insert the supplied bytes at pos.
func Insert(pos int, data []byte) Delta {
	return Delta{
		op:   insertOp,
		from: pos,
		data: data,
	}
}

// InsertString is like Insert but for a string.
func InsertString(pos int, text string) Delta {
	return Insert(pos, []byte(text))
}

// Replace creates a Delta to replace size bytes starting at pos
// with text. The string may be shorter or longer than size.
func Replace(pos, size int, data []byte) Delta {
	return Delta{
		op:   replaceOp,
		from: pos,
		to:   pos + size,
		data: data,
	}
}

// ReplaceString is like Replace but for a string.
func ReplaceString(pos, size int, text string) Delta {
	return Replace(pos, size, []byte(text))
}

// Delete creates a Delta to delete size bytes starting at pos.
func Delete(pos, size int) Delta {
	return Delta{
		op:   deleteOp,
		from: pos,
		to:   pos + size,
	}
}

// Validate determines if the supplied deltas fall within the bounds
// of content.
func Validate(contents []byte, deltas ...Delta) error {
	sort.SliceStable(deltas, func(i, j int) bool {
		// sort by reverse position.
		return deltas[j].from < deltas[i].from
	})
	errs := &errors.M{}
	for _, d := range deltas {
		if d.from > len(contents) {
			errs.Append(fmt.Errorf("out of range: %s", d))
		} else {
			if d.op != insertOp {
				// replace or delete.
				if d.to > len(contents) {
					errs.Append(fmt.Errorf("out of range: %s", d))
				}
				continue
			}
			// insertion, no point looking further since the deltas are sorted
			// by position in decreasing order.
			break
		}
	}
	return errs.Err()
}

func sortDeltas(deltas []Delta) {
	sort.SliceStable(deltas, func(i, j int) bool {
		// Sort by start position.
		if deltas[i].from == deltas[j].from {
			return comparisons[deltas[i].op][deltas[j].op]
		}
		return deltas[i].from < deltas[j].from
	})
}

// overwrite returns a string that represents overwriting a with b.
// If b is shorter than a, then it overwrites that shorter portion of a.
func overwrite(a, b []byte) []byte {
	al, bl := len(a), len(b)
	if bl > al {
		return b
	}
	n := make([]byte, al)
	copy(n, a)
	copy(n[:bl], b)
	return n
}

// Do applies the supplied deltas to contents as follows:
//   1. Deltas are sorted by their start position, then at each position,
//   2. deletions are applied, then
//   3. replacements are applied, then,
//   4. insertions are applied.
// Sorting is stable with respect the order specified in the function invocation.
// Multiple deletions and replacements overwrite each other, whereas insertions
// are concatenated.
// All position values are with respect to the original value of contents.
func Do(contents []byte, deltas ...Delta) []byte {
	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}
	sortDeltas(deltas)

	// Offset is the start index of the unprocessed portion of the
	// original content.
	offset := 0
	// Replacements complicate things, especially when there are multiple
	// ones at the same position. Later replacements overwrite earlier ones.
	// This requires keeping track of runs of replacements.
	replacement := []byte{}
	replaceOffset := 0
	patched := make([]byte, 0, 64*1024)
	prevPos := 0
	for _, d := range deltas {
		if d.from > len(contents) || (d.op != insertOp && d.to > len(contents)) {
			// All operations must start in range, replacements and deletes must
			// end in range.
			break
		}
		if d.op != replaceOp || d.from != prevPos {
			offset = max(offset, replaceOffset)
			patched = append(patched, replacement...)
			replacement = nil
			replaceOffset = 0
		}
		if d.from > offset {
			patched = append(patched, contents[offset:d.from]...)
			offset = max(offset, d.from)
		}
		switch d.op {
		case deleteOp:
			offset = max(offset, d.to)
		case replaceOp:
			replacement = overwrite(replacement, d.data)
			replaceOffset = max(replaceOffset, d.to)
		case insertOp:
			patched = append(patched, d.data...)
		}
		prevPos = d.from
	}
	if len(replacement) > 0 {
		patched = append(patched, replacement...)
		offset = max(offset, replaceOffset)
	}
	if offset < len(contents) {
		patched = append(patched, contents[offset:]...)
	}
	return patched
}

// DoString is like Do but for strings.
func DoString(contents string, deltas ...Delta) string {
	return string(Do([]byte(contents), deltas...))
}
