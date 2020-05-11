// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package lcs provides implementations of algorithms to find the
// longest common subsequence/shortest edit script (LCS/SES) between two
// slices suitable for use with unicode/utf8 and other alphabets.
package lcs

// TODO(cnicolaou): improve DP implementation to use only one row+column to
// store lcs lengths rather than row * column.
// TODO(cnicolaou): improve the Myers implementation as described in
// An O(NP) Sequence Comparison Algorithm, Wu, Manber, Myers.

import (
	"fmt"
	"reflect"
)

type comparator func(i, j int) bool
type accessor func(i int) interface{}

func configureAndValidate(a, b interface{}) (na, nb int, err error) {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		err = fmt.Errorf("input types differ: %T != %T", a, b)
		return
	}
	switch ta := a.(type) {
	case []int64:
		b64 := b.([]int64)
		na, nb = len(ta), len(b64)
	case []int32:
		b32 := b.([]int32)
		na, nb = len(ta), len(b32)
	case []uint8:
		b8 := b.([]uint8)
		na, nb = len(ta), len(b8)
	default:
		err = fmt.Errorf("unsupported type: %T", a)
	}
	return
}

func cmpFor(a, b interface{}) comparator {
	switch ta := a.(type) {
	case []int64:
		b64 := b.([]int64)
		return func(i, j int) bool {
			return ta[i] == b64[j]
		}
	case []int32:
		b32 := b.([]int32)
		return func(i, j int) bool {
			return ta[i] == b32[j]
		}
	case []uint8:
		b8 := b.([]uint8)
		return func(i, j int) bool {
			return ta[i] == b8[j]
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", a))
	}
}

func accessorFor(a interface{}) accessor {
	switch ta := a.(type) {
	case []int64:
		return func(i int) interface{} {
			return ta[i]
		}
	case []int32:
		return func(i int) interface{} {
			return ta[i]
		}
	case []uint8:
		return func(i int) interface{} {
			return ta[i]
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", a))
	}
}
