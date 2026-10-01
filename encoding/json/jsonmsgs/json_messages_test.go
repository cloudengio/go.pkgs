// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"

	"encoding/json/jsontext"

	"cloudeng.io/encoding/json/jsonmsgs"
)

func TestMessagerMultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	// Set maxSize to 50 bytes. We will write 3 messages of 30 bytes each (90 bytes total).
	// Under the old io.LimitReader bug, reading the 3rd message would fail with EOF.
	nmWriter := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), &buf, jsonmsgs.WithMaxSize(50))
	for i := range 3 {
		enc := nmWriter.NewEncoder()
		_ = enc.WriteToken(jsontext.BeginObject)
		_ = enc.WriteToken(jsontext.String("msg"))
		_ = enc.WriteToken(jsontext.Int(int64(i)))
		_ = enc.WriteToken(jsontext.EndObject)
		if err := nmWriter.WriteMessage(enc); err != nil {
			t.Fatalf("write msg %d: %v", i, err)
		}
	}

	nmReader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(buf.Bytes())), io.Discard, jsonmsgs.WithMaxSize(50))
	for i := range 3 {
		nmd, err := nmReader.ReadMessage()
		if err != nil {
			t.Fatalf("read msg %d: %v", i, err)
		}
		if err := decodeMsg(nmd.Decoder, i); err != nil {
			t.Errorf("msg %d: %v", i, err)
		}
		nmReader.ReleaseDecoder(nmd)
	}
}

// decodeMsg reads {"msg":<n>} from dec and verifies n == want.
func decodeMsg(dec *jsontext.Decoder, want int) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("read '{': %w", err)
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("expected '{', got %v", tok.Kind())
	}
	key, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("read key: %w", err)
	}
	if key.String() != "msg" {
		return fmt.Errorf("expected key %q, got %q", "msg", key.String())
	}
	val, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("read val: %w", err)
	}
	n, err := val.Int()
	if err != nil {
		return fmt.Errorf("val not int: %w", err)
	}
	if int(n) != want {
		return fmt.Errorf("got %d, want %d", n, want)
	}
	if _, err := dec.ReadToken(); err != nil { // consume '}'
		return fmt.Errorf("read '}': %w", err)
	}
	return nil
}

func TestMessagerMaxSizeEnforced(t *testing.T) {
	// By default, fragmentation is disabled. WriteMessage must fail if size > maxSize.
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), io.Discard,
		jsonmsgs.WithMaxSize(20))
	enc := nm.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("large_field_content_exceeding_twenty_bytes"))
	_ = enc.WriteToken(jsontext.String("more_data_here"))
	_ = enc.WriteToken(jsontext.EndObject)

	if err := nm.WriteMessage(enc); err == nil {
		t.Fatal("expected WriteMessage to fail for size > maxSize when fragmentation is disabled, got nil")
	} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("expected error wrapping ErrMessageTooLarge, got: %v", err)
	}

	// With MaxFragmentedMessageSize set, WriteMessage must fail if total size > maxFragmentedMessageSize.
	nmMaxMsg := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), io.Discard,
		jsonmsgs.WithMaxSize(20), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(30))
	enc2 := nmMaxMsg.NewEncoder()
	_ = enc2.WriteToken(jsontext.BeginObject)
	_ = enc2.WriteToken(jsontext.String("large_field_content_exceeding_twenty_bytes"))
	_ = enc2.WriteToken(jsontext.String("more_data_here"))
	_ = enc2.WriteToken(jsontext.EndObject)
	if err := nmMaxMsg.WriteMessage(enc2); err == nil {
		t.Fatal("expected WriteMessage to fail for size > maxFragmentedMessageSize, got nil")
	} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("expected error wrapping ErrMessageTooLarge, got: %v", err)
	}

	// Craft a header with size 100 > maxSize 20.
	var fakeStream bytes.Buffer
	fakeStream.Write([]byte{100, 0, 0, 0})
	fakeStream.Write(make([]byte, 100))

	nmReader := jsonmsgs.NewMessager(io.NopCloser(&fakeStream), io.Discard, jsonmsgs.WithMaxSize(20))
	if _, err := nmReader.ReadMessage(); err == nil {
		t.Fatal("expected ReadMessage to fail when length > maxSize, got nil")
	} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("expected error wrapping ErrMessageTooLarge, got: %v", err)
	}
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func TestMessagerConcurrentWrites(t *testing.T) {
	var sbuf safeBuffer
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), &sbuf)

	const numGoroutines = 20
	const msgsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gid int) {
			defer wg.Done()
			for m := range msgsPerGoroutine {
				enc := nm.NewEncoder()
				_ = enc.WriteToken(jsontext.BeginObject)
				_ = enc.WriteToken(jsontext.String("g"))
				_ = enc.WriteToken(jsontext.Int(int64(gid)))
				_ = enc.WriteToken(jsontext.String("m"))
				_ = enc.WriteToken(jsontext.Int(int64(m)))
				_ = enc.WriteToken(jsontext.EndObject)
				if err := nm.WriteMessage(enc); err != nil {
					t.Errorf("WriteMessage g=%d m=%d: %v", gid, m, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	// Read all messages back and verify there was no interleaving/corruption.
	reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(sbuf.buf.Bytes())), io.Discard)
	totalMsgs := numGoroutines * msgsPerGoroutine
	for i := range totalMsgs {
		dec, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage msg %d/%d: %v", i, totalMsgs, err)
		}
		if _, err := dec.ReadValue(); err != nil {
			t.Fatalf("decode value %d: %v", i, err)
		}
		reader.ReleaseDecoder(dec)
	}
}

