// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

func setMask(args []interface{}) []int {
	if len(args) == 0 {
		return nil
	}
	set := make([]int, len(args))
	for i, a := range args {
		typ := reflect.TypeOf(a)
		switch typ.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			v := reflect.ValueOf(a)
			if v.Len() > 0 {
				set[i] = 1
			}
		default:
			set[i] = -1
		}
	}
	return set
}

func validate(mask []int, caller string) {
	_, file, line, _ := runtime.Caller(2)
	callsite := fmt.Sprintf("%v:%v", filepath.Base(file), line)
	for i, v := range mask {
		if v < 0 {
			panic(fmt.Sprintf("parameter %v in call to %v from %v is not a slice, map, array or string", i, caller, callsite))
		}
	}
}

func count(mask []int) int {
	c := 0
	for _, v := range mask {
		if v > 0 {
			c++
		}
	}
	return c
}

// ExactlyOneSet will return true if exactly one of its arguments is 'set',
// where 'set' means:
//   1. for strings, the length is > 0.
//   2. fo slices, arrays and maps, their length is > 0.
// ExactlyOneSet will panic if any of the arguments are not one of the above
// types.
func ExactlyOneSet(args ...interface{}) bool {
	mask := setMask(args)
	validate(mask, "flags.ExactlyOneSet")
	return count(mask) == 1
}

// AtMostOneSet is like ExactlyOne except that it returns true if zero
// or one of its arguments are set.
func AtMostOneSet(args ...interface{}) bool {
	mask := setMask(args)
	validate(mask, "flags.ExactlyOneSet")
	return count(mask) == 1 || count(mask) == 0
}

// AllSet is like ExactlyOne except that it returns true if all of its
// arguments are set.
func AllSet(args ...interface{}) bool {
	mask := setMask(args)
	validate(mask, "flags.ExactlyOneSet")
	return count(mask) == len(args)
}
