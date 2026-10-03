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
		jsonmsgs.WithMaxSize(40), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(30))
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
	if got, want := writeObject(t, plain, &buf), "{\"a\":1}"; got != want {
		t.Errorf("default encoder: got %q, want %q", got, want)
	}

	indented := jsonmsgs.NewMessager(nil, &buf,
		jsonmsgs.WithEncoderOptions(jsontext.WithIndent("\t")))
	// Multiple writes exercise encoder reuse from the pool; the options must
	// survive the Reset performed by NewEncoder.
	for i := range 3 {
		if got, want := writeObject(t, indented, &buf), "{\n\t\"a\": 1\n}"; got != want {
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

func TestNewMessagerPanicsOnInvalidMaxSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected NewMessager to panic when maxSize < 2")
		}
	}()
	jsonmsgs.NewMessager(nil, io.Discard, jsonmsgs.WithMaxSize(1))
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
		frameLen := uint32(raw[off]) | uint32(raw[off+1])<<8 | uint32(raw[off+2])<<16 | uint32(raw[off+3])<<24
		off += 4 + int(frameLen)
		chunkCount++
	}
	if chunkCount <= 1 {
		t.Fatalf("expected a fragmented message, got %d chunks", chunkCount)
	}
	if cw.writes != chunkCount {
		t.Errorf("expected exactly one Write call per fragment (%d), got %d Write calls", chunkCount, cw.writes)
	}
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
