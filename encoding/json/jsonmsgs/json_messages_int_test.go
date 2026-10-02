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

// WriteRawForTests writes raw, which need not be valid JSON, as a message so
// that inputs that jsontext.Encoder cannot produce, such as control
// characters outside of strings or invalid UTF-8, can be exercised. A newline
// is appended to mimic the encoder, WriteMessage trims it.
func WriteRawForTests(m *Writer, raw []byte) error {
	enc := m.NewEncoder()
	enc.buffer.Write(raw)
	enc.buffer.WriteByte('\n')
	return m.WriteMessage(enc)
}

// DecoderBytesForTests returns the message held by dec.
func DecoderBytesForTests(dec *Decoder) []byte {
	return dec.buffer
}

// MaxEscapedLenForTests is the length of the longest escape sequence.
const MaxEscapedLenForTests = maxEscapedLen

// WriteRawIntoEncoderForTests puts raw into enc in the way that
// WriteRawForTests does, without writing it.
func WriteRawIntoEncoderForTests(enc *Encoder, raw []byte) error {
	enc.buffer.Write(raw)
	enc.buffer.WriteByte('\n')
	return nil
}
