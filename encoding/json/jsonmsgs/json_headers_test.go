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
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

func headerOpts(maxHeader uint32, more ...jsonmsgs.Option) []jsonmsgs.Option {
	return append([]jsonmsgs.Option{
		jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxHeaderSize(maxHeader),
	}, more...)
}

// writeWithHeader writes raw with header.
func writeWithHeader(m *jsonmsgs.Writer, raw, header string) error {
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
		{100, math.MaxUint32, math.MaxUint32}, {100, math.MaxUint32 - 10, math.MaxUint32},
	} {
		if got := jsonmsgs.MinFragmentSizeWithHeader(tc.maxFrag, tc.maxHdr); got != tc.want {
			t.Errorf("MinFragmentSizeWithHeader(%d, %d) = %d, want %d", tc.maxFrag, tc.maxHdr, got, tc.want)
		}
	}
}

func TestHeaderGolden(t *testing.T) {
	const msg, hdr = `{"a":"b"}`, `{"r":1}`
	var buf bytes.Buffer
	nm := newWriter(&buf, headerOpts(7, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
	if err := writeWithHeader(nm, msg, hdr); err != nil {
		t.Fatal(err)
	}
	// Small enough for one frame but not bare: a header has nowhere to go.
	want := []string{`{"~":[0,9,7],"h":{"r":1},"p":"{\"a\":\"b\"}"}`}
	got := splitFrames(t, buf.Bytes())
	if len(got) != 1 || string(got[0]) != want[0] {
		t.Fatalf("got %q, want %q", got, want)
	}
	rd := newReader(buf.Bytes(), headerOpts(7, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
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
	opts := headerOpts(7, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100), jsonmsgs.WithFragmentSize(44))
	if err := writeWithHeader(newWriter(&buf, opts...), `{"key":"abcdefghijklmnopqrstuvwxyz"}`, hdr); err != nil {
		t.Fatal(err)
	}
	// The user header is on the first frame only.
	want := []string{
		`{"~":[0,36,7],"h":{"r":1},"p":"{\"key\":\""}`,
		`{"~":[1],"p":"abcdefghijklmnopqrstuvwxyz\""}`,
		`{"~":[2],"p":"}"}`,
	}
	var got []string
	for _, f := range splitFrames(t, buf.Bytes()) {
		got = append(got, string(f))
		if len(f) > 44 || !stdjson.Valid(f) {
			t.Errorf("bad frame %q", f)
		}
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("\ngot  %q\nwant %q", got, want)
	}
	dec, err := newReader(buf.Bytes(), opts...).ReadMessage()
	if err != nil || string(dec.Header()) != hdr || string(jsonmsgs.DecoderBytesForTests(dec)) != `{"key":"abcdefghijklmnopqrstuvwxyz"}` {
		t.Errorf("read back %v", err)
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8)) //nolint:gosec
	for _, budget := range []int{int(jsonmsgs.MinFragmentSizeWithHeader(1<<20, 40)), 80, 200, 1000} {
		for i := range 80 {
			var msg []byte
			if i%2 == 0 {
				msg = randomText(r, 1+r.IntN(600))
			} else {
				msg = worstCaseText(1 + r.IntN(200))
			}
			hdr := fmt.Sprintf(`{"id":%d,"s":%q}`, r.IntN(1000), strings.Repeat("h", r.IntN(15)))
			checkHeaderRoundTrip(t, msg, hdr, budget)
		}
	}
}

func checkHeaderRoundTrip(t *testing.T, msg []byte, hdr string, budget int) {
	t.Helper()
	opts := headerOpts(40, jsonmsgs.WithMaxFragmentedMessageSize(1<<20), jsonmsgs.WithMaxSize(uint32(budget)))
	var buf bytes.Buffer
	if err := writeWithHeader(newWriter(&buf, opts...), string(msg), hdr); err != nil {
		t.Fatalf("budget %d: %v", budget, err)
	}
	for j, f := range splitFrames(t, buf.Bytes()) {
		if len(f) > budget || !stdjson.Valid(f) {
			t.Fatalf("budget %d frame %d: %d bytes, valid %v", budget, j, len(f), stdjson.Valid(f))
		}
		// A payload could contain "h": but only escaped, ie. as \"h\":.
		if hasH := bytes.Contains(f, []byte(`"h":`)); (j == 0) != hasH {
			t.Fatalf("budget %d frame %d: header present is %v: %q", budget, j, hasH, f)
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
		{"headers not enabled", []jsonmsgs.Option{jsonmsgs.WithFragmentation(true)}, `{}`, jsonmsgs.ErrInvalidFrame},
		{"too large", headerOpts(5), `{"a":1}`, jsonmsgs.ErrMessageTooLarge},
		{"empty", headerOpts(5), ``, jsonmsgs.ErrInvalidFrame},
		{"not JSON", headerOpts(50), `{a:1}`, jsonmsgs.ErrInvalidFrame},
		{"two values", headerOpts(50), `1 2`, jsonmsgs.ErrInvalidFrame},
		{"duplicate names", headerOpts(50), `{"a":1,"a":2}`, jsonmsgs.ErrInvalidFrame},
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
		"fragment size":   headerOpts(100, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(130)),
		"too large":       headerOpts(1 << 31),
		"fragment 4G max": headerOpts(10, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(30)),
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
	jsonmsgs.NewMessager(nil, nil, headerOpts(100, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(135))...)
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
	{"header on a later fragment", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,2],"h":[]  ,"p":"abc"}`}},
	{"header differs, same length", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,2],"h":[],"p":"abc"}`}},
	{"header only on later", []string{`{"~":[0,6],"p":"abc"}`, `{"~":[1,2],"h":{},"p":"abc"}`}},
	{"total with header on seq 1", []string{`{"~":[0,6,2],"h":{},"p":"abc"}`, `{"~":[1,3,2],"h":{},"p":"abc"}`}},
}

func TestHeaderReaderRejects(t *testing.T) {
	for _, tc := range headerRejectCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := headerOpts(16, jsonmsgs.WithMaxFragmentedMessageSize(1000))
			if tc.name == "header not enabled" {
				opts = []jsonmsgs.Option{jsonmsgs.WithFragmentation(true)}
			}
			for _, viaFragments := range []bool{false, true} {
				rd := newReader(joinFrames(tc.frames...), opts...)
				var err error
				for err == nil {
					if viaFragments {
						_, err = rd.ReadFragments(nil)
					} else {
						_, err = rd.ReadMessage()
					}
				}
				if !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
					t.Errorf("fragments %v: got %v, want ErrInvalidFrame", viaFragments, err)
				}
			}
		})
	}
}