func TestMessagerReleaseDecoderSafety(t *testing.T) {
	var buf bytes.Buffer
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), &buf)

	// Releasing an unpooled decoder should be safely ignored and not corrupt the pool.
	unpooled := jsonmsgs.NewDecoderForTests(jsontext.NewDecoder(strings.NewReader(`{}`)))
	nm.ReleaseDecoder(unpooled)

	// Writing and reading a message should still work normally without panic.
	enc := nm.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.EndObject)
	if err := nm.WriteMessage(enc); err != nil {
		t.Fatal(err)
	}

	reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(buf.Bytes())), io.Discard)
	dec, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	reader.ReleaseDecoder(dec)
}

func TestMessagerCloseNilReader(t *testing.T) {
	nm := jsonmsgs.NewMessager(nil, io.Discard)
	if err := nm.Close(); err != nil {
		t.Errorf("Close on nil reader returned error: %v", err)
	}
}

// frame returns payload prefixed with its 4-byte little-endian length.
func frame(payload string) []byte {
	n := len(payload)
	return append([]byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}, payload...)
}

// writeObject writes {"a":1} as a single message and returns the payload bytes.
func writeObject(t *testing.T, nm *jsonmsgs.Messager, buf *bytes.Buffer) string {
	t.Helper()
	buf.Reset()
	enc := nm.NewEncoder()
	if err := enc.WriteValue(jsontext.Value(`{"a":1}`)); err != nil {
		t.Fatalf("WriteValue: %v", err)
	}
	if err := nm.WriteMessage(enc); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	return string(buf.Bytes()[4:])
}

func TestMessagerEncoderOptions(t *testing.T) {
	var buf bytes.Buffer

	plain := jsonmsgs.NewMessager(nil, &buf)
	if got, want := writeObject(t, plain, &buf), "{\"a\":1}\n"; got != want {
		t.Errorf("default encoder: got %q, want %q", got, want)
	}

	indented := jsonmsgs.NewMessager(nil, &buf,
		jsonmsgs.WithEncoderOptions(jsontext.WithIndent("\t")))
	// Multiple writes exercise encoder reuse from the pool; the options must
	// survive the Reset performed by NewEncoder.
	for i := range 3 {
		if got, want := writeObject(t, indented, &buf), "{\n\t\"a\": 1\n}\n"; got != want {
			t.Errorf("indented encoder, write %d: got %q, want %q", i, got, want)
		}
	}
}

func TestMessagerDecoderOptions(t *testing.T) {
	const dup = `{"a":1,"a":2}`
	var stream []byte
	for range 3 {
		stream = append(stream, frame(dup)...)
	}

	strict := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(stream)), nil)
	dec, err := strict.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if _, err := dec.ReadValue(); err == nil {
		t.Errorf("default decoder: expected duplicate name error")
	}
	strict.ReleaseDecoder(dec)

	lenient := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(stream)), nil,
		jsonmsgs.WithDecoderOptions(jsontext.AllowDuplicateNames(true)))
	// Multiple reads exercise decoder reuse from the pool; the options must
	// survive the Reset performed by ReadMessage.
	for i := range 3 {
		dec, err := lenient.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage %d: %v", i, err)
		}
		val, err := dec.ReadValue()
		if err != nil {
			t.Errorf("lenient decoder, read %d: %v", i, err)
		} else if string(val) != dup {
			t.Errorf("lenient decoder, read %d: got %q, want %q", i, val, dup)
		}
		lenient.ReleaseDecoder(dec)
	}
}

