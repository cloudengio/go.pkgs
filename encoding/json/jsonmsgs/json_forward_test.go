// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	"encoding/binary"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

// forward forwards every message in stream, read with ropts, to a buffer, using
// ReadFragments and WriteFragments with wopts, and returns what was written.
func forward(t testing.TB, stream []byte, ropts, wopts []jsonmsgs.Option) []byte {
	t.Helper()
	var out bytes.Buffer
	rd, wr := newReader(stream, ropts...), newWriter(&out, wopts...)
	var fr *jsonmsgs.Fragments
	for {
		var err error
		fr, err = rd.ReadFragments(fr)
		if err == io.EOF { //nolint:errorlint
			return out.Bytes()
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragments(fr); err != nil {
			t.Fatal(err)
		}
	}
}

type message struct {
	text, header string
}

// readMessages reads all of the messages in stream.
func readMessages(t testing.TB, stream []byte, opts ...jsonmsgs.Option) []message {
	t.Helper()
	var msgs []message
	rd := newReader(stream, opts...)
	for {
		dec, err := rd.ReadMessage()
		if err == io.EOF { //nolint:errorlint
			return msgs
		}
		if err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, message{string(jsonmsgs.DecoderBytesForTests(dec)), string(dec.Header())})
		rd.ReleaseDecoder(dec)
	}
}

// writeMessages writes msgs, with their headers if they have one, using wr.
func writeMessages(t testing.TB, wr *jsonmsgs.Writer, msgs []message) {
	t.Helper()
	for _, m := range msgs {
		var err error
		if m.header != "" {
			err = writeWithHeader(wr, m.text, m.header)
		} else {
			err = jsonmsgs.WriteRawForTests(wr, []byte(m.text))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

// forwardMessages returns a mix of messages: small and large, with and without
// a user header, and that need escaping.
func forwardMessages(r *rand.Rand, n int) []message {
	var msgs []message
	for i := range n {
		m := message{text: string(randomText(r, 1+r.IntN(500)))}
		switch i % 4 {
		case 0:
			m.text = `{"a":1}`
		case 1:
			m.text = string(longRunText(r, 1+r.IntN(2000)))
		}
		if i%3 != 0 {
			m.header = fmt.Sprintf(`{"to":%d}`, i)
		}
		msgs = append(msgs, m)
	}
	return msgs
}

func equalMessages(t *testing.T, got, want []message) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message %d: got %q (header %q), want %q (header %q)", i, got[i].text, got[i].header, want[i].text, want[i].header)
		}
	}
}

// TestForwardIdentical checks that messages forwarded with options that are the
// same as those that they were written with are the same, byte for byte.
func TestForwardIdentical(t *testing.T) {
	r := rand.New(rand.NewPCG(21, 22)) //nolint:gosec
	for _, size := range []uint32{120, 300, 1000, 1 << 20} {
		opts := headerOpts(30, jsonmsgs.WithMaxSize(size))
		msgs := forwardMessages(r, 60)
		var in bytes.Buffer
		writeMessages(t, newWriter(&in, opts...), msgs)
		if out := forward(t, in.Bytes(), opts, opts); !bytes.Equal(out, in.Bytes()) {
			t.Fatalf("frame size %d: the forwarded stream is different", size)
		}
	}
}

// TestForwardRefragments forwards between frame sizes, larger and smaller.
func TestForwardRefragments(t *testing.T) {
	r := rand.New(rand.NewPCG(23, 24)) //nolint:gosec
	msgs := forwardMessages(r, 80)
	for _, from := range []uint32{100, 400, 5000} {
		var in bytes.Buffer
		ropts := headerOpts(30, jsonmsgs.WithMaxSize(from))
		writeMessages(t, newWriter(&in, ropts...), msgs)
		frames := len(splitFrames(t, in.Bytes()))
		for _, to := range []uint32{80, 100, 150, 400, 1000, 5000, 1 << 20} {
			wopts := headerOpts(30, jsonmsgs.WithMaxSize(to))
			// Read with the size of the frames that were written, and the other
			// end with that of the frames that are forwarded.
			out := forward(t, in.Bytes(), ropts, wopts)
			equalMessages(t, readMessages(t, out, headerOpts(30, jsonmsgs.WithMaxSize(max(from, to)))...), msgs)
			for i, f := range splitFrames(t, out) {
				if len(f) > int(to) {
					t.Fatalf("%d to %d: frame %d is %d bytes", from, to, i, len(f))
				}
			}
			if to >= from && len(splitFrames(t, out)) > frames {
				t.Errorf("%d to %d: %d frames, from %d", from, to, len(splitFrames(t, out)), frames)
			}
		}
	}
}

