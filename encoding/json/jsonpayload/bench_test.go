// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

// The benchmarks below cover every encode and decode path. They report
// allocations because the readers and writers deliberately work with the
// token stream rather than buffering the payload into a Wire value; a rise in
// allocations is the signal that this has stopped being true.

// encodedPayload returns a message for use by the decode benchmarks.
func encodedPayload(b *testing.B) []byte {
	b.Helper()
	buf, err := json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
	if err != nil {
		b.Fatal(err)
	}
	return buf
}

// BenchmarkWriter measures encoding when the type is known at compile time.
func BenchmarkWriter(b *testing.B) {
	b.ReportAllocs()
	val := &payload{A: 42}
	for b.Loop() {
		if _, err := json.Marshal(jsonpayload.NewWriter(val)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWriterAny measures the same encoding when the type is taken from
// the value rather than from a type parameter, which requires a lookup of the
// dynamic type's name.
func BenchmarkWriterAny(b *testing.B) {
	b.ReportAllocs()
	val := &payload{A: 42}
	for b.Loop() {
		if _, err := json.Marshal(jsonpayload.NewWriterAny(val)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode measures the standalone decode path, which involves no
// wrapper and no registry. The decoder is reset rather than reallocated each
// iteration, so that this measures decoding rather than the construction of a
// decoder, which json.Unmarshal does not pay for since it pools them.
func BenchmarkDecode(b *testing.B) {
	b.ReportAllocs()
	buf := encodedPayload(b)
	var into payload
	rd := bytes.NewReader(buf)
	dec := jsontext.NewDecoder(rd)
	for b.Loop() {
		rd.Reset(buf)
		dec.Reset(rd)
		if err := jsonpayload.Decode(dec, &into); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecodeNewDecoder measures the same decode with a freshly
// constructed decoder, which is what a caller decoding a single message pays.
func BenchmarkDecodeNewDecoder(b *testing.B) {
	b.ReportAllocs()
	buf := encodedPayload(b)
	var into payload
	for b.Loop() {
		if err := jsonpayload.Decode(jsontext.NewDecoder(bytes.NewReader(buf)), &into); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReader measures the same decode reached through json.Unmarshal, so
// the difference from BenchmarkDecode is the cost of the Reader wrapper.
func BenchmarkReader(b *testing.B) {
	b.ReportAllocs()
	buf := encodedPayload(b)
	var into payload
	rd := jsonpayload.NewReader(&into)
	for b.Loop() {
		if err := json.Unmarshal(buf, &rd); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReaderAny measures decoding when the type is not known until the
// message is read, which adds a registry lookup and allocates the value.
func BenchmarkReaderAny(b *testing.B) {
	b.ReportAllocs()
	buf := encodedPayload(b)
	for b.Loop() {
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal(buf, &rd); err != nil {
			b.Fatal(err)
		}
	}
}