func TestMessagerFragmentation(t *testing.T) {
	var buf bytes.Buffer
	const frameSize = 50
	nmWriter := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), &buf,
		jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))

	// Create a message that is significantly larger than frameSize (~350 bytes).
	enc := nmWriter.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("payload"))
	payloadStr := strings.Repeat("abcdefghij", 30) // 300 bytes
	_ = enc.WriteToken(jsontext.String(payloadStr))
	_ = enc.WriteToken(jsontext.EndObject)

	if err := nmWriter.WriteMessage(enc); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	// Inspect the raw frames in buf:
	raw := buf.Bytes()
	var chunkCount int
	var totalPayload int
	for off := 0; off < len(raw); {
		if off+4 > len(raw) {
			t.Fatalf("truncated frame header at offset %d", off)
		}
		hdr := uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
		isFrag := (hdr & jsonmsgs.FlagFragment) != 0
		hasMore := (hdr & jsonmsgs.FlagMore) != 0
		chunkLen := int(hdr & jsonmsgs.LengthMask)

		if !isFrag {
			t.Errorf("frame %d: expected FlagFragment bit to be set", chunkCount)
		}
		if chunkLen > frameSize {
			t.Errorf("frame %d: chunkLen %d > frameSize %d", chunkCount, chunkLen, frameSize)
		}
		totalPayload += chunkLen
		chunkCount++
		off += 4 + chunkLen

		// If this is the last frame, hasMore must be false; otherwise true.
		if off == len(raw) {
			if hasMore {
				t.Errorf("last frame %d had FlagMore set", chunkCount-1)
			}
		} else {
			if !hasMore {
				t.Errorf("intermediate frame %d had FlagMore cleared", chunkCount-1)
			}
		}
	}

	if chunkCount <= 1 {
		t.Fatalf("expected message to be fragmented into multiple chunks, got %d", chunkCount)
	}

	// Now read it back and verify reassembly!
	nmReader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(buf.Bytes())), io.Discard,
		jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
	dec, err := nmReader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	defer nmReader.ReleaseDecoder(dec)

	val, err := dec.ReadValue()
	if err != nil {
		t.Fatalf("ReadValue: %v", err)
	}
	if !strings.Contains(string(val), payloadStr) {
		t.Errorf("reassembled JSON does not contain expected payload")
	}
}

func TestMessagerFragmentationBoundarySizes(t *testing.T) {
	const frameSize = 30

	for _, multiplier := range []int{1, 2, 5} {
		for _, delta := range []int{-5, -1, 0, 1, 5} {
			targetLen := frameSize*multiplier + delta
			if targetLen <= 0 {
				continue
			}
			t.Run(fmt.Sprintf("size_%d", targetLen), func(t *testing.T) {
				var buf bytes.Buffer
				writer := jsonmsgs.NewMessager(nil, &buf,
					jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))

				rawPayload := strings.Repeat("a", targetLen)
				enc := writer.NewEncoder()
				_ = enc.WriteToken(jsontext.BeginObject)
				_ = enc.WriteToken(jsontext.String("k"))
				_ = enc.WriteToken(jsontext.String(rawPayload))
				_ = enc.WriteToken(jsontext.EndObject)

				if err := writer.WriteMessage(enc); err != nil {
					t.Fatalf("WriteMessage: %v", err)
				}

				// Read back
				reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(buf.Bytes())), io.Discard,
					jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
				dec, err := reader.ReadMessage()
				if err != nil {
					t.Fatalf("ReadMessage: %v", err)
				}
				defer reader.ReleaseDecoder(dec)

				val, err := dec.ReadValue()
				if err != nil {
					t.Fatalf("ReadValue: %v", err)
				}
				if !strings.Contains(string(val), rawPayload) {
					t.Errorf("reassembled JSON missing expected string of len %d", targetLen)
				}
			})
		}
	}
}

