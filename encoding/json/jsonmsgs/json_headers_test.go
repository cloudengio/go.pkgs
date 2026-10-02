// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

func headerOpts(maxHeader uint32, repeat bool, more ...jsonmsgs.Option) []jsonmsgs.Option {
	return append([]jsonmsgs.Option{
		jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(maxHeader), jsonmsgs.WithRepeatHeader(repeat),
	}, more...)
}

// writeWithHeader writes raw with header.
func writeWithHeader(m *jsonmsgs.Messager, raw, header string) error {
	enc := m.NewEncoder()
	// The raw message is put in the encoder's buffer as WriteRawForTests does.
	if err := jsonmsgs.WriteRawIntoEncoderForTests(enc, []byte(raw)); err != nil {
		return err
	}
	return m.WriteMessageWithHeader(enc, jsontext.Value(header))
}

func TestMinFragmentSizeWithHeader(t *testing.T) {
	for _, tc := range []struct{ maxFrag, maxHdr, want uint32 }{
		{100, 0, 26}, {100, 10, 44}, {100, 9, 42}, {100, 100, 135}, {1 << 20, 4096, 4096 + 30 - 7 + 1 + 4 + 6 + 6},
	} {
		if got := jsonmsgs.MinFragmentSizeWithHeader(tc.maxFrag, tc.maxHdr); got != tc.want {
			t.Errorf("MinFragmentSizeWithHeader(%d, %d) = %d, want %d", tc.maxFrag, tc.maxHdr, got, tc.want)
		}
	}
}

