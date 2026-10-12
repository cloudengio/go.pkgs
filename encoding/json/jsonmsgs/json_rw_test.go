// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

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
		jsonmsgs.WithEncoderOptions(jsontext.WithIndent("\t")))
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

// TestProxyDifferentSizes forwards a message between a Reader and a Writer with
// different limits, so that the message is put in frames that the next hop
// accepts.
func TestProxyDifferentSizes(t *testing.T) {
	upstream := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(4000)}
	text := strings.Repeat("a\"", 10000)
	var in bytes.Buffer
	if err := writeWithHeader(newWriter(&in, upstream...), text, `{"to":1}`); err != nil {
		t.Fatal(err)
	}
	for _, maxSize := range []uint32{8000, 4000, 2000, 100} {
		t.Run(fmt.Sprint(maxSize), func(t *testing.T) {
			downstream := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(maxSize)}
			var out bytes.Buffer
			fr, err := newReader(in.Bytes(), upstream...).ReadFragments(nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := newWriter(&out, downstream...).WriteFragments(fr); err != nil {
				t.Fatal(err)
			}
			for i, f := range splitFrames(t, out.Bytes()) {
				if len(f) > int(maxSize) {
					t.Errorf("frame %d is %d bytes", i, len(f))
				}
			}
			equalMessages(t, readMessages(t, out.Bytes(), downstream...), []message{{text, `{"to":1}`}})
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
	frame0 := `{"~":[0,50000000],"p":"a"}`
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

// TestReassembleAllocation verifies that the memory that is allocated for a
// message is for the size that the message declares, which is limited, and not
// for the size that a frame declares until its data arrives.
func TestReassembleAllocation(t *testing.T) {
	const declared = 1 << 29
	opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(declared)}
	measure := func(stream []byte, opts ...jsonmsgs.Option) (uint64, error) {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		_, err := newReader(stream, opts...).ReadMessage()
		runtime.ReadMemStats(&after)
		return after.TotalAlloc - before.TotalAlloc, err
	}

	// A frame that declares a size and does not send it.
	got, err := measure(binary.LittleEndian.AppendUint32(nil, declared), opts...)
	if !errors.Is(err, io.ErrUnexpectedEOF) || got > 4<<20 {
		t.Errorf("first frame: allocated %d bytes, %v", got, err)
	}
	got, err = measure(binary.LittleEndian.AppendUint32(joinFrames(`{"~":[0,1000000],"p":"a"}`), declared), opts...)
	if !errors.Is(err, io.ErrUnexpectedEOF) || got < 1_000_000 || got > 4<<20 {
		t.Errorf("later frame: allocated %d bytes, %v, want about 1,000,000 for a message of 1,000,000", got, err)
	}

	// A message that declares more than the maximum is refused before anything
	// is allocated for it.
	got, err = measure(joinFrames(`{"~":[0,2000000000],"p":"a"}`), append(opts, jsonmsgs.WithMaxFragmentedMessageSize(1<<20))...)
	if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || got > 1<<20 {
		t.Errorf("too large: allocated %d bytes, %v", got, err)
	}

	// And a message is allocated for once, whatever the number of frames.
	text := strings.Repeat("a\"bc", 1<<20/4)
	var in bytes.Buffer
	if err := jsonmsgs.WriteRawForTests(newWriter(&in, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(2000)), []byte(text)); err != nil {
		t.Fatal(err)
	}
	// Once for the message, or, with the race detector, which allocates more
	// for the same, twice. Growing the buffer as the frames arrive is 2 times,
	// and 4 times with the race detector.
	limit := uint64(len(text)) * 3 / 2
	if raceEnabled {
		limit = uint64(len(text)) * 3
	}
	got, err = measure(in.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(2000))
	if err != nil || got > limit {
		t.Errorf("a message of %d bytes in %d frames allocated %d bytes, %v", len(text), len(splitFrames(t, in.Bytes())), got, err)
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

func TestMaxSizeBoundPanics(t *testing.T) {
	// math.MaxInt32 - 4 is the maximum valid maxSize.
	_ = jsonmsgs.NewReader(nil, jsonmsgs.WithMaxSize(math.MaxInt32-4))
	_ = jsonmsgs.NewWriter(nil, jsonmsgs.WithMaxSize(math.MaxInt32-4))

	// Values exceeding math.MaxInt32 - 4 must panic.
	for _, invalid := range []uint32{math.MaxInt32 - 3, math.MaxInt32} {
		assertPanics(t, func() {
			jsonmsgs.NewReader(nil, jsonmsgs.WithMaxSize(invalid))
		})
		assertPanics(t, func() {
			jsonmsgs.NewWriter(nil, jsonmsgs.WithMaxSize(invalid))
		})
	}
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()
	f()
}

// TestHeaderRetention checks that a very large user header does not stay in the
// memory of the Decoder or Fragments that it was read into.
func TestHeaderRetention(t *testing.T) {
	const bigHeaderSize = 5 * 1024 * 1024
	hdr := `{"big":"` + strings.Repeat("x", bigHeaderSize-16) + `"}`
	opts := []jsonmsgs.Option{
		jsonmsgs.WithFragmentation(true),
		jsonmsgs.WithMaxHeaderSize(bigHeaderSize + 1024),
		jsonmsgs.WithMaxSize(bigHeaderSize + 2048),
		jsonmsgs.WithMaxRetainedBuffer(jsonmsgs.DefaultMaxRetainedBuffer),
	}
	var buf bytes.Buffer
	if err := writeWithHeader(newWriter(&buf, opts...), `{"a":1}`, hdr); err != nil {
		t.Fatal(err)
	}

	rd := newReader(buf.Bytes(), opts...)
	dec, err := rd.ReadMessage()
	if err != nil || string(dec.Header()) != hdr {
		t.Fatalf("got %v", err)
	}
	rd.ReleaseDecoder(dec)
	if got := jsonmsgs.DecoderHeaderCapForTests(dec); got != 0 {
		t.Errorf("the decoder kept a header buffer of %d bytes", got)
	}

	rd = newReader(buf.Bytes(), opts...)
	fr, err := rd.ReadFragments(nil)
	if err != nil || string(fr.Header()) != hdr {
		t.Fatalf("got %v", err)
	}
	// Reading into it again, whatever is read, lets go of the large header.
	rd = newReader(joinFrames(`{"a":1}`), opts...)
	if _, err := rd.ReadFragments(fr); err != nil {
		t.Fatal(err)
	}
	if got := jsonmsgs.FragmentsHeaderCapForTests(fr); got != 0 {
		t.Errorf("the fragments kept a header buffer of %d bytes", got)
	}
}

func TestReadAppendOverflow(t *testing.T) {
	rd := jsonmsgs.NewReader(nil)
	msg := make([]byte, 100)
	// Passing n such that len(msg) + n overflows math.MaxInt must return ErrMessageTooLarge.
	_, _, err := jsonmsgs.ReadAppendForTests(rd, msg, math.MaxInt, 1000)
	if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("got %v, want ErrMessageTooLarge", err)
	}
}

// TestMaxRetainedBuffer checks that buffers are kept for reuse, or not, according
// to the limit that is set.
func TestMaxRetainedBuffer(t *testing.T) {
	text := strings.Repeat("a\"b", 1000) // about 4 KB escaped
	var stream bytes.Buffer
	writeMessages(t, newWriter(&stream, jsonmsgs.WithMaxSize(100_000)), []message{{text, ""}, {`{"a":1}`, ""}})
	for _, tc := range []struct {
		name   string
		retain uint32
		kept   bool
	}{
		{"below the size of the message", 1000, false},
		{"above the size of the message", 100_000, true},
		{"default, large frames", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := []jsonmsgs.Option{jsonmsgs.WithMaxSize(100_000), jsonmsgs.WithMaxRetainedBuffer(tc.retain)}

			rd := newReader(stream.Bytes(), opts...)
			dec, err := rd.ReadMessage()
			if err != nil {
				t.Fatal(err)
			}
			rd.ReleaseDecoder(dec)
			if kept := jsonmsgs.BufferCapForTests(dec) > 0; kept != tc.kept {
				t.Errorf("decoder buffer kept is %v, want %v", kept, tc.kept)
			}

			fr, err := newReader(stream.Bytes(), opts...).ReadFragments(nil)
			if err != nil {
				t.Fatal(err)
			}
			rd = newReader(joinFrames(`{"a":1}`), opts...)
			if _, err := rd.ReadFragments(fr); err != nil {
				t.Fatal(err)
			}
			// Reading something small reuses the large buffer, if it was kept.
			if kept := jsonmsgs.FragmentsPayloadCapForTests(fr) > len(text); kept != tc.kept {
				t.Errorf("fragments buffer kept is %v, want %v", kept, tc.kept)
			}

			var sink bytes.Buffer
			wr := newWriter(&sink, opts...)
			writeMessages(t, wr, []message{{text, ""}})
			frags, err := newReader(stream.Bytes(), opts...).ReadFragments(nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := wr.WriteFragments(frags); err != nil {
				t.Fatal(err)
			}
			if kept := jsonmsgs.WriterBufferCapForTests(wr) > len(text); kept != tc.kept {
				t.Errorf("writer buffer kept is %v, want %v", kept, tc.kept)
			}
		})
	}
}

// TestMaxRetainedBufferScalesWithMaxSize checks that, by default, the buffer for
// a message is kept if it is within 4 times the maximum size, even if it is
// larger than DefaultMaxRetainedBuffer.
func TestMaxRetainedBufferScalesWithMaxSize(t *testing.T) {
	const maxSize = 2 << 20
	text := strings.Repeat("a", 5<<20)
	opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(maxSize)}
	var stream bytes.Buffer
	writeMessages(t, newWriter(&stream, opts...), []message{{text, ""}})
	fr, err := newReader(stream.Bytes(), opts...).ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newReader(joinFrames(`{"a":1}`), opts...).ReadFragments(fr); err != nil {
		t.Fatal(err)
	}
	if got := jsonmsgs.FragmentsPayloadCapForTests(fr); got < len(text) {
		t.Errorf("the buffer of %d bytes was not kept", got)
	}
}

// deadlineWriter is a writer that has a write deadline, that records how it is set.
type deadlineWriter struct {
	buf      bytes.Buffer
	deadline time.Time
	sets     []time.Time // the deadlines that were set, in order
	writes   int
	setErr   error
	atWrite  []time.Time // the deadline at each write
}

func (d *deadlineWriter) SetWriteDeadline(t time.Time) error {
	if d.setErr != nil {
		return d.setErr
	}
	d.deadline = t
	d.sets = append(d.sets, t)
	return nil
}

func (d *deadlineWriter) Write(p []byte) (int, error) {
	d.writes++
	d.atWrite = append(d.atWrite, d.deadline)
	return d.buf.Write(p)
}

// TestWriteTimeoutSetsDeadlines checks that the deadline is set before, and
// cleared after, each of the frames of a message.
func TestWriteTimeoutSetsDeadlines(t *testing.T) {
	const timeout = time.Hour
	text := strings.Repeat("a\"b", 100)
	for name, write := range map[string]func(*jsonmsgs.Writer) error{
		"WriteMessage": func(wr *jsonmsgs.Writer) error { return jsonmsgs.WriteRawForTests(wr, []byte(text)) },
		"WriteFragments": func(wr *jsonmsgs.Writer) error {
			var in bytes.Buffer
			if err := jsonmsgs.WriteRawForTests(newWriter(&in, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(1000)), []byte(text)); err != nil {
				return err
			}
			fr, err := newReader(in.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(1000)).ReadFragments(nil)
			if err != nil {
				return err
			}
			return wr.WriteFragments(fr)
		},
	} {
		t.Run(name, func(t *testing.T) {
			dw := &deadlineWriter{}
			wr := jsonmsgs.NewWriter(dw, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(100), jsonmsgs.WithWriteTimeout(timeout))
			start := time.Now()
			if err := write(wr); err != nil {
				t.Fatal(err)
			}
			if dw.writes < 3 || len(dw.sets) != 2*dw.writes {
				t.Fatalf("%d writes, %d deadlines set", dw.writes, len(dw.sets))
			}
			for i := range dw.writes {
				set, cleared := dw.sets[2*i], dw.sets[2*i+1]
				if set.Before(start.Add(timeout)) || set.After(time.Now().Add(timeout)) {
					t.Errorf("write %d: deadline %v is not a timeout from now", i, set)
				}
				if dw.atWrite[i] != set {
					t.Errorf("write %d: the deadline was %v, not %v", i, dw.atWrite[i], set)
				}
				if !cleared.IsZero() {
					t.Errorf("write %d: the deadline was not cleared: %v", i, cleared)
				}
			}
		})
	}
}

func TestWriteTimeoutIgnored(t *testing.T) {
	// Not set if there is no timeout, or the writer has no deadline.
	dw := &deadlineWriter{}
	if err := jsonmsgs.WriteRawForTests(jsonmsgs.NewWriter(dw), []byte(`{}`)); err != nil || len(dw.sets) != 0 {
		t.Errorf("got %v, %d deadlines set", err, len(dw.sets))
	}
	dw = &deadlineWriter{}
	if err := jsonmsgs.WriteRawForTests(jsonmsgs.NewWriter(dw, jsonmsgs.WithWriteTimeout(-time.Second)), []byte(`{}`)); err != nil || len(dw.sets) != 0 {
		t.Errorf("got %v, %d deadlines set with a negative timeout", err, len(dw.sets))
	}
	var buf bytes.Buffer
	if err := jsonmsgs.WriteRawForTests(jsonmsgs.NewWriter(&buf, jsonmsgs.WithWriteTimeout(time.Second)), []byte(`{}`)); err != nil || buf.Len() != 6 {
		t.Errorf("got %v, wrote %d bytes", err, buf.Len())
	}
}

func TestWriteTimeoutDeadlineError(t *testing.T) {
	errClosed := errors.New("closed")
	dw := &deadlineWriter{setErr: errClosed}
	wr := jsonmsgs.NewWriter(dw, jsonmsgs.WithWriteTimeout(time.Second))
	err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`))
	if !errors.Is(err, errClosed) || !errors.Is(err, jsonmsgs.ErrPermanent) || dw.writes != 0 {
		t.Fatalf("got %v, %d writes, want a permanent %v and nothing written", err, dw.writes, errClosed)
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error", err2)
	}
}

// TestWriteTimeoutConn checks the timeout with a connection, which does not
// complete a write if no one is reading.
func TestWriteTimeoutConn(t *testing.T) {
	const timeout = 50 * time.Millisecond
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	wr := jsonmsgs.NewWriter(local, jsonmsgs.WithWriteTimeout(timeout))

	start := time.Now()
	err := jsonmsgs.WriteRawForTests(wr, []byte(`{"a":1}`))
	var ne net.Error
	if !errors.Is(err, os.ErrDeadlineExceeded) || !errors.As(err, &ne) || !ne.Timeout() || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Fatalf("got %v, want a permanent timeout", err)
	}
	if took := time.Since(start); took < timeout || took > 20*timeout {
		t.Errorf("took %v for a timeout of %v", took, timeout)
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error", err2)
	}
}

// TestWriteTimeoutConnReading checks that a connection that is read is written
// to within the timeout, and that the deadline does not outlast the write.
func TestWriteTimeoutConnReading(t *testing.T) {
	const timeout = 200 * time.Millisecond
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	msgs := make(chan []message, 1)
	go func() {
		var got []byte
		buf := make([]byte, 100)
		for len(got) < 2*(4+7) {
			n, err := remote.Read(buf)
			got = append(got, buf[:n]...)
			if err != nil {
				break
			}
		}
		msgs <- readMessages(t, got)
	}()
	wr := jsonmsgs.NewWriter(local, jsonmsgs.WithWriteTimeout(timeout))
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	// A write after the time has passed, as the reader is slow, does not time
	// out as it would if the deadline had been left set.
	time.Sleep(2 * timeout)
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{"b":2}`)); err != nil {
		t.Fatal(err)
	}
	equalMessages(t, <-msgs, []message{{`{"a":1}`, ""}, {`{"b":2}`, ""}})
}

