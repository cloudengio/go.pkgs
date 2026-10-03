// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"io"
	"math"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"

	"cloudeng.io/encoding/json/jsonmsgs"
)

// TestReaderWriterIndependentSizes checks that a Reader accepts whatever frame
// size a Writer was configured for, as long as it is within its own maximum.
func TestReaderWriterIndependentSizes(t *testing.T) {
	msg := []byte(`{"a":"` + strings.Repeat(`x"é`, 400) + `"}`)
	var stream bytes.Buffer
	wr := newWriter(&stream, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(100))
	if err := jsonmsgs.WriteRawForTests(wr, msg); err != nil {
		t.Fatal(err)
	}
	for _, maxSize := range []uint32{100, 101, 4096, 1 << 20} {
		got, err := readRaw(newReader(stream.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(maxSize)))
		if err != nil || !bytes.Equal(got, msg) {
			t.Errorf("reader with maximum size %d: %v", maxSize, err)
		}
	}
	// A reader whose maximum is smaller than the frames that are written.
	_, err := readRaw(newReader(stream.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(99)))
	if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("got %v, want ErrMessageTooLarge", err)
	}
	// A fragment size that is not that of the writer is of no consequence to a reader.
	got, err := readRaw(newReader(stream.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(1<<20), jsonmsgs.WithFragmentSize(5)))
	if err != nil || !bytes.Equal(got, msg) {
		t.Errorf("reader with a fragment size: %v", err)
	}
}

func TestReaderWriterIgnoreOptions(t *testing.T) {
	// Options for the other of the two are ignored, including those that
	// would be invalid for it.
	jsonmsgs.NewReader(nil, jsonmsgs.WithFragmentation(true), jsonmsgs.WithFragmentSize(5),
		jsonmsgs.WithRepeatHeader(true), jsonmsgs.WithEncoderOptions(jsontext.WithIndent("\t")))
	jsonmsgs.NewWriter(nil, jsonmsgs.WithDecoderOptions(jsontext.AllowDuplicateNames(true)))

	for name, f := range map[string]func(){
		"reader maximum size": func() { jsonmsgs.NewReader(nil, jsonmsgs.WithMaxSize(1)) },
		"writer maximum size": func() { jsonmsgs.NewWriter(nil, jsonmsgs.WithMaxSize(1)) },
		"reader header size":  func() { jsonmsgs.NewReader(nil, jsonmsgs.WithMaxHeaderSize(1<<31)) },
		"writer fragment size": func() {
			jsonmsgs.NewWriter(nil, jsonmsgs.WithFragmentation(true), jsonmsgs.WithFragmentSize(5))
		},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected a panic", name)
				}
			}()
			f()
		}()
	}
}

// TestProxyDifferentSizes forwards frames between a Reader and a Writer with
// different limits.
func TestProxyDifferentSizes(t *testing.T) {
	upstream := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(4000)}
	var in bytes.Buffer
	if err := writeWithHeader(newWriter(&in, upstream...), strings.Repeat("a", 20000), `{"to":1}`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		maxSize uint32
		ok      bool
	}{
		{"larger", 8000, true},
		{"same", 4000, true},
		{"smaller", 2000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rd := newReader(in.Bytes(), upstream...)
			var out bytes.Buffer
			wr := newWriter(&out, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(tc.maxSize))
			var err error
			for last := false; !last && err == nil; {
				var f *jsonmsgs.Fragment
				if f, err = rd.ReadFragment(); err != nil {
					t.Fatal(err)
				}
				last = f.Last
				err = wr.WriteFragment(f)
			}
			if tc.ok != (err == nil) || (!tc.ok && !errors.Is(err, jsonmsgs.ErrMessageTooLarge)) {
				t.Fatalf("got %v", err)
			}
			if !tc.ok {
				return
			}
			dec, err := newReader(out.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(tc.maxSize)).ReadMessage()
			if err != nil || string(dec.Header()) != `{"to":1}` || string(jsonmsgs.DecoderBytesForTests(dec)) != strings.Repeat("a", 20000) {
				t.Errorf("read back: %v", err)
			}
		})
	}
}

func TestMessagerIsReaderAndWriter(t *testing.T) {
	var buf bytes.Buffer
	nm := jsonmsgs.NewMessager(io.NopCloser(&buf), &buf, jsonmsgs.WithMaxHeaderSize(8), jsonmsgs.WithMaxSize(100))
	enc := nm.NewEncoder()
	_ = enc.WriteValue(jsontext.Value(`{"a":1}`))
	if err := nm.WriteMessageWithHeader(enc, jsontext.Value(`{"b":1}`)); err != nil {
		t.Fatal(err)
	}
	dec, err := nm.ReadMessage()
	if err != nil || string(dec.Header()) != `{"b":1}` {
		t.Fatalf("got %v", err)
	}
	nm.ReleaseDecoder(dec)
	if err := nm.Close(); err != nil {
		t.Fatal(err)
	}
	if nm.Reader == nil || nm.Writer == nil {
		t.Error("a Messager has a Reader and a Writer")
	}
}