func TestHeaderGolden(t *testing.T) {
	const msg, hdr = `{"a":"b"}`, `{"r":1}`
	var buf bytes.Buffer
	nm := newWriter(&buf, headerOpts(7, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
	if err := writeWithHeader(nm, msg, hdr); err != nil {
		t.Fatal(err)
	}
	// Small enough for one frame but not bare: a header has nowhere to go.
	want := []string{`{"~":[0,9,7],"h":{"r":1},"p":"{\"a\":\"b\"}"}`}
	got := splitFrames(t, buf.Bytes())
	if len(got) != 1 || string(got[0]) != want[0] {
		t.Fatalf("got %q, want %q", got, want)
	}
	rd := newReader(buf.Bytes(), headerOpts(7, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
	dec, err := rd.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(dec.Header()) != hdr || string(jsonmsgs.DecoderBytesForTests(dec)) != msg {
		t.Errorf("got header %q, message %q", dec.Header(), jsonmsgs.DecoderBytesForTests(dec))
	}
	rd.ReleaseDecoder(dec)

}

func TestHeaderGoldenFragmented(t *testing.T) {
	const hdr = `{"r":1}`
	var buf bytes.Buffer
	// Once with the header on every fragment and once without.
	for _, repeat := range []bool{false, true} {
		buf.Reset()
		opts := headerOpts(7, repeat, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100), jsonmsgs.WithFragmentSize(44))
		if err := writeWithHeader(newWriter(&buf, opts...), `{"key":"abcdefghijklmnopqrstuvwxyz"}`, hdr); err != nil {
			t.Fatal(err)
		}
		wantFrames := []string{
			`{"~":[0,36,7],"h":{"r":1},"p":"{\"key\":\""}`,
			`{"~":[1],"p":"abcdefghijklmnopqrstuvwxyz\""}`,
			`{"~":[2],"p":"}"}`,
		}
		if repeat {
			wantFrames = []string{
				`{"~":[0,36,7],"h":{"r":1},"p":"{\"key\":\""}`,
				`{"~":[1,7],"h":{"r":1},"p":"abcdefghijklmn"}`,
				`{"~":[2,7],"h":{"r":1},"p":"opqrstuvwxyz\""}`,
				`{"~":[3,7],"h":{"r":1},"p":"}"}`,
			}
		}
		frames := splitFrames(t, buf.Bytes())
		var got []string
		for _, f := range frames {
			got = append(got, string(f))
			if len(f) > 44 || !stdjson.Valid(f) {
				t.Errorf("repeat %v: bad frame %q", repeat, f)
			}
		}
		if strings.Join(got, "\n") != strings.Join(wantFrames, "\n") {
			t.Errorf("repeat %v:\ngot  %q\nwant %q", repeat, got, wantFrames)
		}
		dec, err := newReader(buf.Bytes(), opts...).ReadMessage()
		if err != nil || string(dec.Header()) != hdr || string(jsonmsgs.DecoderBytesForTests(dec)) != `{"key":"abcdefghijklmnopqrstuvwxyz"}` {
			t.Errorf("repeat %v: read back %v", repeat, err)
		}
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8)) //nolint:gosec
	for _, repeat := range []bool{false, true} {
		for _, budget := range []int{int(jsonmsgs.MinFragmentSizeWithHeader(1<<20, 40)), 80, 200, 1000} {
			for i := range 80 {
				var msg []byte
				if i%2 == 0 {
					msg = randomText(r, 1+r.IntN(600))
				} else {
					msg = worstCaseText(1 + r.IntN(200))
				}
				hdr := fmt.Sprintf(`{"id":%d,"s":%q}`, r.IntN(1000), strings.Repeat("h", r.IntN(15)))
				checkHeaderRoundTrip(t, msg, hdr, budget, repeat)
			}
		}
	}
}

func checkHeaderRoundTrip(t *testing.T, msg []byte, hdr string, budget int, repeat bool) {
	t.Helper()
	opts := headerOpts(40, repeat, jsonmsgs.WithMaxFragmentedMessageSize(1<<20), jsonmsgs.WithMaxSize(uint32(budget)))
	var buf bytes.Buffer
	if err := writeWithHeader(newWriter(&buf, opts...), string(msg), hdr); err != nil {
		t.Fatalf("budget %d: %v", budget, err)
	}
	for j, f := range splitFrames(t, buf.Bytes()) {
		if len(f) > budget || !stdjson.Valid(f) {
			t.Fatalf("budget %d frame %d: %d bytes, valid %v", budget, j, len(f), stdjson.Valid(f))
		}
		// A payload could contain "h": but only escaped, ie. as \"h\":.
		if hasH := bytes.Contains(f, []byte(`"h":`)); j > 0 && hasH != repeat {
			t.Fatalf("budget %d frame %d: header present %v, repeat %v: %q", budget, j, hasH, repeat, f)
		}
	}
	dec, err := newReader(buf.Bytes(), opts...).ReadMessage()
	if err != nil {
		t.Fatalf("budget %d: %v", budget, err)
	}
	if string(dec.Header()) != hdr || !bytes.Equal(jsonmsgs.DecoderBytesForTests(dec), msg) {
		t.Fatalf("budget %d: header %q want %q", budget, dec.Header(), hdr)
	}
}

func TestHeaderWriteErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		opts   []jsonmsgs.Option
		header string
		want   error
	}{
		{"no fragmentation", []jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(10)}, `{}`, jsonmsgs.ErrInvalidFrame},
		{"headers not enabled", []jsonmsgs.Option{jsonmsgs.WithFragmentation(true)}, `{}`, jsonmsgs.ErrInvalidFrame},
		{"too large", headerOpts(5, false), `{"a":1}`, jsonmsgs.ErrMessageTooLarge},
		{"empty", headerOpts(5, false), ``, jsonmsgs.ErrInvalidFrame},
		{"not JSON", headerOpts(50, false), `{a:1}`, jsonmsgs.ErrInvalidFrame},
		{"two values", headerOpts(50, false), `1 2`, jsonmsgs.ErrInvalidFrame},
		{"duplicate names", headerOpts(50, false), `{"a":1,"a":2}`, jsonmsgs.ErrInvalidFrame},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			nm := newWriter(&buf, tc.opts...)
			if err := writeWithHeader(nm, `{"a":1}`, tc.header); !errors.Is(err, tc.want) || buf.Len() != 0 {
				t.Errorf("got %v, wrote %d bytes, want %v", err, buf.Len(), tc.want)
			}
			// Nothing was written, and so a later message is not affected.
			if err := jsonmsgs.WriteRawForTests(nm, []byte(`{}`)); err != nil && !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
				t.Errorf("write after rejected header: %v", err)
			}
		})
	}
}

