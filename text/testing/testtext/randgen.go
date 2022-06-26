// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package testtext

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

// Random can be used to generate strings containing randomly selected runes.
type Random struct {
	options
	r *rand.Rand
}

type options struct {
	includeControl bool
}

// Option represents an option to the factory methods in this package.
type Option func(o *options)

// IncludeControlOpt controls whether control characters can be included
// in the generated strings.
func IncludeControlOpt(v bool) Option {
	return func(o *options) {
		o.includeControl = v
	}
}

// NewRandom returns a new instance of Random.
func NewRandom(opts ...Option) *Random {
	r := &Random{
		r: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, fn := range opts {
		fn(&r.options)
	}
	return r
}

// tableRange is used to represent script code point ranges of
// different sizes.
type tableRange struct {
	lo, hi, stride int32
}

var tableRanges = []tableRange{}

func init() {
	tableRanges = append(tableRanges,
		tableRange{0, 127, 1},         // ASCII, 1 byte
		tableRange{248, 696, 1},       // Latin, 2 byte
		tableRange{7680, 7935, 1},     // Latin, 3 byte
		tableRange{118784, 119029, 1}, // Common, 4 byte
	)
}

/*
func init() {
	for _, s := range []*unicode.RangeTable{
		unicode.Adlam,
		unicode.Common,
	} {
		fmt.Println(s)
		for _, l16 := range s.R32 {
			fmt.Println(l16, l16.Hi-l16.Lo)
		}
	}
}*/

// RuneLen generates a string of length nRunes that contains only the
// requested number of nBytes (1-4) per rune.
func (r Random) WithRuneLen(nBytes int, nRunes int) string {
	sb := &strings.Builder{}
	for i := 0; i < nRunes; i++ {
		sb.WriteRune(r.genInRange(tableRanges[nBytes-1]))
	}
	return sb.String()
}

func (r Random) genInRange(tr tableRange) rune {
	if tr.stride != 1 {
		panic(fmt.Sprintf("unsupported stride: %v, only stride of 1 supported", tr.stride))
	}
	for {
		c := r.r.Int31n(tr.hi-tr.lo) + tr.lo
		if r.includeControl || !unicode.IsControl(c) {
			return c
		}
	}
}

// uniqueNRand returns a slice nItems long with non-repeating numbers in
// the range [1..nItems).
func uniqueNRand(rnd *rand.Rand, nItems int) []int {
	pattern := []int{}
	exists := func(c int) bool {
		for _, p := range pattern {
			if c == p {
				return true
			}
		}
		return false
	}
	for {
		c := rnd.Intn(nItems)
		if exists(c) {
			continue
		}
		pattern = append(pattern, c)
		if len(pattern) == nItems {
			return pattern
		}
	}
}

// AllRuneLens generates a string of length nRunes that contains runes of
// differing lengths. The lengths used are a randomized but repeating order
// of 1..4.
func (r Random) AllRuneLens(nRunes int) string {
	sb := &strings.Builder{}
	pattern := uniqueNRand(r.r, 4)
	for i := 0; i < nRunes; i++ {
		tr := tableRanges[pattern[i%4]]
		sb.WriteRune(r.genInRange(tr))
	}
	return sb.String()
}