func TestHeaderMessageTooLarge(t *testing.T) {
	var buf bytes.Buffer
	nm := newWriter(&buf, headerOpts(10, jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithMaxSize(100))...)
	if err := writeWithHeader(nm, strings.Repeat("a", 101), `{}`); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || buf.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes", err, buf.Len())
	}
}

// TestHeaderDoesNotLeak checks that the header of one message is not that of
// the next, which is read with a decoder that is reused.
func TestHeaderDoesNotLeak(t *testing.T) {
	opts := headerOpts(10)
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

// TestHeaderWithoutFragmentation checks that headers are independent of
// fragmentation: the message is in a single envelope, that has to fit in a frame.
func TestHeaderWithoutFragmentation(t *testing.T) {
	opts := []jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(100)}
	var buf bytes.Buffer
	nm := newWriter(&buf, opts...)
	if err := writeWithHeader(nm, `{"a":"b"}`, `{"r":1}`); err != nil {
		t.Fatal(err)
	}
	if got, want := string(splitFrames(t, buf.Bytes())[0]), `{"~":[0,9,7],"h":{"r":1},"p":"{\"a\":\"b\"}"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	dec, err := newReader(buf.Bytes(), opts...).ReadMessage()
	if err != nil || string(dec.Header()) != `{"r":1}` || string(jsonmsgs.DecoderBytesForTests(dec)) != `{"a":"b"}` {
		t.Errorf("read back: %v", err)
	}
	// Forwarded.
	if out := forward(t, buf.Bytes(), opts, opts); !bytes.Equal(out, buf.Bytes()) {
		t.Errorf("forwarded %q", out)
	}
	// A message that does not fit in an envelope is too large, and is not split.
	buf.Reset()
	if err := writeWithHeader(nm, `"`+strings.Repeat("a", 90)+`"`, `{"r":1}`); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || buf.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes", err, buf.Len())
	}
	// A header is still limited, and a reader that does not enable them rejects it.
	if err := writeWithHeader(nm, `{}`, `{"a":"bbbbbbbbbbbbbbbbbb"}`); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("got %v", err)
	}
	if err := writeWithHeader(nm, `{}`, `{"r":1}`); err != nil {
		t.Fatal(err)
	}
	if _, err := newReader(buf.Bytes()).ReadMessage(); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Errorf("got %v, want ErrInvalidFrame", err)
	}
}

// TestFragmentSizeIgnoredWithoutFragmentation checks that, without
// fragmentation, an envelope is limited by the maximum size, as a bare frame is.
func TestFragmentSizeIgnoredWithoutFragmentation(t *testing.T) {
	var buf bytes.Buffer
	nm := newWriter(&buf, jsonmsgs.WithMaxHeaderSize(16), jsonmsgs.WithMaxSize(200), jsonmsgs.WithFragmentSize(60))
	if err := writeWithHeader(nm, `"`+strings.Repeat("a", 100)+`"`, `{"r":1}`); err != nil {
		t.Fatal(err)
	}
	if n := len(splitFrames(t, buf.Bytes())); n != 1 {
		t.Errorf("%d frames", n)
	}
}