func TestNewMessagerHeaderPanics(t *testing.T) {
	for name, opts := range map[string][]jsonmsgs.Option{
		"fragment size":   headerOpts(100, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(130)),
		"too large":       headerOpts(1<<31, false),
		"fragment 4G max": headerOpts(10, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(30)),
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected a panic", name)
				}
			}()
			jsonmsgs.NewMessager(nil, nil, opts...)
		}()
	}
	jsonmsgs.NewMessager(nil, nil, headerOpts(100, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(135))...)
}

var headerRejectCases = []struct {
	name   string
	frames []string
}{
	{"header not enabled", []string{`{"~":[0,3,2],"h":{},"p":"abc"}`}},
	{"zero length", []string{`{"~":[0,3,0],"h":,"p":"abc"}`}},
	{"leading zero length", []string{`{"~":[0,3,02],"h":{},"p":"abc"}`}},
	{"length over limit", []string{`{"~":[0,3,17],"h":{"a":"bbbbbbbbb"},"p":"abc"}`}},
	{"length too short", []string{`{"~":[0,3,1],"h":{},"p":"abc"}`}},
	{"length too long", []string{`{"~":[0,3,3],"h":{},"p":"abc"}`}},
	{"length past the end", []string{`{"~":[0,3,16],"h":{},"p":"abc"}`}},
	{"header not JSON", []string{`{"~":[0,3,3],"h":{a},"p":"abc"}`}},
	{"header two values", []string{`{"~":[0,3,3],"h":1 2,"p":"abc"}`}},
	{"header smuggles member", []string{`{"~":[0,3,13],"h":1,"x":2,"p":2,"p":"abc"}`}},
	{"missing h key", []string{`{"~":[0,3,2],"x":{},"p":"abc"}`}},
	{"no header after length", []string{`{"~":[0,3,2],"p":"abc"}`}},
	{"whitespace", []string{`{"~":[0,3,2],"h": {},"p":"abc"}`}},
	{"header differs", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,2],"h":[]  ,"p":"abc"}`}},
	{"header differs, same length", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,2],"h":[],"p":"abc"}`}},
	{"header only on later", []string{`{"~":[0,6],"p":"abc"}`, `{"~":[1,2],"h":{},"p":"abc"}`}},
	{"total with header on seq 1", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,3,2],"h":{},"p":"abc"}`}},
}

func TestHeaderReaderRejects(t *testing.T) {
	for _, tc := range headerRejectCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := headerOpts(16, false, jsonmsgs.WithMaxFragmentedMessageSize(1000))
			if tc.name == "header not enabled" {
				opts = []jsonmsgs.Option{jsonmsgs.WithFragmentation(true)}
			}
			for _, viaFragment := range []bool{false, true} {
				rd := newReader(joinFrames(tc.frames...), opts...)
				var err error
				for err == nil {
					if viaFragment {
						_, err = rd.ReadFragment()
					} else {
						_, err = rd.ReadMessage()
					}
				}
				if !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
					t.Errorf("fragment %v: got %v, want ErrInvalidFrame", viaFragment, err)
				}
			}
		})
	}
}

// relay forwards every frame read from stream, with the options given, to a
// buffer using ReadFragment and WriteFragment, and returns what was written.
func relay(t testing.TB, stream []byte, ropts, wopts []jsonmsgs.Option) []byte {
	t.Helper()
	var out bytes.Buffer
	rd, wr := newReader(stream, ropts...), newWriter(&out, wopts...)
	for {
		f, err := rd.ReadFragment()
		if err == io.EOF { //nolint:errorlint
			return out.Bytes()
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragment(f); err != nil {
			t.Fatal(err)
		}
	}
}

func TestForwardFragments(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10)) //nolint:gosec
	for _, repeat := range []bool{false, true} {
		opts := headerOpts(30, repeat, jsonmsgs.WithMaxSize(120))
		var in bytes.Buffer
		nm := newWriter(&in, opts...)
		var wantHdr []string
		var msgs [][]byte
		for i := range 40 {
			var err error
			msg := randomText(r, 1+r.IntN(500))
			if i%4 == 0 {
				msg = []byte(`{"a":1}`)
			}
			hdr := ""
			if i%3 != 0 {
				hdr = fmt.Sprintf(`{"to":%d}`, i)
				err = writeWithHeader(nm, string(msg), hdr)
			} else {
				err = jsonmsgs.WriteRawForTests(nm, msg)
			}
			if err != nil {
				t.Fatal(err)
			}
			wantHdr = append(wantHdr, hdr)
			msgs = append(msgs, msg)
		}
		// What is forwarded is the same, byte for byte.
		if out := relay(t, in.Bytes(), opts, opts); !bytes.Equal(out, in.Bytes()) {
			t.Fatalf("repeat %v: forwarded stream differs", repeat)
		}
		checkForwardedFrames(t, in.Bytes(), opts, msgs, wantHdr)
	}
}

// checkForwardedFrames checks the frames that ReadFragment returns for stream.
func checkForwardedFrames(t *testing.T, stream []byte, opts []jsonmsgs.Option, msgs [][]byte, wantHdr []string) {
	t.Helper()
	rd := newReader(stream, opts...)
	for i := range msgs {
		var got []byte
		for {
			f, err := rd.ReadFragment()
			if err != nil {
				t.Fatal(err)
			}
			if f.Seq == 0 && string(f.Header) != wantHdr[i] {
				t.Fatalf("message %d: header %q, want %q", i, f.Header, wantHdr[i])
			}
			if f.Seq == 0 && f.Total != uint64(len(msgs[i])) {
				t.Fatalf("message %d: total %d, want %d", i, f.Total, len(msgs[i]))
			}
			if f.Seq != 0 && f.Total != 0 {
				t.Fatalf("message %d: total set on fragment %d", i, f.Seq)
			}
			got = append(got, unescapedPayload(t, f)...)
			if f.Last {
				break
			}
		}
		if !bytes.Equal(got, msgs[i]) {
			t.Fatalf("message %d: got %q, want %q", i, got, msgs[i])
		}
	}
}

// unescapedPayload returns the message bytes that f carries, using encoding/json.
func unescapedPayload(t *testing.T, f *jsonmsgs.Fragment) []byte {
	t.Helper()
	if f.Bare {
		return f.Payload
	}
	var s string
	if err := stdjson.Unmarshal([]byte(`"`+string(f.Payload)+`"`), &s); err != nil || len(s) != f.Len {
		t.Fatalf("fragment %d: payload %q len %d: %v", f.Seq, f.Payload, f.Len, err)
	}
	return []byte(s)
}

// TestForwardAddsHeader checks that headers can be added on forwarding by a
// sender that repeats them, which does not require that the receiver did.
func TestForwardAddsHeader(t *testing.T) {
	in := headerOpts(10, false, jsonmsgs.WithMaxSize(60), jsonmsgs.WithMaxFragmentedMessageSize(1000))
	var stream bytes.Buffer
	if err := writeWithHeader(newWriter(&stream, in...), strings.Repeat("x", 200), `{"to":1}`); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	wr := newWriter(&out, headerOpts(10, true, jsonmsgs.WithMaxSize(80), jsonmsgs.WithMaxFragmentedMessageSize(1000))...)
	rd := newReader(stream.Bytes(), headerOpts(10, false, jsonmsgs.WithMaxSize(80), jsonmsgs.WithMaxFragmentedMessageSize(1000))...)
	var hdr []byte
	for {
		f, err := rd.ReadFragment()
		if err != nil {
			t.Fatal(err)
		}
		if f.Seq == 0 {
			hdr = bytes.Clone(f.Header)
		}
		f.Header = hdr
		if err := wr.WriteFragment(f); err != nil {
			t.Fatal(err)
		}
		if f.Last {
			break
		}
	}
	for i, f := range splitFrames(t, out.Bytes()) {
		if !bytes.Contains(f, []byte(`"h":{"to":1}`)) {
			t.Errorf("frame %d has no header: %q", i, f)
		}
	}
	dec, err := newReader(out.Bytes(), headerOpts(10, false, jsonmsgs.WithMaxSize(80), jsonmsgs.WithMaxFragmentedMessageSize(1000))...).ReadMessage()
	if err != nil || string(jsonmsgs.DecoderBytesForTests(dec)) != strings.Repeat("x", 200) || string(dec.Header()) != `{"to":1}` {
		t.Errorf("read back: %v", err)
	}
}

// TestForwardDoesNotBuffer checks that a message of 8 MiB is forwarded with
// memory that does not grow with the size of the message.
func TestForwardDoesNotBuffer(t *testing.T) {
	const size = 8 << 20
	opts := headerOpts(30, false, jsonmsgs.WithMaxSize(64<<10), jsonmsgs.WithMaxFragmentedMessageSize(size))
	var in bytes.Buffer
	msg := []byte(`{"a":"` + strings.Repeat(`xyz"é`, size/8) + `"}`)
	if err := writeWithHeader(newWriter(&in, opts...), string(msg), `{"to":1}`); err != nil {
		t.Fatal(err)
	}
	stream := in.Bytes()
	rd, wr := newReader(stream, opts...), newWriter(io.Discard, opts...)
	// Warm up, so that what is measured is what the steady state needs.
	for {
		f, err := rd.ReadFragment()
		if err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragment(f); err != nil {
			t.Fatal(err)
		}
		if f.Last {
			break
		}
	}
	rd = newReader(stream, opts...)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	for done := false; !done; {
		f, err := rd.ReadFragment()
		if err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragment(f); err != nil {
			t.Fatal(err)
		}
		done = f.Last
	}
	runtime.ReadMemStats(&after)
	if got := after.TotalAlloc - before.TotalAlloc; got > 512<<10 {
		t.Errorf("allocated %d bytes forwarding a message of %d bytes", got, len(msg))
	}
}

