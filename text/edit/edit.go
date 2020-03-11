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
// function. Note that since SliceStable is used, comparions of the same
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
	text     string
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
		return fmt.Sprintf("> @%d#%d", d.from, len(d.text))
	default:
		return fmt.Sprintf("~ @%d#%d/%d", d.from, d.to-d.from, len(d.text))
	}
}

// Insert creates a Delta to insert text at pos.
func Insert(pos uint, text string) Delta {
	return Delta{
		op:   insertOp,
		from: int(pos),
		text: text,
	}
}

// Replace creates a Delta to replace the size bytes starting at pos
// with the specified string. The string may be shorter or longer than size.
func Replace(pos, size uint, text string) Delta {
	return Delta{
		op:   replaceOp,
		from: int(pos),
		to:   int(pos + size),
		text: text,
	}
}

// Delete creates a Delta to delete size bytes starting at pos.
func Delete(pos, size uint) Delta {
	return Delta{
		op:   deleteOp,
		from: int(pos),
		to:   int(pos + size),
	}
}

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
			if d.to != 0 {
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
func overwrite(a, b string) string {
	al, bl := len(a), len(b)
	if bl > al {
		return b
	}
	n := make([]byte, al)
	copy(n, []byte(a))
	copy(n[:bl], []byte(b))
	return string(n)
}

// Do applies the supplied deltas to the supplied contents as follows:
//   1. Deltas are sorted by their start position, then at each position,
//   2. deletions are applied, then
//   3. replacements are applied, then,
//   4. insertions are applied.
// Sorting is stable with respect the order specified in the function invocation.
// Multiple deletions and replacements overwrite each other, whereas insertions
// are concatenated.
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
	// This requires keeping trakc of runs of replacements.
	replacement := ""
	replaceOffset := 0
	patched := make([]byte, 0, 64*1024)
	for _, d := range deltas {
		if d.from > len(contents) || (d.op != insertOp && d.to > len(contents)) {
			// all operations must start in range, replacements and deletes must
			// end in range.
			break
		}
		if d.op != replaceOp {
			offset = max(offset, replaceOffset)
			patched = append(patched, replacement...)
			replacement = ""
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
			replacement = overwrite(replacement, d.text)
			replaceOffset = max(replaceOffset, d.to)
		case insertOp:
			patched = append(patched, d.text...)
		}
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