// TestForwardCoalesces checks that frames are put together if they fit.
func TestForwardCoalesces(t *testing.T) {
	msg := message{strings.Repeat("a\"b", 1200), `{"to":1}`}
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, headerOpts(30, jsonmsgs.WithMaxSize(200))...), []message{msg})
	if n := len(splitFrames(t, in.Bytes())); n < 10 {
		t.Fatalf("only %d frames to coalesce", n)
	}
	out := forward(t, in.Bytes(), headerOpts(30, jsonmsgs.WithMaxSize(200)), headerOpts(30, jsonmsgs.WithMaxSize(1<<20)))
	if n := len(splitFrames(t, out)); n != 1 {
		t.Errorf("got %d frames, want 1", n)
	}
	equalMessages(t, readMessages(t, out, headerOpts(30)...), []message{msg})
	// Without fragmentation it must be one.
	out = forward(t, in.Bytes(), headerOpts(30, jsonmsgs.WithMaxSize(200)), []jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(30), jsonmsgs.WithMaxSize(1 << 20)})
	if n := len(splitFrames(t, out)); n != 1 {
		t.Errorf("got %d frames, want 1", n)
	}
}

// checkAccessors checks what fr holds: its user header, whether it is bare, the
// size of the message and, if frames is not 0, the number of frames.
func checkAccessors(t *testing.T, fr *jsonmsgs.Fragments, header string, bare bool, length, frames int) {
	t.Helper()
	if string(fr.Header()) != header || (header == "") != (fr.Header() == nil) {
		t.Errorf("header %q, want %q", fr.Header(), header)
	}
	if fr.Bare() != bare || fr.Len() != length || (frames != 0 && fr.Frames() != frames) {
		t.Errorf("bare %v len %d frames %d, want %v, %d, %d", fr.Bare(), fr.Len(), fr.Frames(), bare, length, frames)
	}
}

func TestFragmentsAccessors(t *testing.T) {
	opts := headerOpts(30, jsonmsgs.WithMaxSize(100))
	text := strings.Repeat("a\"é", 100)
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{{text, `{"to":1}`}, {`{"a":1}`, ""}, {text, ""}})
	rd := newReader(in.Bytes(), opts...)

	fr, err := rd.ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	checkAccessors(t, fr, `{"to":1}`, false, len(text), 0)
	if fr.Frames() < 3 {
		t.Errorf("%d frames", fr.Frames())
	}
	if msg, err := jsonmsgs.Unescape(nil, fr.Payload()); err != nil || string(msg) != text {
		t.Errorf("payload does not stand for the message: %v", err)
	}

	// A small message without a header is bare, and the Fragments are reused.
	if fr2, err := rd.ReadFragments(fr); err != nil || fr2 != fr {
		t.Fatalf("got %v, %v, want the Fragments that were passed", fr2, err)
	}
	checkAccessors(t, fr, "", true, 7, 1)
	if string(fr.Payload()) != `{"a":1}` {
		t.Errorf("payload %q", fr.Payload())
	}

	// A message that is not bare, without a header.
	if _, err := rd.ReadFragments(fr); err != nil {
		t.Fatal(err)
	}
	checkAccessors(t, fr, "", false, len(text), 0)
	checkFragmentsEmptied(t, rd, fr)
}

func TestFragmentsNilAccessors(t *testing.T) {
	var fr *jsonmsgs.Fragments
	if fr.Header() != nil {
		t.Errorf("Header() = %v, want nil", fr.Header())
	}
	if fr.Len() != 0 {
		t.Errorf("Len() = %d, want 0", fr.Len())
	}
	if fr.Bare() {
		t.Errorf("Bare() = true, want false")
	}
	if fr.Frames() != 0 {
		t.Errorf("Frames() = %d, want 0", fr.Frames())
	}
	if fr.Payload() != nil {
		t.Errorf("Payload() = %v, want nil", fr.Payload())
	}
	r := fr.Reader()
	buf := make([]byte, 10)
	n, err := r.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("Reader().Read() = (%d, %v), want (0, io.EOF)", n, err)
	}
}

// checkFragmentsEmptied checks that the end of the stream leaves nothing in fr.
func checkFragmentsEmptied(t *testing.T, rd *jsonmsgs.Reader, fr *jsonmsgs.Fragments) {
	t.Helper()
	if got, err := rd.ReadFragments(fr); got != nil || err != io.EOF { //nolint:errorlint
		t.Fatalf("got %v, %v", got, err)
	}
	if fr.Len() != 0 || fr.Frames() != 0 || len(fr.Payload()) != 0 {
		t.Errorf("fragments were not emptied: len %d frames %d", fr.Len(), fr.Frames())
	}
}

