// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

// loopReader replays data indefinitely, simulating a continuous stream of
// identical framed messages. Not safe for concurrent use; each goroutine
// should have its own instance.
type loopReader struct {
	data []byte
	off  int
}

func (r *loopReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		r.off = 0
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

func (r *loopReader) Close() error { return nil }

// encodeFramedMsg encodes a small JSON object as a framed message and
// returns the raw bytes (4-byte LE length prefix + JSON).
func encodeFramedMsg(b *testing.B) []byte {
	b.Helper()
	var buf bytes.Buffer
	nm := jsonmsgs.NewWriter(&buf)
	enc := nm.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("version"))
	_ = enc.WriteToken(jsontext.Uint(0))
	_ = enc.WriteToken(jsontext.String("payload"))
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("x"))
	_ = enc.WriteToken(jsontext.Int(42))
	_ = enc.WriteToken(jsontext.String("y"))
	_ = enc.WriteToken(jsontext.Int(7))
	_ = enc.WriteToken(jsontext.EndObject)
	_ = enc.WriteToken(jsontext.EndObject)
	if err := nm.WriteMessage(enc); err != nil {
		b.Fatal(err)
	}
	return buf.Bytes()
}

// drainDecoder reads all tokens from nmd to exercise the decoder state machine
// and populate namespace slices so Decoder.Reset can reuse them next iteration.
func drainDecoder(b *testing.B, nmd *jsonmsgs.Decoder) {
	b.Helper()
	for {
		if _, err := nmd.ReadToken(); err != nil {
			return
		}
	}
}

func BenchmarkMessagerWriteMessage(b *testing.B) {
	nm := jsonmsgs.NewWriter(io.Discard)
	b.ResetTimer()
	for b.Loop() {
		enc := nm.NewEncoder()
		_ = enc.WriteToken(jsontext.BeginObject)
		_ = enc.WriteToken(jsontext.String("x"))
		_ = enc.WriteToken(jsontext.Int(42))
		_ = enc.WriteToken(jsontext.String("y"))
		_ = enc.WriteToken(jsontext.Int(7))
		_ = enc.WriteToken(jsontext.EndObject)
		if err := nm.WriteMessage(enc); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMessagerWriteMessageParallel stresses the sync.Pool and mutex serialization
// under concurrent access. Note: io.Discard is used as the writer to isolate CPU and pool
// throughput without I/O blocking.
func BenchmarkMessagerWriteMessageParallel(b *testing.B) {
	nm := jsonmsgs.NewWriter(io.Discard)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			enc := nm.NewEncoder()
			_ = enc.WriteToken(jsontext.BeginObject)
			_ = enc.WriteToken(jsontext.String("x"))
			_ = enc.WriteToken(jsontext.Int(42))
			_ = enc.WriteToken(jsontext.EndObject)
			if err := nm.WriteMessage(enc); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMessagerReadMessage measures the ReadMessage path including
// decoder pool Get/Reset/Put and full token decoding. Draining tokens populates
// the jsontext namespace slices so Decoder.Reset can reuse them on the next
// iteration rather than reallocating.
func BenchmarkMessagerReadMessage(b *testing.B) {
	msg := encodeFramedMsg(b)
	nm := jsonmsgs.NewReader(&loopReader{data: msg})
	b.ResetTimer()
	for b.Loop() {
		nmd, err := nm.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		drainDecoder(b, nmd)
		nm.ReleaseDecoder(nmd)
	}
}

// BenchmarkMessagerReadMessageParallel measures per-goroutine read
// throughput. Each goroutine owns its Messager and decoder pool to
// avoid cross-goroutine pool contention; the shared msg slice is read-only.
func BenchmarkMessagerReadMessageParallel(b *testing.B) {
	msg := encodeFramedMsg(b)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		nm := jsonmsgs.NewReader(&loopReader{data: msg})
		for pb.Next() {
			nmd, err := nm.ReadMessage()
			if err != nil {
				b.Fatal(err)
			}
			drainDecoder(b, nmd)
			nm.ReleaseDecoder(nmd)
		}
	})
}

// Large message benchmarks. Each compares sending a message of the same size
// whole, as one bare frame, with fragmenting it into frames of
// DefaultMaxNativeMessageSize, for three kinds of content that differ in how
// much of their escaped form is escape sequences, and so in the cost of
// fragmenting them:
//
//	text:    1 MB of a string with nothing to escape.
//	keys:    1 MB of JSON consisting of short string keys and values, about
//	         23% of whose bytes are the quotes that must be escaped.
//	numeric: 1 MB of an array of numbers, nothing to escape.
//
// SetBytes is the size of the message, so that the throughputs are comparable.
const benchLargeSize = 1_000_000

func largeMessage(kind string) []byte {
	var b bytes.Buffer
	switch kind {
	case "text":
		b.WriteString(`{"t":"`)
		for b.Len() < benchLargeSize {
			b.WriteString("the quick brown fox jumps over the lazy dog ")
		}
		b.WriteString(`"}`)
	case "keys":
		b.WriteByte('{')
		for i := 0; b.Len() < benchLargeSize; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, `"k%d":"v%d"`, i, i)
		}
		b.WriteByte('}')
	default:
		b.WriteByte('[')
		for i := 0; b.Len() < benchLargeSize; i++ {
			if i > 0 {
				b.WriteByte(',')
			}
			fmt.Fprintf(&b, "%d", 1000000+i*7)
		}
		b.WriteByte(']')
	}
	return b.Bytes()
}

