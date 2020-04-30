// Package lcs provides implementations of alogorithms to find the
// longest common subsequence/shortest edit script (LCS/SES) suitable for
// use with unicode/utf8 and other alphabets.
package lcs

import (
	"fmt"
	"reflect"
)

type comparator func(i, j int) bool
type accessor func(i int) interface{}
type appendor func(slice, value interface{}) interface{}

func configureAndValidate(a, b interface{}) (na, nb int, err error) {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		err = fmt.Errorf("input types differ: %T != %T", a, b)
		return
	}
	switch ta := a.(type) {
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

func newSliceFor(item interface{}) func() interface{} {
	switch item.(type) {
	case []int32:
		return func() interface{} {
			return []int32{}
		}
	case []uint8:
		return func() interface{} {
			return []uint8{}
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", item))
	}
}

func appendorFor(p interface{}) appendor {
	switch p.(type) {
	case []int32:
		return func(s, v interface{}) interface{} {
			return append(s.([]int32), v.(int32))
		}
	case []uint8:
		return func(s, v interface{}) interface{} {
			return append(s.([]uint8), v.(uint8))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", p))
	}
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