func TestMessagerConcurrentFragmentedWrites(t *testing.T) {
	var sbuf safeBuffer
	const frameSize = 50
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), &sbuf,
		jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))

	const numGoroutines = 10
	const msgsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gid int) {
			defer wg.Done()
			for m := range msgsPerGoroutine {
				enc := nm.NewEncoder()
				_ = enc.WriteToken(jsontext.BeginObject)
				_ = enc.WriteToken(jsontext.String("g"))
				_ = enc.WriteToken(jsontext.Int(int64(gid)))
				_ = enc.WriteToken(jsontext.String("m"))
				_ = enc.WriteToken(jsontext.Int(int64(m)))
				_ = enc.WriteToken(jsontext.String("large"))
				_ = enc.WriteToken(jsontext.String(fmt.Sprintf("%0300d", m)))
				_ = enc.WriteToken(jsontext.EndObject)
				if err := nm.WriteMessage(enc); err != nil {
					t.Errorf("WriteMessage g=%d m=%d: %v", gid, m, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(sbuf.buf.Bytes())), io.Discard,
		jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
	totalMsgs := numGoroutines * msgsPerGoroutine
	for i := range totalMsgs {
		dec, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage msg %d/%d: %v", i, totalMsgs, err)
		}
		if _, err := dec.ReadValue(); err != nil {
			t.Fatalf("decode value %d: %v", i, err)
		}
		reader.ReleaseDecoder(dec)
	}
}

func TestMessagerFragmentationErrors(t *testing.T) {
	const frameSize = 20

	t.Run("truncated_mid_fragment", func(t *testing.T) {
		// Frame with FlagFragment | FlagMore | length 15, but only 5 bytes provided.
		hdr := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 15
		var buf bytes.Buffer
		buf.Write([]byte{byte(hdr), byte(hdr >> 8), byte(hdr >> 16), byte(hdr >> 24)})
		buf.Write(make([]byte, 5))

		r := jsonmsgs.NewMessager(io.NopCloser(&buf), io.Discard,
			jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
		if _, err := r.ReadMessage(); err == nil {
			t.Error("expected EOF/error, got nil")
		}
	})

	t.Run("unfragmented_frame_mid_fragment", func(t *testing.T) {
		// Frame 1: FlagFragment | FlagMore | 10
		hdr1 := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 10
		var buf bytes.Buffer
		buf.Write([]byte{byte(hdr1), byte(hdr1 >> 8), byte(hdr1 >> 16), byte(hdr1 >> 24)})
		buf.Write(make([]byte, 10))

		// Frame 2: Missing FlagFragment (unfragmented frame sent mid-fragment)
		hdr2 := uint32(10)
		buf.Write([]byte{byte(hdr2), byte(hdr2 >> 8), byte(hdr2 >> 16), byte(hdr2 >> 24)})
		buf.Write(make([]byte, 10))

		r := jsonmsgs.NewMessager(io.NopCloser(&buf), io.Discard,
			jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
		if _, err := r.ReadMessage(); err == nil {
			t.Error("expected error for unfragmented frame mid-fragment, got nil")
		}
	})

	t.Run("fragment_disabled_on_read", func(t *testing.T) {
		hdr := jsonmsgs.FlagFragment | 10
		rawFrame := append([]byte{byte(hdr), byte(hdr >> 8), byte(hdr >> 16), byte(hdr >> 24)}, make([]byte, 10)...)

		// By default (fragmentation disabled), reading a fragment frame must fail.
		r := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(rawFrame)), io.Discard,
			jsonmsgs.WithMaxSize(frameSize))
		if _, err := r.ReadMessage(); err == nil {
			t.Error("expected error when fragment received with default options, got nil")
		} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("got %v, want ErrMessageTooLarge", err)
		}

		// Explicit WithFragmentation(false)
		r2 := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(rawFrame)), io.Discard,
			jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(false))
		if _, err := r2.ReadMessage(); err == nil {
			t.Error("expected error when fragment received with WithFragmentation(false), got nil")
		} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("got %v, want ErrMessageTooLarge", err)
		}
	})

	t.Run("zero_length_fragment_rejected", func(t *testing.T) {
		// Frame 1: 10 bytes, FlagMore.
		hdr1 := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 10
		var buf bytes.Buffer
		buf.Write([]byte{byte(hdr1), byte(hdr1 >> 8), byte(hdr1 >> 16), byte(hdr1 >> 24)})
		buf.Write(make([]byte, 10))

		// Frame 2: zero-length fragment with FlagMore still set. A peer could
		// repeat this frame forever to stall ReadMessage; it must be rejected
		// as soon as it is seen instead of being accepted and looped on.
		hdr2 := jsonmsgs.FlagFragment | jsonmsgs.FlagMore
		buf.Write([]byte{byte(hdr2), byte(hdr2 >> 8), byte(hdr2 >> 16), byte(hdr2 >> 24)})

		r := jsonmsgs.NewMessager(io.NopCloser(&buf), io.Discard,
			jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))
		if _, err := r.ReadMessage(); err == nil {
			t.Error("expected error for zero-length fragment, got nil")
		} else if !strings.Contains(err.Error(), "zero-length fragment") {
			t.Errorf("expected zero-length fragment error, got: %v", err)
		}
	})

	t.Run("total_reassembled_size_exceeds_max_fragmented_message_size", func(t *testing.T) {
		// Frame 1: 15 bytes, FlagMore
		hdr1 := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 15
		// Frame 2: 15 bytes (total 30 bytes > maxFragmentedMessageSize 25)
		hdr2 := jsonmsgs.FlagFragment | 15

		var buf bytes.Buffer
		buf.Write([]byte{byte(hdr1), byte(hdr1 >> 8), byte(hdr1 >> 16), byte(hdr1 >> 24)})
		buf.Write(make([]byte, 15))
		buf.Write([]byte{byte(hdr2), byte(hdr2 >> 8), byte(hdr2 >> 16), byte(hdr2 >> 24)})
		buf.Write(make([]byte, 15))

		r := jsonmsgs.NewMessager(io.NopCloser(&buf), io.Discard,
			jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithMaxFragmentedMessageSize(25), jsonmsgs.WithFragmentation(true))
		if _, err := r.ReadMessage(); err == nil {
			t.Error("expected error when total size exceeds maxFragmentedMessageSize, got nil")
		} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("got %v, want ErrMessageTooLarge", err)
		}
	})
}

// rawHeader returns the 4-byte little-endian encoding of a frame header.
func rawHeader(hdr uint32) []byte {
	return []byte{byte(hdr), byte(hdr >> 8), byte(hdr >> 16), byte(hdr >> 24)}
}

