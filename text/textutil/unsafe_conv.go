// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"reflect"
	"unsafe"
)

// StringToBytes returns the byte slice containing the data for the
// supplied string without any allocations or copies. It should only be
// used when the resulting byte slice will never be modified.
// See https://groups.google.com/g/golang-nuts/c/Zsfk-VMd_fU/m/O1ru4fO-BgAJ
func StringToBytes(s string) []byte {
	const max = 0x7fff0000
	if len(s) > max {
		panic("string too long")
	}
	return (*[max]byte)(unsafe.Pointer((*reflect.StringHeader)(unsafe.Pointer(&s)).Data))[:len(s):len(s)]
}

// BytesToString returns a string with the supplied byte slice as its contents.
// The original byte slice must never be modified.
// Taken from strings.Builder.String().
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