var largeKinds = []string{"text", "keys", "numeric"}

// largeOptions returns options for sending a message of up to benchLargeSize
// bytes whole, if whole is true, or fragmented into frames of the default size.
func largeOptions(whole bool) []jsonmsgs.Option {
	if whole {
		return []jsonmsgs.Option{jsonmsgs.WithMaxSize(2 * benchLargeSize)}
	}
	return []jsonmsgs.Option{jsonmsgs.WithFragmentation(true)}
}

func BenchmarkMessagerWriteLarge(b *testing.B) {
	for _, kind := range largeKinds {
		msg := largeMessage(kind)
		for _, whole := range []bool{true, false} {
			name := kind + "/fragmented"
			if whole {
				name = kind + "/whole"
			}
			b.Run(name, func(b *testing.B) {
				nm := jsonmsgs.NewWriter(io.Discard, largeOptions(whole)...)
				b.SetBytes(int64(len(msg)))
				b.ReportAllocs()
				for b.Loop() {
					if err := jsonmsgs.WriteRawForTests(nm, msg); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkMessagerReadLarge(b *testing.B) {
	for _, kind := range largeKinds {
		msg := largeMessage(kind)
		for _, whole := range []bool{true, false} {
			var stream bytes.Buffer
			opts := largeOptions(whole)
			if err := jsonmsgs.WriteRawForTests(jsonmsgs.NewWriter(&stream, opts...), msg); err != nil {
				b.Fatal(err)
			}
			name := kind + "/fragmented"
			if whole {
				name = kind + "/whole"
			}
			b.Run(name, func(b *testing.B) {
				nm := jsonmsgs.NewReader(&loopReader{data: stream.Bytes()}, opts...)
				b.SetBytes(int64(len(msg)))
				b.ReportAllocs()
				for b.Loop() {
					dec, err := nm.ReadMessage()
					if err != nil {
						b.Fatal(err)
					}
					nm.ReleaseDecoder(dec)
				}
			})
		}
	}
}

// headerBenchOptions are the options for the benchmarks that use user headers.
func headerBenchOptions(repeat bool) []jsonmsgs.Option {
	return []jsonmsgs.Option{
		jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(64), jsonmsgs.WithRepeatHeader(repeat),
	}
}

const benchHeader = `{"to":7,"op":"x"}`

// BenchmarkMessagerWriteMessageWithHeader is BenchmarkMessagerWriteMessage for
// a message with a user header, which is sent as a single fragment.
func BenchmarkMessagerWriteMessageWithHeader(b *testing.B) {
	nm := jsonmsgs.NewWriter(io.Discard, headerBenchOptions(false)...)
	b.ReportAllocs()
	for b.Loop() {
		enc := nm.NewEncoder()
		_ = enc.WriteToken(jsontext.BeginObject)
		_ = enc.WriteToken(jsontext.String("x"))
		_ = enc.WriteToken(jsontext.Int(42))
		_ = enc.WriteToken(jsontext.String("y"))
		_ = enc.WriteToken(jsontext.Int(7))
		_ = enc.WriteToken(jsontext.EndObject)
		if err := nm.WriteMessageWithHeader(enc, jsontext.Value(benchHeader)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMessagerReadMessageWithHeader is BenchmarkMessagerReadMessage for a
// message with a user header.
func BenchmarkMessagerReadMessageWithHeader(b *testing.B) {
	var stream bytes.Buffer
	opts := headerBenchOptions(false)
	wr := jsonmsgs.NewWriter(&stream, opts...)
	enc := wr.NewEncoder()
	_ = enc.WriteValue(jsontext.Value(`{"version":0,"payload":{"x":42,"y":7}}`))
	if err := wr.WriteMessageWithHeader(enc, jsontext.Value(benchHeader)); err != nil {
		b.Fatal(err)
	}
	nm := jsonmsgs.NewReader(&loopReader{data: stream.Bytes()}, opts...)
	b.ReportAllocs()
	for b.Loop() {
		nmd, err := nm.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		drainDecoder(b, nmd)
		nm.ReleaseDecoder(nmd)
	}
}

// BenchmarkMessagerForwardLarge measures forwarding, with ReadFragment and
// WriteFragment, a message of 1 MB that is in frames of 64 KiB, which is to
// say the cost of a router that neither reassembles nor decodes. The
// reassembling alternative, for comparison, is the ReadLarge benchmark.
func BenchmarkMessagerForwardLarge(b *testing.B) {
	for _, kind := range largeKinds {
		msg := largeMessage(kind)
		for _, repeat := range []bool{false, true} {
			name := kind + "/header"
			if repeat {
				name = kind + "/repeated-header"
			}
			b.Run(name, func(b *testing.B) {
				forwardBench(b, msg, append(headerBenchOptions(repeat), jsonmsgs.WithMaxSize(64<<10)))
			})
		}
	}
}

func forwardBench(b *testing.B, msg []byte, opts []jsonmsgs.Option) {
	var stream bytes.Buffer
	wr := jsonmsgs.NewWriter(&stream, opts...)
	enc := wr.NewEncoder()
	_ = jsonmsgs.WriteRawIntoEncoderForTests(enc, msg)
	if err := wr.WriteMessageWithHeader(enc, jsontext.Value(benchHeader)); err != nil {
		b.Fatal(err)
	}
	rd := jsonmsgs.NewReader(&loopReader{data: stream.Bytes()}, opts...)
	out := jsonmsgs.NewWriter(io.Discard, opts...)
	b.SetBytes(int64(len(msg)))
	b.ReportAllocs()
	for b.Loop() {
		for last := false; !last; {
			f, err := rd.ReadFragment()
			if err != nil {
				b.Fatal(err)
			}
			if err := out.WriteFragment(f); err != nil {
				b.Fatal(err)
			}
			last = f.Last
		}
	}
}

// BenchmarkMessagerForwardMax is BenchmarkMessagerForwardLarge for the largest
// message that the default options allow, DefaultMaxFragmentedMessageSize less
// room for the header, which is in frames of the default size,
// DefaultMaxNativeMessageSize. The content, a repetition of that used above,
// need not be valid JSON for it to be forwarded.
func BenchmarkMessagerForwardMax(b *testing.B) {
	const size = jsonmsgs.DefaultMaxFragmentedMessageSize - 100
	for _, kind := range largeKinds {
		var msg []byte
		for unit := largeMessage(kind); len(msg) < size; {
			msg = append(msg, unit...)
		}
		msg = msg[:size]
		b.Run(kind, func(b *testing.B) { forwardBench(b, msg, headerBenchOptions(false)) })
	}
}

// modeBenchmarks are the ways of sending the same message, which is of about
// 100 KB, compared by BenchmarkMessagerWriteModes and BenchmarkMessagerReadModes:
// as a bare frame, in a single envelope with a header, and, in frames of 16 KiB,
// fragmented without a header, with one on the first frame and with one on every
// frame.
var modeBenchmarks = []struct {
	name   string
	header bool
	opts   []jsonmsgs.Option
}{
	{"bare", false, []jsonmsgs.Option{jsonmsgs.WithMaxSize(1 << 20)}},
	{"header", true, []jsonmsgs.Option{jsonmsgs.WithMaxSize(1 << 20), jsonmsgs.WithMaxHeaderSize(64)}},
	{"fragmented", false, []jsonmsgs.Option{jsonmsgs.WithMaxSize(16 << 10), jsonmsgs.WithFragmentation(true)}},
	{"fragmented-header", true, []jsonmsgs.Option{jsonmsgs.WithMaxSize(16 << 10), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(64)}},
	{"fragmented-repeated-header", true, []jsonmsgs.Option{jsonmsgs.WithMaxSize(16 << 10), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(64), jsonmsgs.WithRepeatHeader(true)}},
}

func modeMessage() []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i := 0; b.Len() < 100_000; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"k%d":"v%d"`, i, i)
	}
	b.WriteByte('}')
	return b.Bytes()
}

func writeMode(wr *jsonmsgs.Writer, msg []byte, header bool) error {
	enc := wr.NewEncoder()
	_ = jsonmsgs.WriteRawIntoEncoderForTests(enc, msg)
	if header {
		return wr.WriteMessageWithHeader(enc, jsontext.Value(benchHeader))
	}
	return wr.WriteMessage(enc)
}

// BenchmarkMessagerWriteModes compares writing the same message bare, with a
// header and fragmented, see modeBenchmarks.
func BenchmarkMessagerWriteModes(b *testing.B) {
	msg := modeMessage()
	for _, m := range modeBenchmarks {
		b.Run(m.name, func(b *testing.B) {
			wr := jsonmsgs.NewWriter(io.Discard, m.opts...)
			b.SetBytes(int64(len(msg)))
			b.ReportAllocs()
			for b.Loop() {
				if err := writeMode(wr, msg, m.header); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMessagerReadModes compares reading the same message bare, with a
// header and fragmented, see modeBenchmarks.
func BenchmarkMessagerReadModes(b *testing.B) {
	msg := modeMessage()
	for _, m := range modeBenchmarks {
		var stream bytes.Buffer
		if err := writeMode(jsonmsgs.NewWriter(&stream, m.opts...), msg, m.header); err != nil {
			b.Fatal(err)
		}
		b.Run(m.name, func(b *testing.B) {
			rd := jsonmsgs.NewReader(&loopReader{data: stream.Bytes()}, m.opts...)
			b.SetBytes(int64(len(msg)))
			b.ReportAllocs()
			for b.Loop() {
				dec, err := rd.ReadMessage()
				if err != nil {
					b.Fatal(err)
				}
				rd.ReleaseDecoder(dec)
			}
		})
	}
}