// TestMessagerFlagMoreWithoutFragment verifies that a header with FlagMore but
// not FlagFragment, which declares continuation fragments for a frame that is
// not itself a fragment, is rejected, whether or not fragmentation is enabled,
// before its payload is read.
func TestMessagerFlagMoreWithoutFragment(t *testing.T) {
	const frameSize = 20
	const payload = `{"a":1}xxx`
	raw := append(rawHeader(jsonmsgs.FlagMore|uint32(len(payload))), payload...)

	for _, fragmentation := range []bool{false, true} {
		t.Run(fmt.Sprintf("fragmentation=%v", fragmentation), func(t *testing.T) {
			rd := bytes.NewReader(raw)
			r := jsonmsgs.NewMessager(io.NopCloser(rd), io.Discard,
				jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(fragmentation))
			dec, err := r.ReadMessage()
			if err == nil {
				t.Fatalf("expected an error for FlagMore without FlagFragment, got a message: %v", dec)
			}
			if !strings.Contains(err.Error(), "FlagMore set without FlagFragment") {
				t.Errorf("unexpected error: %v", err)
			}
			// Not mistaken for a size or fragmentation-disabled error.
			if errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
				t.Errorf("got ErrMessageTooLarge, want an invalid header error: %v", err)
			}
			// Only the header was consumed: the payload was not read as a message.
			if got, want := rd.Len(), len(payload); got != want {
				t.Errorf("%d bytes left unread, want %d: the payload was read", got, want)
			}
		})
	}
}

// TestMessagerFlagMoreWithoutFragmentBeforeSizeCheck verifies that an
// oversized length with the invalid flags is reported as the invalid header it
// is, not as too large.
func TestMessagerFlagMoreWithoutFragmentBeforeSizeCheck(t *testing.T) {
	raw := rawHeader(jsonmsgs.FlagMore | jsonmsgs.LengthMask)
	r := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(raw)), io.Discard, jsonmsgs.WithMaxSize(20))
	_, err := r.ReadMessage()
	if err == nil || !strings.Contains(err.Error(), "FlagMore set without FlagFragment") {
		t.Errorf("got %v, want an invalid header error", err)
	}
}

// TestMessagerValidFlagCombinations verifies that the frames a Messager writes
// itself are unaffected by the header check: unfragmented, and fragmented with
// both flags set on the first frame.
func TestMessagerValidFlagCombinations(t *testing.T) {
	msgs := []string{`{"a":1}`, `{"key":"a value that spans several fragments"}`}
	var wbuf bytes.Buffer
	w := jsonmsgs.NewMessager(nil, &wbuf, jsonmsgs.WithMaxSize(20), jsonmsgs.WithFragmentation(true))
	for _, v := range msgs {
		enc := w.NewEncoder()
		if err := enc.WriteValue(jsontext.Value(v)); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteMessage(enc); err != nil {
			t.Fatal(err)
		}
	}
	r := jsonmsgs.NewMessager(io.NopCloser(&wbuf), io.Discard,
		jsonmsgs.WithMaxSize(20), jsonmsgs.WithFragmentation(true))
	for _, want := range msgs {
		dec, err := r.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		val, err := dec.ReadValue()
		if err != nil {
			t.Fatal(err)
		}
		if got := string(val); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
		r.ReleaseDecoder(dec)
	}
}

func TestMessagerWithFragmentSize(t *testing.T) {
	const maxSize = 1000
	const fragSize = 100

	t.Run("fragmented", func(t *testing.T) {
		var buf bytes.Buffer
		writer := jsonmsgs.NewMessager(nil, &buf,
			jsonmsgs.WithMaxSize(maxSize),
			jsonmsgs.WithFragmentSize(fragSize),
			jsonmsgs.WithFragmentation(true),
		)

		enc := writer.NewEncoder()
		_ = enc.WriteToken(jsontext.BeginObject)
		_ = enc.WriteToken(jsontext.String("k"))
		rawPayload := strings.Repeat("x", 240)
		_ = enc.WriteToken(jsontext.String(rawPayload))
		_ = enc.WriteToken(jsontext.EndObject)

		if err := writer.WriteMessage(enc); err != nil {
			t.Fatalf("WriteMessage: %v", err)
		}

		verifyFragmentFrames(t, buf.Bytes(), fragSize)

		reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(buf.Bytes())), io.Discard,
			jsonmsgs.WithMaxSize(maxSize), jsonmsgs.WithFragmentation(true))
		dec, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		defer reader.ReleaseDecoder(dec)

		val, err := dec.ReadValue()
		if err != nil {
			t.Fatalf("ReadValue: %v", err)
		}
		if !strings.Contains(string(val), rawPayload) {
			t.Errorf("reassembled JSON does not contain original payload")
		}
	})

	t.Run("small_unfragmented", func(t *testing.T) {
		var buf bytes.Buffer
		writer := jsonmsgs.NewMessager(nil, &buf,
			jsonmsgs.WithMaxSize(maxSize),
			jsonmsgs.WithFragmentSize(fragSize),
			jsonmsgs.WithFragmentation(true),
		)

		enc := writer.NewEncoder()
		_ = enc.WriteToken(jsontext.BeginObject)
		_ = enc.WriteToken(jsontext.String("small"))
		_ = enc.WriteToken(jsontext.String("data"))
		_ = enc.WriteToken(jsontext.EndObject)
		if err := writer.WriteMessage(enc); err != nil {
			t.Fatalf("WriteMessage: %v", err)
		}

		raw := buf.Bytes()
		hdr := uint32(raw[0]) | uint32(raw[1])<<8 | uint32(raw[2])<<16 | uint32(raw[3])<<24
		if (hdr & jsonmsgs.FlagFragment) != 0 {
			t.Errorf("small message should not be fragmented, but FlagFragment was set")
		}

		reader := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(raw)), io.Discard,
			jsonmsgs.WithMaxSize(maxSize), jsonmsgs.WithFragmentation(true))
		dec, err := reader.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage small: %v", err)
		}
		defer reader.ReleaseDecoder(dec)
		val, err := dec.ReadValue()
		if err != nil {
			t.Fatalf("ReadValue small: %v", err)
		}
		if !strings.Contains(string(val), "small") {
			t.Errorf("small message read value mismatch: %s", string(val))
		}
	})
}

