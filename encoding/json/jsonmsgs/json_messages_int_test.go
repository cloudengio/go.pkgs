// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/json/jsontext"
)

func NewDecoderForTests(dec *jsontext.Decoder) *Decoder {
	return &Decoder{Decoder: dec}
}

// BufferCapForTests returns the capacity of dec's internal reassembly buffer.
func BufferCapForTests(dec *Decoder) int {
	return cap(dec.buffer)
}

// SetDecoderBufferForTests mimics the aliasing that ReadMessage performs on a
// successful read: dec.buffer and dec.buf end up referencing the same
// backing array via bytes.NewBuffer (which does not copy its argument).
func SetDecoderBufferForTests(dec *Decoder, data []byte) {
	dec.buffer = data
	dec.buf = bytes.NewBuffer(data)
}

// DecoderBufPointerForTests exposes dec.buf for identity comparisons in tests.
func DecoderBufPointerForTests(dec *Decoder) *bytes.Buffer {
	return dec.buf
}
