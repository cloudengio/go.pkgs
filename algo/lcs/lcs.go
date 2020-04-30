// Package lcs provides implementations of alogorithms to find the
// longest common subsequence/shortest edit script (LCS/SES) suitable for
// use with unicode/utf8 and other alphabets.
package lcs

import (
	"fmt"
	"reflect"
)

func configure(a, b interface{}) (na, nb int, cmp comparator, err error) {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		err = fmt.Errorf("input types differ: %T != %T", a, b)
		return
	}
	switch ta := a.(type) {
	case []int32:
		na, nb = len(ta), len(b.([]int32))
		cmp = compare32
	case []uint8:
		na, nb = len(ta), len(b.([]uint8))
		cmp = compare8
	default:
		err = fmt.Errorf("unsupported type: %T", a)
	}
	return
}

func compare32(a, b interface{}, i, j int) bool {
	return a.([]int32)[i] == b.([]int32)[j]
}

func compare8(a, b interface{}, i, j int) bool {
	return a.([]uint8)[i] == b.([]uint8)[j]
}

func Path(a interface{}, indices []int) interface{} {
	switch v := a.(type) {
	case []int32:
		out := make([]int32, len(indices))
		for i, idx := range indices {
			out[i] = v[idx]
		}
		return out
	case []uint8:
		out := make([]uint8, len(indices))
		for i, idx := range indices {
			out[i] = v[idx]
		}
		return out
	}
	panic(fmt.Errorf("unsupported type: %T", a))
}