const (
	forwardFirst  = `{"~":[0,6],"p":"abc"}`
	forwardSecond = `{"~":[1],"p":"def"}`
)

func TestForwardRules(t *testing.T) {
	t.Run("ReadMessage mid message", testForwardReadmessageMidMessage)
	t.Run("WriteMessage mid message", testForwardWritemessageMidMessage)
	t.Run("bad sequences", testForwardBadSequences)
	t.Run("mid message error poisons", testForwardMidMessageErrorPoisons)
	t.Run("error between messages does not poison", testForwardErrorBetweenMessagesDoesNotPoison)
	t.Run("failed first fragment does not leave a message open", testForwardFailedFirstFragmentDoesNotLeaveAMessageOpen)
	t.Run("nil and no fragmentation", testForwardNilAndNoFragmentation)
}

func testForwardReadmessageMidMessage(t *testing.T) {
	opts := headerOpts(30, false)
	rd := newReader(joinFrames(forwardFirst, forwardSecond, `{"a":1}`), opts...)
	if _, err := rd.ReadFragment(); err != nil {
		t.Fatal(err)
	}
	if _, err := rd.ReadMessage(); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Fatalf("got %v, want ErrInvalidFrame", err)
	}
	// A mistake by the caller, and not of the stream: carry on.
	if f, err := rd.ReadFragment(); err != nil || !f.Last {
		t.Fatalf("got %v, %v", f, err)
	}
	if dec, err := rd.ReadMessage(); err != nil || string(jsonmsgs.DecoderBytesForTests(dec)) != `{"a":1}` {
		t.Fatalf("got %v", err)
	}
}