func verifyFragmentFrames(t *testing.T, raw []byte, fragSize int) {
	t.Helper()
	var chunkLens []int
	for off := 0; off < len(raw); {
		if off+4 > len(raw) {
			t.Fatalf("truncated frame header at offset %d", off)
		}
		hdr := uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
		isFrag := (hdr & jsonmsgs.FlagFragment) != 0
		hasMore := (hdr & jsonmsgs.FlagMore) != 0
		chunkLen := int(hdr & jsonmsgs.LengthMask)

		if !isFrag {
			t.Errorf("expected frame to have FlagFragment set")
		}
		if chunkLen > fragSize {
			t.Errorf("chunkLen %d > fragSize %d", chunkLen, fragSize)
		}
		chunkLens = append(chunkLens, chunkLen)
		off += 4 + chunkLen

		if off == len(raw) {
			if hasMore {
				t.Errorf("final frame had FlagMore set")
			}
		} else {
			if !hasMore {
				t.Errorf("intermediate frame had FlagMore cleared")
			}
		}
	}
	if len(chunkLens) <= 1 {
		t.Fatalf("expected message to be fragmented into multiple chunks, got %d", len(chunkLens))
	}
}

func TestMessagerFragmentSizeOptions(t *testing.T) {
	// 1. Clamping: fragmentSize > maxSize should be clamped to maxSize.
	var buf bytes.Buffer
	writerClamped := jsonmsgs.NewMessager(nil, &buf,
		jsonmsgs.WithMaxSize(100),
		jsonmsgs.WithFragmentSize(500),
		jsonmsgs.WithFragmentation(true),
	)

	enc := writerClamped.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("k"))
	_ = enc.WriteToken(jsontext.String(strings.Repeat("y", 250)))
	_ = enc.WriteToken(jsontext.EndObject)

	if err := writerClamped.WriteMessage(enc); err != nil {
		t.Fatalf("WriteMessage clamped: %v", err)
	}

	raw := buf.Bytes()
	for off := 0; off < len(raw); {
		hdr := uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
		chunkLen := int(hdr & jsonmsgs.LengthMask)
		if chunkLen > 100 {
			t.Errorf("chunkLen %d exceeded maxSize 100", chunkLen)
		}
		off += 4 + chunkLen
	}

	// 2. Defaulting: fragmentSize 0 should default to maxSize.
	buf.Reset()
	writerZeroFrag := jsonmsgs.NewMessager(nil, &buf,
		jsonmsgs.WithMaxSize(80),
		jsonmsgs.WithFragmentSize(0),
		jsonmsgs.WithFragmentation(true),
	)
	enc2 := writerZeroFrag.NewEncoder()
	_ = enc2.WriteToken(jsontext.BeginObject)
	_ = enc2.WriteToken(jsontext.String("k"))
	_ = enc2.WriteToken(jsontext.String(strings.Repeat("z", 150)))
	_ = enc2.WriteToken(jsontext.EndObject)
	if err := writerZeroFrag.WriteMessage(enc2); err != nil {
		t.Fatalf("WriteMessage zero frag: %v", err)
	}
	raw2 := buf.Bytes()
	for off := 0; off < len(raw2); {
		hdr := uint32(raw2[off]) | uint32(raw2[off+1])<<8 | uint32(raw2[off+2])<<16 | uint32(raw2[off+3])<<24
		chunkLen := int(hdr & jsonmsgs.LengthMask)
		if chunkLen > 80 {
			t.Errorf("chunkLen %d exceeded maxSize 80", chunkLen)
		}
		off += 4 + chunkLen
	}

	// 3. With fragmentation disabled (default or WithFragmentation(false)):
	// A message between fragmentSize (50) and maxSize (200) should be written unfragmented.
	buf.Reset()
	writerNoFrag := jsonmsgs.NewMessager(nil, &buf,
		jsonmsgs.WithMaxSize(200),
		jsonmsgs.WithFragmentSize(50),
		jsonmsgs.WithFragmentation(false),
	)
	enc3 := writerNoFrag.NewEncoder()
	_ = enc3.WriteToken(jsontext.BeginObject)
	_ = enc3.WriteToken(jsontext.String("k"))
	_ = enc3.WriteToken(jsontext.String(strings.Repeat("w", 80)))
	_ = enc3.WriteToken(jsontext.EndObject)
	if err := writerNoFrag.WriteMessage(enc3); err != nil {
		t.Fatalf("WriteMessage disabled frag: %v", err)
	}
	raw3 := buf.Bytes()
	hdr3 := uint32(raw3[0]) | uint32(raw3[1])<<8 | uint32(raw3[2])<<16 | uint32(raw3[3])<<24
	if (hdr3 & jsonmsgs.FlagFragment) != 0 {
		t.Errorf("FlagFragment should not be set when fragmentation is disabled")
	}

	// And a message larger than maxSize (200) should fail.
	enc4 := writerNoFrag.NewEncoder()
	_ = enc4.WriteToken(jsontext.BeginObject)
	_ = enc4.WriteToken(jsontext.String("k"))
	_ = enc4.WriteToken(jsontext.String(strings.Repeat("w", 220)))
	_ = enc4.WriteToken(jsontext.EndObject)
	if err := writerNoFrag.WriteMessage(enc4); err == nil {
		t.Fatal("expected WriteMessage to fail for message > maxSize, got nil")
	} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("expected ErrMessageTooLarge, got: %v", err)
	}
}