func TestFragmentsBare(t *testing.T) {
	var in bytes.Buffer
	msg := `{"a":"` + strings.Repeat("é\"", 300) + `"}`
	if err := jsonmsgs.WriteRawForTests(newWriter(&in, jsonmsgs.WithMaxSize(1<<20)), []byte(msg)); err != nil {
		t.Fatal(err)
	}
	ropts := []jsonmsgs.Option{jsonmsgs.WithMaxSize(1 << 20)}

	// As a frame of its own where it fits.
	out := forward(t, in.Bytes(), ropts, []jsonmsgs.Option{jsonmsgs.WithMaxSize(2000)})
	if !bytes.Equal(out, in.Bytes()) {
		t.Errorf("a bare message was changed")
	}
	// In envelopes, if fragmentation is enabled, where it does not.
	out = forward(t, in.Bytes(), ropts, []jsonmsgs.Option{jsonmsgs.WithMaxSize(300), jsonmsgs.WithFragmentation(true)})
	if n := len(splitFrames(t, out)); n < 3 {
		t.Errorf("got %d frames", n)
	}
	equalMessages(t, readMessages(t, out, jsonmsgs.WithMaxSize(300), jsonmsgs.WithFragmentation(true)), []message{{msg, ""}})
	// And it is too large if it is not.
	fr, err := newReader(in.Bytes(), ropts...).ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	var sink bytes.Buffer
	err = newWriter(&sink, jsonmsgs.WithMaxSize(300)).WriteFragments(fr)
	if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || errors.Is(err, jsonmsgs.ErrPermanent) || sink.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes", err, sink.Len())
	}
}

// readOneFragments returns the Fragments of a message with a user header and a
// payload that needs 3 frames of 100 bytes.
func readOneFragments(t *testing.T) (*jsonmsgs.Fragments, string) {
	t.Helper()
	opts := headerOpts(16, jsonmsgs.WithMaxSize(100))
	text := strings.Repeat("a\"b", 100)
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{{text, `{"to":1}`}})
	fr, err := newReader(in.Bytes(), opts...).ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	return fr, text
}

func TestWriteFragmentsErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		opts []jsonmsgs.Option
		want error
	}{
		"header too large":        {headerOpts(4, jsonmsgs.WithMaxSize(100)), jsonmsgs.ErrMessageTooLarge},
		"headers not enabled":     {[]jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(100)}, jsonmsgs.ErrInvalidFrame},
		"message too large":       {headerOpts(16, jsonmsgs.WithMaxSize(100), jsonmsgs.WithMaxFragmentedMessageSize(200)), jsonmsgs.ErrMessageTooLarge},
		"not in one frame":        {[]jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(100)}, jsonmsgs.ErrMessageTooLarge},
		"not in one large frame":  {[]jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(300)}, jsonmsgs.ErrMessageTooLarge},
		"not in one frame, small": {[]jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(3)}, jsonmsgs.ErrMessageTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			fr, _ := readOneFragments(t)
			var out bytes.Buffer
			err := newWriter(&out, tc.opts...).WriteFragments(fr)
			if !errors.Is(err, tc.want) || errors.Is(err, jsonmsgs.ErrPermanent) || out.Len() != 0 {
				t.Fatalf("got %v, wrote %d bytes, want a %v that can be retried", err, out.Len(), tc.want)
			}
		})
	}
}

func TestWriteFragmentsNothing(t *testing.T) {
	wr := newWriter(io.Discard, headerOpts(16, jsonmsgs.WithMaxSize(100))...)
	for _, fr := range []*jsonmsgs.Fragments{nil, {}} {
		if err := wr.WriteFragments(fr); !errors.Is(err, jsonmsgs.ErrInvalidFrame) || errors.Is(err, jsonmsgs.ErrPermanent) {
			t.Errorf("got %v", err)
		}
	}
}

// TestWriteFragmentsRetried checks that what is refused can be written to a
// Writer that accepts it, and to the one that refused it if it is not changed.
func TestWriteFragmentsRetried(t *testing.T) {
	fr, text := readOneFragments(t)
	var out bytes.Buffer
	if err := newWriter(&out, headerOpts(4, jsonmsgs.WithMaxSize(100))...).WriteFragments(fr); err == nil || out.Len() != 0 {
		t.Fatalf("got %v, wrote %d bytes, want an error", err, out.Len())
	}
	// To a writer that accepts the header.
	opts := headerOpts(16, jsonmsgs.WithMaxSize(100))
	if err := newWriter(&out, opts...).WriteFragments(fr); err != nil {
		t.Fatal(err)
	}
	equalMessages(t, readMessages(t, out.Bytes(), opts...), []message{{text, `{"to":1}`}})
}