func testForwardWritemessageMidMessage(t *testing.T) {
	opts := headerOpts(30, false)
	var out bytes.Buffer
	wr := newWriter(&out, opts...)
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 0, Total: 6, Payload: []byte("abc")}); err != nil {
		t.Fatal(err)
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Fatalf("got %v, want ErrInvalidFrame", err)
	}
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 1, Payload: []byte("def")}); err != nil {
		t.Fatal(err)
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err != nil {
		t.Fatalf("after the message: %v", err)
	}
	dec, err := newReader(out.Bytes(), opts...).ReadMessage()
	if err != nil || string(jsonmsgs.DecoderBytesForTests(dec)) != "abcdef" {
		t.Fatalf("got %v", err)
	}
}

func testForwardBadSequences(t *testing.T) {
	for name, frags := range map[string][]*jsonmsgs.Fragment{
		"starts at 1":     {{Seq: 1, Payload: []byte("a")}},
		"skips":           {{Seq: 0, Total: 6, Payload: []byte("abc")}, {Seq: 2, Payload: []byte("def")}},
		"overruns":        {{Seq: 0, Total: 2, Payload: []byte("abc")}},
		"bare mid":        {{Seq: 0, Total: 6, Payload: []byte("abc")}, {Bare: true, Payload: []byte(`{}`)}},
		"too large":       {{Seq: 0, Total: 1 << 30, Payload: []byte("abc")}},
		"bad payload":     {{Seq: 0, Total: 6, Payload: []byte(`a"b`)}},
		"empty payload":   {{Seq: 0, Total: 6}},
		"bad header":      {{Seq: 0, Total: 6, Header: []byte(`{`), Payload: []byte("abc")}},
		"big header":      {{Seq: 0, Total: 6, Header: []byte(`"` + strings.Repeat("h", 40) + `"`), Payload: []byte("abc")}},
		"bare reserved":   {{Bare: true, Payload: []byte(`{"~":[1]}`)}},
		"bare empty":      {{Bare: true}},
		"changed header":  {{Seq: 0, Total: 6, Header: []byte(`1`), Payload: []byte("abc")}, {Seq: 1, Header: []byte(`2`), Payload: []byte("def")}},
		"frame too large": {{Seq: 0, Total: 6000, Payload: bytes.Repeat([]byte("a"), 2000)}},
	} {
		var out bytes.Buffer
		wr := newWriter(&out, headerOpts(30, false, jsonmsgs.WithMaxSize(1000))...)
		var err error
		for _, f := range frags {
			if err = wr.WriteFragment(f); err != nil {
				break
			}
		}
		if !errors.Is(err, jsonmsgs.ErrInvalidFrame) && !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
			t.Errorf("%s: got %v", name, err)
		}
		// Nothing that is not valid is written.
		for _, f := range splitFrames(t, out.Bytes()) {
			if !stdjson.Valid(f) {
				t.Errorf("%s: wrote %q", name, f)
			}
		}
	}
}