// closeRecorder is an io.ReadWriteCloser that records how it is closed.
type closeRecorder struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closes int
	err    error // returned by Close
}

func (c *closeRecorder) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *closeRecorder) Read([]byte) (int, error) { return 0, io.EOF }

func (c *closeRecorder) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closes++
	return c.err
}

func (c *closeRecorder) timesClosed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closes
}

func TestWriterClose(t *testing.T) {
	errClose := errors.New("close failed")
	cr := &closeRecorder{err: errClose}
	wr := jsonmsgs.NewWriter(cr)
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := wr.Close(); err != errClose { //nolint:errorlint
		t.Fatalf("got %v, want %v", err, errClose)
	}
	// Once, whatever the number of calls, with the same result.
	for range 3 {
		if err := wr.Close(); err != errClose { //nolint:errorlint
			t.Fatalf("got %v, want %v", err, errClose)
		}
	}
	if n := cr.timesClosed(); n != 1 {
		t.Errorf("closed %d times", n)
	}
	written := cr.buf.Len()

	// And nothing is written after.
	enc := wr.NewEncoder()
	_ = enc.WriteValue(jsontext.Value(`{}`))
	err := wr.WriteMessage(enc)
	if !errors.Is(err, jsonmsgs.ErrClosed) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Errorf("WriteMessage: got %v", err)
	}
	enc = wr.NewEncoder()
	_ = enc.WriteValue(jsontext.Value(`{}`))
	if err := wr.WriteMessageWithHeader(enc, jsontext.Value(`1`)); !errors.Is(err, jsonmsgs.ErrClosed) {
		t.Errorf("WriteMessageWithHeader: got %v", err)
	}
	fr, err := jsonmsgs.NewFragments(nil, "x")
	if err != nil {
		t.Fatal(err)
	}
	if err := wr.WriteFragments(fr); !errors.Is(err, jsonmsgs.ErrClosed) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Errorf("WriteFragments: got %v", err)
	}
	if cr.buf.Len() != written {
		t.Errorf("%d bytes were written after Close", cr.buf.Len()-written)
	}
}

