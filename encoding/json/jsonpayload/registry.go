// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload

import (
	"cloudeng.io/types"
)

var (
	decodeRegistry types.Registry
)

// RegisterType registers T under the name reported by TypeName[T], replacing
// any type previously registered under that name. It is used by ReaderAny
// to construct a new instance of the type when decoding a message.
func RegisterType[T any, PT *T]() {
	decodeRegistry.RegisterType[T]()
}

func newInstance(typeName string) (any, bool) {
	return decodeRegistry.New(typeName)
}