func testForwardMidMessageErrorPoisons(t *testing.T) {
	opts := headerOpts(30, false)
	wr := newWriter(io.Discard, opts...)
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 0, Total: 6, Payload: []byte("abc")}); err != nil {
		t.Fatal(err)
	}
	err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 5, Payload: []byte("def")})
	if !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Fatalf("got %v", err)
	}
	if err2 := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err2 != err { //nolint:errorlint
		t.Errorf("got %v, want %v", err2, err)
	}
}

func testForwardErrorBetweenMessagesDoesNotPoison(t *testing.T) {
	opts := headerOpts(30, false)
	wr := newWriter(io.Discard, opts...)
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 3, Payload: []byte("def")}); err == nil {
		t.Fatal("expected an error")
	}
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 0, Total: 3, Payload: []byte("def")}); err != nil {
		t.Fatal(err)
	}
}

func testForwardFailedFirstFragmentDoesNotLeaveAMessageOpen(t *testing.T) {
	opts := headerOpts(30, false)
	wr := newWriter(io.Discard, opts...)
	// Passes the checks of the sequence but not of the size.
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 0, Total: 2, Payload: []byte("abc")}); err == nil {
		t.Fatal("expected an error")
	}
	if err := jsonmsgs.WriteRawForTests(wr, []byte(`{}`)); err != nil {
		t.Fatalf("a message after the rejected fragment: %v", err)
	}
}

