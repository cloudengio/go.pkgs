// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap

import (
	"bytes"
	"encoding/json"
	"iter"
	"math"
	"strconv"
)

// gemini 2.5 wrote some this code, the exceptions being the unsafe and iterator
// methods, as well as the JSON marshaling/unmarshaling. It didn't understand
// iterators at all, nor that JSON doesn't support uint64 directly, hence
// the conversion to/from strings.

// T is a bitmap type that represents a set of bits using a slice of uint64.
type T []uint64

// New creates a new bitmap of the specified size in bits. The size must be
// greater than zero. The bitmap is represented as a slice of uint64. The
// caller must keep track of size if it cares that the size of the bitmap
// is rounded up to the nearest multiple of 64 bits.
func New(size int) T {
	if size <= 0 {
		return nil
	}
	return make(T, (size+63)/64)
}

// Set sets the bit at index i in the bitmap to 1. If i is out of bounds,
// the function does nothing.
func (b T) Set(i int) {
	if i < 0 || i >= len(b)*64 {
		return
	}
	b.SetUnsafe(i)
}

// SetUnsafe sets the bit at index i in the bitmap to 1 without bounds checking.
func (b T) SetUnsafe(i int) {
	b[i/64] |= 1 << (i % 64)
}

// Clear clears the bit at index i in the bitmap, setting it to 0. If i is out of
// bounds, the function does nothing.
func (b T) Clear(i int) {
	if i < 0 || i >= len(b)*64 {
		return
	}
	b.ClearUnsafe(i)
}

// ClearUnsafe clears the bit at index i in the bitmap without bounds checking.
func (b T) ClearUnsafe(i int) {
	b[i/64] &^= 1 << (i % 64)
}

// IsSet checks if the bit at index i in the bitmap is set (1). If i is out of
// bounds, it returns false.
func (b T) IsSet(i int) bool {
	if i < 0 || i >= len(b)*64 {
		return false
	}
	return b.IsSetUnsafe(i)
}

// IsSetUnsafe checks if the bit at index i in the bitmap is set (1) without
// bounds checking.
func (b T) IsSetUnsafe(i int) bool {
	return (b[i/64] & (1 << (i % 64))) != 0
}

// NextSet returns an iterator over all set bits in the bitmap starting from
// the specified index and never exceeding the specified size or size of the
// bitmap itself.
func (b T) NextSet(start, size int) iter.Seq[int] {
	return func(yield func(int) bool) {
		last := min(len(b)*64, size)
		if start < 0 || start >= last {
			return
		}
		for nb := start; nb < last; {
			if nb%64 == 0 && b[nb/64] == 0 {
				nb += 64
				continue
			}
			if b[nb/64]&(1<<(nb%64)) != 0 {
				if !yield(nb) {
					return
				}
			}
			nb++
		}

	}
}

// NextClear returns an iterator over all clear bits in the bitmap starting from
// the specified index and never exceeding the specified size or size of the
// bitmap itself.
func (b T) NextClear(start, size int) iter.Seq[int] {
	return func(yield func(int) bool) {
		last := min(len(b)*64, size)
		if start < 0 || start >= last {
			return
		}
		for nb := start; nb < last; {
			if nb%64 == 0 && b[nb/64] == math.MaxUint64 {
				nb += 64
				continue
			}
			if b[nb/64]&(1<<(nb%64)) == 0 {
				if !yield(nb) {
					return
				}
			}
			nb++
		}
	}
}

func (b T) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, len(b)*12) // Estimate size for JSON encoding.
	wr := bytes.NewBuffer(buf)
	wr.WriteString("[")
	for i, v := range b {
		str := strconv.FormatUint(v, 16)
		wr.WriteRune('"') // Start of string.
		if _, err := wr.WriteString(str); err != nil {
			return nil, err
		}
		wr.WriteRune('"') // End of string.
		if i < len(b)-1 {
			// Write a comma after each value except the last.
			wr.WriteRune(',')
		}

	}
	wr.WriteString("]")
	return wr.Bytes(), nil
}

func (b *T) UnmarshalJSON(data []byte) error {
	var vals []string
	if err := json.Unmarshal(data, &vals); err != nil {
		return err
	}
	if len(vals) == 0 {
		return nil // Empty bitmap.
	}
	*b = make(T, len(vals))
	for i, v := range vals {
		num, err := strconv.ParseUint(v, 16, 64)
		if err != nil {
			return err
		}
		(*b)[i] = num
	}
	return nil
}
