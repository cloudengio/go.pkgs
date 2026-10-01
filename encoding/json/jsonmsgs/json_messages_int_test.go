// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/json/jsontext"
	"unsafe"
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

// SetOversizedEncoderBufferForTests overrides enc's internal buffer with an
// oversized slice header without physically allocating the memory.
func SetOversizedEncoderBufferForTests(enc *Encoder, size uint64) {
	backing := make([]byte, 16)
	buf := new(bytes.Buffer)
	hdr := (*struct {
		data unsafe.Pointer
		len  int
		cap  int
	})(unsafe.Pointer(buf))
	hdr.data = unsafe.Pointer(&backing[0])
	hdr.len = int(size + 4)
	hdr.cap = int(size + 4)
	enc.buffer = buf
}