// TestPermanentAndRetryableErrors checks that the errors that leave a Reader
// or Writer unusable are told apart from those that do not.
func TestPermanentAndRetryableErrors(t *testing.T) {
	// A message that cannot be written, found before anything is: retryable.
	var buf bytes.Buffer
	wr := newWriter(&buf, jsonmsgs.WithMaxSize(10))
	for name, msg := range map[string][]byte{
		"too large": []byte(`{"a":"bbbbbbbbbb"}`),
		"empty":     nil,
		"reserved":  []byte(`{"~":[1,2,3,4,5,6]}`),
	} {
		err := jsonmsgs.WriteRawForTests(wr, msg)
		if err == nil || errors.Is(err, jsonmsgs.ErrPermanent) {
			t.Errorf("%s: got %v, want an error that can be retried", name, err)
		}
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err != nil {
		t.Errorf("after the errors: %v", err)
	}

	// An error from the underlying writer: permanent, the same for each call, and
	// the text of the error is not changed.
	fw := &failAfter{n: 0}
	wr = newWriter(fw)
	err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`))
	if !errors.Is(err, jsonmsgs.ErrPermanent) || !errors.Is(err, errDisk) {
		t.Fatalf("got %v", err)
	}
	if strings.Contains(err.Error(), "permanent") {
		t.Errorf("the text of the error was changed: %v", err)
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error, %v", err2, err)
	}

	// The end of the stream is neither, and an error part way through a
	// message is permanent.
	rd := newReader(nil)
	if _, err := rd.ReadMessage(); err != io.EOF { //nolint:errorlint
		t.Errorf("got %v, want io.EOF", err)
	}
	rd = newReader(joinFrames(`{"a":1}`)[:6])
	if _, err := rd.ReadMessage(); !errors.Is(err, io.ErrUnexpectedEOF) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Errorf("got %v, want a permanent ErrUnexpectedEOF", err)
	}
}

// TestReassembleAllocationBounds verifies that reassembly buffer allocation
// handles large limits without overflowing or panicking.
func TestReassembleAllocationBounds(t *testing.T) {
	frame0 := `{"~":[0,2000000000],"p":"a"}`
	frame1 := `{"~":[1],"p":"` + strings.Repeat("b", 300) + `"}`
	stream := joinFrames(frame0, frame1)

	rd := newReader(stream,
		jsonmsgs.WithFragmentation(true),
		jsonmsgs.WithMaxFragmentedMessageSize(math.MaxInt32),
		jsonmsgs.WithMaxSize(1<<29),
	)
	_, err := rd.ReadMessage()
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
	}
}

// TestReassembleDoesNotAllocateAheadOfData verifies that the size that a frame
// declares, which may be as large as the maximum, does not cause memory to be
// allocated for the frame until its data arrives.
func TestReassembleDoesNotAllocateAheadOfData(t *testing.T) {
	const declared = 1 << 29
	stream := joinFrames(`{"~":[0,2000000000],"p":"a"}`)
	stream = binary.LittleEndian.AppendUint32(stream, declared) // and no body
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	rd := newReader(stream,
		jsonmsgs.WithFragmentation(true),
		jsonmsgs.WithMaxFragmentedMessageSize(math.MaxInt32),
		jsonmsgs.WithMaxSize(declared),
	)
	_, err := rd.ReadMessage()
	runtime.ReadMemStats(&after)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want io.ErrUnexpectedEOF", err)
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 4<<20 {
		t.Errorf("allocated %d bytes for a frame that declared %d and sent none", got, declared)
	}
}

// TestReassembleLaterFrames checks messages whose later frames are larger than
// the chunks that they are read in, and that arrive in pieces.
func TestReassembleLaterFrames(t *testing.T) {
	msg := []byte(`{"a":"` + strings.Repeat(`é"x`, 100_000) + `"}`)
	for _, size := range []uint32{100, 1000, 65_536, 65_537, 200_000, 1 << 20} {
		var stream bytes.Buffer
		opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(size)}
		if err := jsonmsgs.WriteRawForTests(newWriter(&stream, opts...), msg); err != nil {
			t.Fatal(err)
		}
		// Read a byte at a time, so that the data arrives in the smallest pieces.
		rd := jsonmsgs.NewReader(io.NopCloser(iotest.OneByteReader(bytes.NewReader(stream.Bytes()))), opts...)
		got, err := readRaw(rd)
		if err != nil || !bytes.Equal(got, msg) {
			t.Errorf("frame size %d: %v", size, err)
		}
	}
}