func TestWriteFragmentsIOError(t *testing.T) {
	fr, _ := readOneFragments(t)
	fw := &failAfter{n: 150}
	wr := newWriter(fw, headerOpts(16, jsonmsgs.WithMaxSize(100))...)
	err := wr.WriteFragments(fr)
	if !errors.Is(err, errDisk) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Fatalf("got %v, want a permanent %v", err, errDisk)
	}
	fw.n = 1 << 20
	if err2 := wr.WriteFragments(fr); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error, %v", err2, err)
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error, %v", err2, err)
	}
}

func TestWriteFragmentsAgain(t *testing.T) {
	fr, text := readOneFragments(t)
	var out bytes.Buffer
	opts := headerOpts(16, jsonmsgs.WithMaxSize(100))
	wr := newWriter(&out, opts...)
	for range 3 {
		if err := wr.WriteFragments(fr); err != nil {
			t.Fatal(err)
		}
	}
	equalMessages(t, readMessages(t, out.Bytes(), opts...), []message{{text, `{"to":1}`}, {text, `{"to":1}`}, {text, `{"to":1}`}})
}

// TestReadFragmentsAllocation checks the memory that is allocated for a message.
// It is for the size that the message declares, which is limited, and not for
// the size that a frame declares, which is not allocated for until its data
// arrives.
func TestReadFragmentsAllocation(t *testing.T) {
	const declared = 1 << 29
	opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(declared)}
	measure := func(stream []byte, opts ...jsonmsgs.Option) (uint64, error) {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		_, err := newReader(stream, opts...).ReadFragments(nil)
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
		t.Errorf("later frame: allocated %d bytes, %v, want the 1,250,000 for a message of 1,000,000", got, err)
	}

	// A message that declares a size that is more than the maximum is refused
	// before anything is allocated for it.
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
	// Allocated once, for 1.25 times the size of the message, or, with the race
	// detector, which allocates more for the same, 2.5 times. Growing the buffer
	// as the frames arrive is 5 times.
	limit := uint64(len(text)) * 3 / 2
	if raceEnabled {
		limit = uint64(len(text)) * 3
	}
	got, err = measure(in.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(2000))
	if err != nil || got > limit {
		t.Errorf("a message of %d bytes in %d frames allocated %d bytes, %v", len(text), len(splitFrames(t, in.Bytes())), got, err)
	}
}

// TestReadFragmentsOneByteAtATime reads messages that arrive in the smallest pieces.
func TestReadFragmentsOneByteAtATime(t *testing.T) {
	r := rand.New(rand.NewPCG(25, 26)) //nolint:gosec
	msgs := forwardMessages(r, 20)
	opts := headerOpts(30, jsonmsgs.WithMaxSize(150))
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), msgs)
	rd := jsonmsgs.NewReader(io.NopCloser(&oneByteReader{in.Bytes()}), opts...)
	var out bytes.Buffer
	wr := newWriter(&out, opts...)
	var fr *jsonmsgs.Fragments
	for {
		var err error
		if fr, err = rd.ReadFragments(fr); err == io.EOF { //nolint:errorlint
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragments(fr); err != nil {
			t.Fatal(err)
		}
	}
	equalMessages(t, readMessages(t, out.Bytes(), opts...), msgs)
}

type oneByteReader struct{ b []byte }

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(r.b) == 0 {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0], r.b = r.b[0], r.b[1:]
	return 1, nil
}