func TestWriterCloseNotCloser(t *testing.T) {
	var buf bytes.Buffer
	wr := jsonmsgs.NewWriter(&buf)
	if err := wr.Close(); err != nil {
		t.Fatal(err)
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); !errors.Is(err, jsonmsgs.ErrClosed) || buf.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes", err, buf.Len())
	}
	if err := jsonmsgs.NewWriter(nil).Close(); err != nil {
		t.Errorf("got %v", err)
	}
}

// TestWriterCloseUnblocks checks that Close does not wait for a write that is
// blocked, and causes it to return.
func TestWriterCloseUnblocks(t *testing.T) {
	local, remote := net.Pipe() // no one reads remote, so that writes block
	defer remote.Close()
	wr := jsonmsgs.NewWriter(local)
	done := make(chan error, 1)
	go func() { done <- jsonmsgs.WriteRawForTests(wr, []byte(`{"a":1}`)) }()
	time.Sleep(50 * time.Millisecond) // for the write to block
	closed := make(chan error, 1)
	go func() { closed <- wr.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close waited for the write")
	}
	select {
	case err := <-done:
		if !errors.Is(err, jsonmsgs.ErrPermanent) {
			t.Errorf("the write returned %v, want a permanent error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the write did not return")
	}
}

// TestWriterCloseMidMessage checks that a message that is part way through is
// not completed, even if the underlying writer can still be written to.
func TestWriterCloseMidMessage(t *testing.T) {
	var wr *jsonmsgs.Writer
	sink := &closeAfterWrites{}
	wr = jsonmsgs.NewWriter(sink, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(100))
	sink.closeAt, sink.close = 2, func() { _ = wr.Close() }
	err := jsonmsgs.WriteRawForTests(wr, bytes.Repeat([]byte("a"), 1000))
	if !errors.Is(err, jsonmsgs.ErrClosed) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Fatalf("got %v", err)
	}
	if sink.writes != 2 {
		t.Errorf("%d frames were written, want 2", sink.writes)
	}
}