func testForwardNilAndNoFragmentation(t *testing.T) {
	wr := newWriter(io.Discard)
	if err := wr.WriteFragment(nil); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Error(err)
	}
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Seq: 0, Total: 3, Payload: []byte("def")}); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Error(err)
	}
	// Whole messages can be forwarded without fragmentation.
	if err := wr.WriteFragment(&jsonmsgs.Fragment{Bare: true, Payload: []byte(`{}`)}); err != nil {
		t.Error(err)
	}
}

// FuzzForward checks that, whatever the stream, forwarding it with ReadFragment
// and WriteFragment is the same as reading it with ReadMessage: the same
// messages, headers and errors.
func FuzzForward(f *testing.F) {
	f.Add(joinFrames(`{"a":1}`))
	f.Add(joinFrames(`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,2],"h":{},"p":"def"}`))
	f.Add(joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[1],"p":"d\u00e9"}`, `{"a":1}`))
	f.Fuzz(func(t *testing.T, stream []byte) {
		opts := headerOpts(32, false, jsonmsgs.WithMaxSize(1<<16), jsonmsgs.WithMaxFragmentedMessageSize(1<<20))
		type result struct{ hdr, msg string }
		collect := func(rd *jsonmsgs.Messager) (rs []result, failed bool) {
			for {
				dec, err := rd.ReadMessage()
				if err != nil {
					return rs, err != io.EOF //nolint:errorlint
				}
				rs = append(rs, result{string(dec.Header()), string(jsonmsgs.DecoderBytesForTests(dec))})
				rd.ReleaseDecoder(dec)
			}
		}
		want, wantFailed := collect(newReader(stream, opts...))

		var out bytes.Buffer
		rd, wr := newReader(stream, opts...), newWriter(&out, opts...)
		var failed bool
		for {
			fr, err := rd.ReadFragment()
			if err != nil {
				failed = err != io.EOF //nolint:errorlint
				break
			}
			if err := wr.WriteFragment(fr); err != nil {
				t.Fatalf("WriteFragment of a frame that was read: %v", err)
			}
		}
		if failed != wantFailed {
			t.Fatalf("ReadFragment failed %v, ReadMessage %v", failed, wantFailed)
		}
		got, _ := collect(newReader(out.Bytes(), opts...))
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

func TestHeaderMessageTooLarge(t *testing.T) {
	var buf bytes.Buffer
	nm := newWriter(&buf, headerOpts(10, false, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
	if err := writeWithHeader(nm, strings.Repeat("a", 101), `{}`); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || buf.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes", err, buf.Len())
	}
}

// TestHeaderDoesNotLeak checks that the header of one message is not that of
// the next, which is read with a decoder that is reused.
func TestHeaderDoesNotLeak(t *testing.T) {
	opts := headerOpts(10, false)
	var buf bytes.Buffer
	nm := newWriter(&buf, opts...)
	for range 5 {
		if err := writeWithHeader(nm, `{"a":1}`, `{"to":1}`); err != nil {
			t.Fatal(err)
		}
		if err := jsonmsgs.WriteRawForTests(nm, []byte(`{"b":2}`)); err != nil {
			t.Fatal(err)
		}
	}
	rd := newReader(buf.Bytes(), opts...)
	for i := range 10 {
		dec, err := rd.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(dec.Header()), []string{`{"to":1}`, ""}[i%2]; got != want {
			t.Errorf("message %d: header %q, want %q", i, got, want)
		}
		rd.ReleaseDecoder(dec)
	}
}