// FuzzForward checks that, whatever the stream, forwarding it with ReadFragments
// and WriteFragments is the same as reading it with ReadMessage: the same
// messages, headers and errors, whatever the size of the frames that are written.
func FuzzForward(f *testing.F) {
	for _, frag := range []bool{false, true} {
		f.Add(joinFrames(`{"a":1}`), frag, uint16(100))
		f.Add(joinFrames(`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1],"p":"def"}`), frag, uint16(40))
		f.Add(joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[1],"p":"dé"}`, `{"a":1}`), frag, uint16(1000))
		f.Add(joinFrames(`{"~":[0,3,2],"h":{},"p":"abc"}`), frag, uint16(60))
	}
	f.Fuzz(func(t *testing.T, stream []byte, frag bool, size uint16) {
		ropts := headerOpts(32, jsonmsgs.WithMaxSize(1<<16), jsonmsgs.WithMaxFragmentedMessageSize(1<<20), jsonmsgs.WithFragmentation(frag))
		wsize := uint32(size) + jsonmsgs.MinFragmentSizeWithHeader(1<<20, 32)
		wopts := headerOpts(32, jsonmsgs.WithMaxSize(wsize), jsonmsgs.WithMaxFragmentedMessageSize(1<<20), jsonmsgs.WithFragmentation(true))
		collect := func(rd *jsonmsgs.Reader) (rs []message, failed bool) {
			for {
				dec, err := rd.ReadMessage()
				if err != nil {
					return rs, err != io.EOF //nolint:errorlint
				}
				rs = append(rs, message{string(jsonmsgs.DecoderBytesForTests(dec)), string(dec.Header())})
				rd.ReleaseDecoder(dec)
			}
		}
		want, wantFailed := collect(newReader(stream, ropts...))

		var out bytes.Buffer
		rd, wr := newReader(stream, ropts...), newWriter(&out, wopts...)
		var fr *jsonmsgs.Fragments
		var failed bool
		for {
			var err error
			if fr, err = rd.ReadFragments(fr); err != nil {
				failed = err != io.EOF //nolint:errorlint
				break
			}
			if err := wr.WriteFragments(fr); err != nil {
				// Only a message that is larger than the size of the frames that
				// can be written can be refused, or one that is not valid UTF-8.
				if !errors.Is(err, jsonmsgs.ErrInvalidFrame) && !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
					t.Fatalf("WriteFragments of a message that was read: %v", err)
				}
				return
			}
		}
		if failed != wantFailed {
			t.Fatalf("ReadFragments failed %v, ReadMessage %v", failed, wantFailed)
		}
		got, _ := collect(newReader(out.Bytes(), wopts...))
		if len(got) != len(want) {
			t.Fatalf("got %d messages, want %d", len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("message %d: got %q, want %q", i, got[i], want[i])
			}
		}
	})
}

// TestWriteFragmentsExactFit checks the size at which a message goes in a single
// frame, and that it is too large when it is one byte smaller.
func TestWriteFragmentsExactFit(t *testing.T) {
	text := strings.Repeat("a\"b", 50)
	var big bytes.Buffer
	writeMessages(t, newWriter(&big, headerOpts(16, jsonmsgs.WithMaxSize(1<<20))...), []message{{text, `{"to":1}`}})
	exact := uint32(len(splitFrames(t, big.Bytes())[0]))
	read := func() *jsonmsgs.Fragments {
		// In frames of different sizes than those it is written in.
		fr, err := newReader(big.Bytes(), headerOpts(16, jsonmsgs.WithMaxSize(1<<20))...).ReadFragments(nil)
		if err != nil {
			t.Fatal(err)
		}
		return fr
	}
	for _, frag := range []bool{false, true} {
		opts := []jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithFragmentation(frag)}
		var out bytes.Buffer
		if err := newWriter(&out, append(opts, jsonmsgs.WithMaxSize(exact))...).WriteFragments(read()); err != nil {
			t.Fatalf("fragmentation %v: %v", frag, err)
		}
		if n := len(splitFrames(t, out.Bytes())); n != 1 || out.Len() != int(exact)+4 {
			t.Errorf("fragmentation %v: %d frames of %d bytes, want 1 of %d", frag, n, out.Len(), exact)
		}
		out.Reset()
		err := newWriter(&out, append(opts, jsonmsgs.WithMaxSize(exact-1))...).WriteFragments(read())
		if frag {
			if err != nil || len(splitFrames(t, out.Bytes())) != 2 {
				t.Errorf("fragmentation %v: got %v, %d frames, want 2", frag, err, len(splitFrames(t, out.Bytes())))
			}
		} else if !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || out.Len() != 0 {
			t.Errorf("fragmentation %v: got %v, wrote %d bytes", frag, err, out.Len())
		}
	}
}

// TestWriteFragmentsCorrupt checks the handling of Fragments that can not have been read.
func TestWriteFragmentsCorrupt(t *testing.T) {
	opts := headerOpts(16, jsonmsgs.WithMaxSize(100))
	ok := bytes.Repeat([]byte("a"), 60)
	bad := bytes.Repeat([]byte(`a"`), 60) // not escaped, and larger than a frame
	// Bad in the first frame of a message, so that nothing has been written.
	var out bytes.Buffer
	wr := newWriter(&out, opts...)
	fr := jsonmsgs.NewFragmentsForTests(false, 120, nil, append(bytes.Clone(bad), ok...), []int{len(bad), len(bad) + len(ok)})
	err := wr.WriteFragments(fr)
	if !errors.Is(err, jsonmsgs.ErrInvalidFrame) || errors.Is(err, jsonmsgs.ErrPermanent) || out.Len() != 0 {
		t.Fatalf("got %v, wrote %d bytes, want an ErrInvalidFrame that can be retried", err, out.Len())
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err != nil {
		t.Errorf("after the error: %v", err)
	}
	// Bad after part of the message has been written, so that it can not be completed.
	out.Reset()
	fr = jsonmsgs.NewFragmentsForTests(false, 120, nil, append(bytes.Clone(ok), bad...), []int{len(ok), len(ok) + len(bad)})
	err = wr.WriteFragments(fr)
	if !errors.Is(err, jsonmsgs.ErrInvalidFrame) || !errors.Is(err, jsonmsgs.ErrPermanent) || out.Len() == 0 {
		t.Fatalf("got %v, wrote %d bytes, want a permanent ErrInvalidFrame after writing", err, out.Len())
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want the same error", err2)
	}
}

// TestReadFragmentsErrorEmpties checks that nothing of a message that failed to
// be read is left in the Fragments that it was being read into.
func TestReadFragmentsErrorEmpties(t *testing.T) {
	rd := newReader(joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[2],"p":"def"}`), jsonmsgs.WithFragmentation(true))
	fr := &jsonmsgs.Fragments{}
	if got, err := rd.ReadFragments(fr); got != nil || !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Fatalf("got %v, %v", got, err)
	}
	if fr.Len() != 0 || fr.Frames() != 0 || len(fr.Payload()) != 0 {
		t.Errorf("len %d, %d frames, %d bytes of payload were left", fr.Len(), fr.Frames(), len(fr.Payload()))
	}
}

// TestForwardReservedPrefix forwards messages that begin with the bytes that mark
// an envelope, which are sent in an envelope even if they would fit in a frame.
func TestForwardReservedPrefix(t *testing.T) {
	msgs := []message{
		{`{"~":[1,2]}`, ""},
		{`{"~":[0,3],"p":"abc"}`, `{"to":1}`},
		{`{"~":[` + strings.Repeat("1,", 300) + `1]}`, ""},
		{`{"~":[0,1,2],"h":{},"p":"x"}`, `{"to":2}`},
	}
	for _, frag := range []bool{false, true} {
		ropts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(frag), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(1000)}
		var in bytes.Buffer
		writeMessages(t, newWriter(&in, ropts...), msgs)
		// They are not bare, and so are not written as bare frames.
		rd := newReader(in.Bytes(), ropts...)
		for i := range msgs {
			fr, err := rd.ReadFragments(nil)
			if err != nil || fr.Bare() || fr.Len() != len(msgs[i].text) {
				t.Fatalf("fragmentation %v, message %d: got %v, bare %v, len %d", frag, i, err, fr.Bare(), fr.Len())
			}
		}
		for _, size := range []uint32{1000, 80, 100_000} {
			wopts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(size)}
			out := forward(t, in.Bytes(), ropts, wopts)
			for i, f := range splitFrames(t, out) {
				if len(f) > int(size) || !bytes.HasPrefix(f, []byte(`{"~":[`)) {
					t.Fatalf("fragmentation %v, size %d: frame %d is %q", frag, size, i, f)
				}
			}
			equalMessages(t, readMessages(t, out, wopts...), msgs)
		}
	}
}

// TestForwardConcurrent forwards messages from several streams, each by a
// goroutine, to one Writer, which writes each message whole.
func TestForwardConcurrent(t *testing.T) {
	const streams, perStream = 6, 20
	r := rand.New(rand.NewPCG(27, 28)) //nolint:gosec
	var want []message
	var inputs [][]byte
	for s := range streams {
		var msgs []message
		for i, m := range forwardMessages(r, perStream) {
			m.text = fmt.Sprintf("%d/%d:", s, i) + m.text // so that each is different
			m.header = `{"to":` + fmt.Sprint(s) + `}`
			msgs = append(msgs, m)
		}
		want = append(want, msgs...)
		var in bytes.Buffer
		writeMessages(t, newWriter(&in, headerOpts(30, jsonmsgs.WithMaxSize(300))...), msgs)
		inputs = append(inputs, in.Bytes())
	}
	var sink safeBuffer
	wr := newWriter(&sink, headerOpts(30, jsonmsgs.WithMaxSize(150))...)
	var wg sync.WaitGroup
	for _, in := range inputs {
		wg.Go(func() {
			rd := newReader(in, headerOpts(30, jsonmsgs.WithMaxSize(300))...)
			var fr *jsonmsgs.Fragments
			for {
				var err error
				if fr, err = rd.ReadFragments(fr); err == io.EOF { //nolint:errorlint
					return
				} else if err != nil {
					t.Error(err)
					return
				}
				if err := wr.WriteFragments(fr); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Wait()
	got := readMessages(t, sink.Bytes(), headerOpts(30, jsonmsgs.WithMaxSize(150))...)
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	seen := map[message]bool{}
	for _, m := range got {
		seen[m] = true
	}
	for _, m := range want {
		if !seen[m] {
			t.Fatalf("missing or interleaved: %q", m.text)
		}
	}
}

// runNode sends the frames in stream through the JavaScript implementation, that
// writes frames of at most budget bytes.
func runNode(t *testing.T, node string, budget int, stream []byte) []byte {
	t.Helper()
	cmd := exec.Command(node, "testdata/fragments.js", fmt.Sprint(budget)) //nolint:gosec
	cmd.Stdin = bytes.NewReader(stream)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node: %v\n%s", err, stderr.String())
	}
	return out.Bytes()
}

// TestForwardThroughNode has a proxy, in this package, forward messages to, and
// from, an implementation in JavaScript that uses different frame sizes.
func TestForwardThroughNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	r := rand.New(rand.NewPCG(29, 30)) //nolint:gosec
	msgs := []message{
		{`{"a":1}`, ""},
		{`{"~":[1,2]}`, `{"to":1}`},
		{`{"text":"` + strings.Repeat(`é"\世😀 `, 500) + `"}`, `{"to":2,"via":["a"]}`},
		{`{"k":"` + strings.Repeat("a", 5000) + `"}`, ""},
		{string(randomText(r, 3000)), `{"to":3}`},
		{"{\n\t\"a\":\t[1,\n2]\n}" + strings.Repeat(" \n\t\r", 400), ""},
	}
	opts := func(size uint32) []jsonmsgs.Option { return headerOpts(40, jsonmsgs.WithMaxSize(size)) }
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts(700)...), msgs)

	// Forwarded to the JavaScript in smaller frames, that writes its own, which are
	// forwarded back in larger.
	toNode := forward(t, in.Bytes(), opts(700), opts(300))
	fromNode := runNode(t, node, 500, toNode)
	for _, f := range splitFrames(t, fromNode) {
		if len(f) > 500 {
			t.Fatalf("JavaScript wrote a frame of %d bytes, over 500", len(f))
		}
	}
	back := forward(t, fromNode, opts(500), opts(1500))
	for i, f := range splitFrames(t, back) {
		if len(f) > 1500 {
			t.Fatalf("frame %d is %d bytes, over 1500", i, len(f))
		}
	}
	equalMessages(t, readMessages(t, back, opts(1500)...), msgs)
}

// readAllWith reads all of rd using a buffer of size bytes.
func readAllWith(t *testing.T, rd io.Reader, size int) []byte {
	t.Helper()
	buf := make([]byte, size)
	var out []byte
	for {
		n, err := rd.Read(buf)
		out = append(out, buf[:n]...)
		if err == io.EOF { //nolint:errorlint
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestFragmentsReader checks that the Reader gives the message, whatever the
// frames it was read from, and whatever the size of the buffer that it is read with.
func TestFragmentsReader(t *testing.T) {
	r := rand.New(rand.NewPCG(31, 32)) //nolint:gosec
	for i := range 40 {
		text := longRunText(r, 1+r.IntN(3000))
		if i%3 == 0 {
			text = randomText(r, 1+r.IntN(300))
		}
		for _, size := range []uint32{100, 337, 5000} {
			opts := headerOpts(30, jsonmsgs.WithMaxSize(size))
			var in bytes.Buffer
			writeMessages(t, newWriter(&in, opts...), []message{{string(text), `{"to":1}`}})
			fr, err := newReader(in.Bytes(), opts...).ReadFragments(nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, bufSize := range []int{1, 7, 64, 1000, 1 << 20} {
				if got := readAllWith(t, fr.Reader(), bufSize); !bytes.Equal(got, text) {
					t.Fatalf("size %d, buffer %d: got %q, want %q", size, bufSize, got, text)
				}
			}
		}
	}
}

func TestFragmentsReaderBare(t *testing.T) {
	msg := `{"a":"b"}`
	fr, err := newReader(joinFrames(msg)).ReadFragments(nil)
	if err != nil || !fr.Bare() {
		t.Fatalf("got %v, bare %v", err, fr.Bare())
	}
	for _, bufSize := range []int{1, 4, 100} {
		if got := readAllWith(t, fr.Reader(), bufSize); string(got) != msg {
			t.Errorf("buffer %d: got %q, want %q", bufSize, got, msg)
		}
	}
}

func TestFragmentsReaderEmpty(t *testing.T) {
	rd := (&jsonmsgs.Fragments{}).Reader()
	if n, err := rd.Read(make([]byte, 10)); n != 0 || err != io.EOF { //nolint:errorlint
		t.Errorf("got %d, %v", n, err)
	}
	if n, err := rd.Read(nil); n != 0 || err != nil {
		t.Errorf("got %d, %v for an empty buffer", n, err)
	}
	// And at the end it stays the end.
	fr, err := newReader(joinFrames(`{"a":1}`)).ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	rd = fr.Reader()
	readAllWith(t, rd, 100)
	for range 3 {
		if n, err := rd.Read(make([]byte, 10)); n != 0 || err != io.EOF { //nolint:errorlint
			t.Errorf("got %d, %v", n, err)
		}
	}
}

// TestFragmentsReaderDecodes has the message decoded from the Reader, by the
// decoders of the standard library, and this package's.
func TestFragmentsReaderDecodes(t *testing.T) {
	type doc struct {
		Name  string            `json:"name"`
		Items []string          `json:"items"`
		Extra map[string]string `json:"extra"`
	}
	want := doc{Name: "é\"world\\", Items: nil, Extra: map[string]string{}}
	for i := range 400 {
		want.Items = append(want.Items, fmt.Sprintf("item %d \"quoted\"   世界", i))
		want.Extra[fmt.Sprint("k", i)] = strings.Repeat("v\n", i%7)
	}
	msg, err := stdjson.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	opts := headerOpts(30, jsonmsgs.WithMaxSize(500))
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{{string(msg), `{"to":1}`}})
	fr, err := newReader(in.Bytes(), opts...).ReadFragments(nil)
	if err != nil || fr.Frames() < 10 {
		t.Fatalf("got %v, %d frames", err, fr.Frames())
	}
	var got doc
	if err := stdjson.NewDecoder(fr.Reader()).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("decoded %v, want %v", got, want)
	}
	// And with a decoder that is given the tokens, as a router may.
	dec := jsontext.NewDecoder(fr.Reader())
	var tokens int
	for {
		if _, err := dec.ReadToken(); err != nil {
			if err != io.EOF { //nolint:errorlint
				t.Fatal(err)
			}
			break
		}
		tokens++
	}
	if tokens < 800 {
		t.Errorf("%d tokens", tokens)
	}
}

// TestFragmentsReaderIndependent checks that several Readers can read the same
// Fragments, and that they do not need to hold the message in memory.
func TestFragmentsReaderIndependent(t *testing.T) {
	text := strings.Repeat("a\"bé", 100_000)
	opts := headerOpts(30, jsonmsgs.WithMaxSize(2000))
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{{text, ""}})
	fr, err := newReader(in.Bytes(), opts...).ReadFragments(nil)
	if err != nil {
		t.Fatal(err)
	}
	a, b := fr.Reader(), fr.Reader()
	bufA, bufB := make([]byte, 33), make([]byte, 1000)
	var gotA, gotB []byte
	for doneA, doneB := false, false; !doneA || !doneB; {
		if !doneA {
			n, err := a.Read(bufA)
			gotA = append(gotA, bufA[:n]...)
			doneA = err == io.EOF //nolint:errorlint
		}
		if !doneB {
			n, err := b.Read(bufB)
			gotB = append(gotB, bufB[:n]...)
			doneB = err == io.EOF //nolint:errorlint
		}
	}
	if string(gotA) != text || string(gotB) != text {
		t.Fatal("the readers did not read the message")
	}

	// Read with a small buffer the memory is for a frame, not the message.
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	readAllWithDiscard(fr.Reader(), 64)
	runtime.ReadMemStats(&after)
	if got := after.TotalAlloc - before.TotalAlloc; got > 64<<10 {
		t.Errorf("allocated %d bytes to read a message of %d bytes in frames of 2000", got, len(text))
	}
}

func readAllWithDiscard(rd io.Reader, size int) {
	buf := make([]byte, size)
	for {
		if _, err := rd.Read(buf); err != nil {
			return
		}
	}
}
