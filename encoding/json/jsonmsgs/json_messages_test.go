// Copyright 2026 On Your Behalf Inc. All rights reserved.

package jsonmsgs_test

import (
	"bytes"
	"fmt"
	"io"
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
	nmWriter := jsonmsgs.NewMessager(&buf, io.NopCloser(bytes.NewReader(nil)), jsonmsgs.WithMaxSize(50))
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

	nmReader := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(buf.Bytes())), jsonmsgs.WithMaxSize(50))
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
	nm := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(nil)), jsonmsgs.WithMaxSize(20))
	enc := nm.NewEncoder()
	_ = enc.WriteToken(jsontext.BeginObject)
	_ = enc.WriteToken(jsontext.String("large_field_content_exceeding_twenty_bytes"))
	_ = enc.WriteToken(jsontext.String("more_data_here"))
	_ = enc.WriteToken(jsontext.EndObject)

	if err := nm.WriteMessage(enc); err == nil {
		t.Fatal("expected WriteMessage to fail for size > maxSize, got nil")
	}

	// Craft a header with size 100 > maxSize 20.
	var fakeStream bytes.Buffer
	fakeStream.Write([]byte{100, 0, 0, 0})
	fakeStream.Write(make([]byte, 100))

	nmReader := jsonmsgs.NewMessager(io.Discard, io.NopCloser(&fakeStream), jsonmsgs.WithMaxSize(20))
	if _, err := nmReader.ReadMessage(); err == nil {
		t.Fatal("expected ReadMessage to fail when length > maxSize, got nil")
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
	nm := jsonmsgs.NewMessager(&sbuf, io.NopCloser(bytes.NewReader(nil)))

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
	reader := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(sbuf.buf.Bytes())))
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
	nm := jsonmsgs.NewMessager(&buf, io.NopCloser(bytes.NewReader(nil)))

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

	reader := jsonmsgs.NewMessager(io.Discard, io.NopCloser(bytes.NewReader(buf.Bytes())))
	dec, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	reader.ReleaseDecoder(dec)
}