func TestNewMessagerPanicsOnOversizedMaxSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected NewMessager to panic when maxSize exceeds LengthMask")
		}
	}()
	jsonmsgs.NewMessager(nil, io.Discard, jsonmsgs.WithMaxSize(jsonmsgs.LengthMask+1))
}

// TestMessagerReleaseDecoderFreesLargeBuffer verifies that ReleaseDecoder
// actually drops the large reassembly buffer rather than merely clearing
// one of two aliased references to it.
func TestMessagerReleaseDecoderFreesLargeBuffer(t *testing.T) {
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), io.Discard,
		jsonmsgs.WithMaxSize(1024))

	// cap must exceed both maxSize*4 (4096) and 4MiB to trigger the release.
	big := make([]byte, 5*1024*1024)
	dec := jsonmsgs.NewDecoderForTests(jsontext.NewDecoder(bytes.NewReader(nil)))
	jsonmsgs.SetDecoderBufferForTests(dec, big)

	before := jsonmsgs.DecoderBufPointerForTests(dec)
	nm.ReleaseDecoder(dec)
	after := jsonmsgs.DecoderBufPointerForTests(dec)

	if before == after {
		t.Error("expected ReleaseDecoder to replace dec.buf so the large backing array becomes collectible")
	}
	if got := jsonmsgs.BufferCapForTests(dec); got != 0 {
		t.Errorf("expected dec.buffer to be dropped, got cap %d", got)
	}
}

// countingWriter counts the number of Write calls it receives.
type countingWriter struct {
	w      io.Writer
	writes int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes++
	return c.w.Write(p)
}

// TestMessagerFragmentationSingleWritePerFragment verifies that each
// fragment (including non-first ones) is written with a single Write call
// combining its header and payload, rather than two separate calls.
func TestMessagerFragmentationSingleWritePerFragment(t *testing.T) {
	var buf bytes.Buffer
	cw := &countingWriter{w: &buf}
	const frameSize = 50
	nm := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), cw,
		jsonmsgs.WithMaxSize(frameSize), jsonmsgs.WithFragmentation(true))

	enc := nm.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("payload"))
	_ = enc.WriteToken(jsontext.String(strings.Repeat("abcdefghij", 30)))
	_ = enc.WriteToken(jsontext.EndObject)
	if err := nm.WriteMessage(enc); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	raw := buf.Bytes()
	var chunkCount int
	for off := 0; off < len(raw); {
		hdr := uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
		chunkLen := int(hdr & jsonmsgs.LengthMask)
		off += 4 + chunkLen
		chunkCount++
	}
	if chunkCount <= 1 {
		t.Fatalf("expected a fragmented message, got %d chunks", chunkCount)
	}
	if cw.writes != chunkCount {
		t.Errorf("expected exactly one Write call per fragment (%d), got %d Write calls", chunkCount, cw.writes)
	}
}

