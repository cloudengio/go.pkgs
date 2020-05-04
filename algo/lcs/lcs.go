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

func sliceLen(a interface{}) int {
	v := reflect.ValueOf(a)
	if v.Type().Kind() == reflect.Slice {
		return v.Len()
	}
	panic(fmt.Sprintf("unsupported type: %T", a))
}

/*
func emptySliceOf(a interface{}) interface{} {
	t := reflect.TypeOf(a)
	return reflect.New(t).Elem().Interface()
}*/