// closeAfterWrites calls close after closeAt writes.
type closeAfterWrites struct {
	writes, closeAt int
	close           func()
}

func (c *closeAfterWrites) Write(p []byte) (int, error) {
	c.writes++
	if c.writes == c.closeAt {
		c.close()
	}
	return len(p), nil
}

func TestWriterCloseConcurrent(t *testing.T) {
	cr := &closeRecorder{}
	wr := jsonmsgs.NewWriter(cr)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				_ = jsonmsgs.WriteRawForTests(wr, []byte(`{"a":1}`))
			}
		})
		wg.Go(func() { _ = wr.Close() })
	}
	wg.Wait()
	if n := cr.timesClosed(); n != 1 {
		t.Errorf("closed %d times", n)
	}
	// What was written before is whole frames.
	for _, f := range splitFrames(t, cr.buf.Bytes()) {
		if string(f) != `{"a":1}` {
			t.Fatalf("frame %q", f)
		}
	}
}

func TestMessagerClose(t *testing.T) {
	// One object that is read and written is closed once.
	both := &closeRecorder{err: errors.New("close failed")}
	nm := jsonmsgs.NewMessager(both, both)
	if err := nm.Close(); err == nil || err.Error() != "close failed" {
		t.Errorf("got %v", err)
	}
	if n := both.timesClosed(); n != 1 {
		t.Errorf("closed %d times", n)
	}
	if err := jsonmsgs.WriteRawForTests(nm.Writer, []byte(`{}`)); !errors.Is(err, jsonmsgs.ErrClosed) {
		t.Errorf("got %v after Close", err)
	}

	// Two are both closed, and the errors from both are returned.
	rd, wr := &closeRecorder{err: errors.New("read side")}, &closeRecorder{err: errors.New("write side")}
	nm = jsonmsgs.NewMessager(rd, wr)
	err := nm.Close()
	if err == nil || !strings.Contains(err.Error(), "read side") || !strings.Contains(err.Error(), "write side") {
		t.Errorf("got %v", err)
	}
	if rd.timesClosed() != 1 || wr.timesClosed() != 1 {
		t.Errorf("closed %d and %d times", rd.timesClosed(), wr.timesClosed())
	}

	// Not closers, and not objects that can be compared.
	if err := jsonmsgs.NewMessager(nil, nil).Close(); err != nil {
		t.Errorf("got %v", err)
	}
	if err := jsonmsgs.NewMessager(io.NopCloser(bytes.NewReader(nil)), io.Discard).Close(); err != nil {
		t.Errorf("got %v", err)
	}
}