func TestMessagerMaxFragmentedMessageSize(t *testing.T) {
	const maxSize = 100
	const fragSize = 50
	const maxFragMsgSize = 120

	t.Run("write_enforced", func(t *testing.T) {
		var buf bytes.Buffer
		writer := jsonmsgs.NewMessager(nil, &buf,
			jsonmsgs.WithMaxSize(maxSize),
			jsonmsgs.WithFragmentSize(fragSize),
			jsonmsgs.WithFragmentation(true),
			jsonmsgs.WithMaxFragmentedMessageSize(maxFragMsgSize),
		)

		// 1. Message <= fragSize is unfragmented and succeeds.
		encSmall := writer.NewEncoder()
		_ = encSmall.WriteToken(jsontext.BeginObject)
		_ = encSmall.WriteToken(jsontext.String("k"))
		_ = encSmall.WriteToken(jsontext.String("short"))
		_ = encSmall.WriteToken(jsontext.EndObject)
		if err := writer.WriteMessage(encSmall); err != nil {
			t.Fatalf("WriteMessage small: %v", err)
		}

		// 2. Message > fragSize but <= maxFragMsgSize is fragmented and succeeds.
		buf.Reset()
		encMed := writer.NewEncoder()
		_ = encMed.WriteToken(jsontext.BeginObject)
		_ = encMed.WriteToken(jsontext.String("k"))
		_ = encMed.WriteToken(jsontext.String(strings.Repeat("a", 80)))
		_ = encMed.WriteToken(jsontext.EndObject)
		if err := writer.WriteMessage(encMed); err != nil {
			t.Fatalf("WriteMessage medium: %v", err)
		}

		// 3. Message > maxFragMsgSize must fail with ErrMessageTooLarge.
		encLarge := writer.NewEncoder()
		_ = encLarge.WriteToken(jsontext.BeginObject)
		_ = encLarge.WriteToken(jsontext.String("k"))
		_ = encLarge.WriteToken(jsontext.String(strings.Repeat("b", 150)))
		_ = encLarge.WriteToken(jsontext.EndObject)
		err := writer.WriteMessage(encLarge)
		if err == nil {
			t.Fatal("expected WriteMessage to fail for message > maxFragMsgSize, got nil")
		}
		if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("got %v, want ErrMessageTooLarge", err)
		}
		if !strings.Contains(err.Error(), "exceeds maximum fragmented message size") {
			t.Errorf("unexpected error text: %v", err)
		}
	})

	t.Run("read_enforced", func(t *testing.T) {
		// Valid fragmented message: 2 frames of 50 bytes (total 100 <= 120).
		hdr1 := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 50
		hdr2 := jsonmsgs.FlagFragment | 50
		var bufValid bytes.Buffer
		bufValid.Write([]byte{byte(hdr1), byte(hdr1 >> 8), byte(hdr1 >> 16), byte(hdr1 >> 24)})
		bufValid.Write(bytes.Repeat([]byte("a"), 50))
		bufValid.Write([]byte{byte(hdr2), byte(hdr2 >> 8), byte(hdr2 >> 16), byte(hdr2 >> 24)})
		bufValid.Write(bytes.Repeat([]byte("a"), 50))

		readerValid := jsonmsgs.NewMessager(io.NopCloser(&bufValid), io.Discard,
			jsonmsgs.WithMaxSize(maxSize),
			jsonmsgs.WithFragmentation(true),
			jsonmsgs.WithMaxFragmentedMessageSize(maxFragMsgSize),
		)
		dec, err := readerValid.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage valid: %v", err)
		}
		readerValid.ReleaseDecoder(dec)

		// Oversized fragmented message: 3 frames of 50 bytes (total 150 > 120).
		hdrMore := jsonmsgs.FlagFragment | jsonmsgs.FlagMore | 50
		hdrLast := jsonmsgs.FlagFragment | 50
		var bufOver bytes.Buffer
		for range 2 {
			bufOver.Write([]byte{byte(hdrMore), byte(hdrMore >> 8), byte(hdrMore >> 16), byte(hdrMore >> 24)})
			bufOver.Write(bytes.Repeat([]byte("b"), 50))
		}
		bufOver.Write([]byte{byte(hdrLast), byte(hdrLast >> 8), byte(hdrLast >> 16), byte(hdrLast >> 24)})
		bufOver.Write(bytes.Repeat([]byte("b"), 50))

		readerOver := jsonmsgs.NewMessager(io.NopCloser(&bufOver), io.Discard,
			jsonmsgs.WithMaxSize(maxSize),
			jsonmsgs.WithFragmentation(true),
			jsonmsgs.WithMaxFragmentedMessageSize(maxFragMsgSize),
		)
		_, errOver := readerOver.ReadMessage()
		if errOver == nil {
			t.Fatal("expected ReadMessage to fail when total reassembled size > maxFragMsgSize, got nil")
		}
		if !errors.Is(errOver, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("got %v, want ErrMessageTooLarge", errOver)
		}
		if !strings.Contains(errOver.Error(), "exceeds maximum fragmented message size") {
			t.Errorf("unexpected error text: %v", errOver)
		}
	})
}

func TestMessagerOversizedBufferDoesNotTruncate(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("skipping 64-bit test on 32-bit platform")
	}
	for _, frag := range []bool{false, true} {
		t.Run(fmt.Sprintf("frag_%v", frag), func(t *testing.T) {
			nm := jsonmsgs.NewMessager(nil, io.Discard,
				jsonmsgs.WithMaxSize(100),
				jsonmsgs.WithFragmentation(frag),
			)
			enc := nm.NewEncoder()
			// Set size to 4 GiB + 10 bytes (wraps to 10 if truncated to uint32).
			oversized := uint64(1<<32) + 10
			jsonmsgs.SetOversizedEncoderBufferForTests(enc, oversized)

			err := nm.WriteMessage(enc)
			if err == nil {
				t.Fatal("expected WriteMessage to fail for buffer > 4 GiB, got nil")
			}
			if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
				t.Errorf("got %v, want ErrMessageTooLarge", err)
			}
		})
	}
}
