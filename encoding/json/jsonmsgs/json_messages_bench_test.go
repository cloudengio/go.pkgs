// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"encoding/json/jsontext"
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
	nm := jsonmsgs.NewMessager(&buf, io.NopCloser(bytes.NewReader(nil)))
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
	nm := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(nil)))
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

// BenchmarkMessagerWriteMessageParallel stresses the sync.Pool under
// concurrent access. Note: io.Discard is used as the writer since Messager
// does not serialise concurrent writes — this benchmark isolates pool throughput.
func BenchmarkMessagerWriteMessageParallel(b *testing.B) {
	nm := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(nil)))
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
	nm := jsonmsgs.NewMessager(io.Discard, &loopReader{data: msg})
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
		nm := jsonmsgs.NewMessager(io.Discard, &loopReader{data: msg})
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
