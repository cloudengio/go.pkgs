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
	"os/exec"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

type reply struct {
	Status string            `json:"status"`
	Items  []int             `json:"items,omitempty"`
	Extra  map[string]string `json:"extra,omitempty"`
}

type route struct {
	To  int      `json:"to"`
	Via []string `json:"via,omitempty"`
}

func TestFragmentsSet(t *testing.T) {
	msg, hdr := reply{Status: "ok", Items: []int{1, 2, 3}}, route{To: 7, Via: []string{"a", "b"}}
	wantMsg, _ := stdjson.Marshal(msg)
	wantHdr, _ := stdjson.Marshal(hdr)

	fr, err := jsonmsgs.NewFragments(hdr, msg)
	if err != nil {
		t.Fatal(err)
	}
	checkAccessors(t, fr, string(wantHdr), true, len(wantMsg), 1)
	if string(fr.Payload()) != string(wantMsg) {
		t.Errorf("payload %q, want %q", fr.Payload(), wantMsg)
	}
	if got := readAllWith(t, fr.Reader(), 5); !bytes.Equal(got, wantMsg) {
		t.Errorf("reader gave %q, want %q", got, wantMsg)
	}

	// Written to writers of different sizes, with the header in an envelope
	// even where the message would otherwise be a frame of its own.
	for _, size := range []uint32{80, 120, 1000} {
		opts := headerOpts(40, jsonmsgs.WithMaxFragmentedMessageSize(1000), jsonmsgs.WithMaxSize(size))
		var out bytes.Buffer
		if err := newWriter(&out, opts...).WriteFragments(fr); err != nil {
			t.Fatalf("size %d: %v", size, err)
		}
		for i, f := range splitFrames(t, out.Bytes()) {
			if !bytes.HasPrefix(f, []byte(`{"~":[`)) || len(f) > int(size) {
				t.Errorf("size %d: frame %d is %q", size, i, f)
			}
		}
		if n := len(splitFrames(t, out.Bytes())); (size == 80) != (n > 1) {
			t.Errorf("size %d: %d frames", size, n)
		}
		equalMessages(t, readMessages(t, out.Bytes(), opts...), []message{{string(wantMsg), string(wantHdr)}})
	}

	// A message with a header is a single envelope if fragmentation is not enabled.
	var single bytes.Buffer
	if err := newWriter(&single, jsonmsgs.WithMaxHeaderSize(40), jsonmsgs.WithMaxSize(1000)).WriteFragments(fr); err != nil {
		t.Fatal(err)
	}
	if n := len(splitFrames(t, single.Bytes())); n != 1 || !bytes.HasPrefix(splitFrames(t, single.Bytes())[0], []byte(`{"~":[0,`)) {
		t.Errorf("%d frames, %q", n, single.Bytes())
	}
	equalMessages(t, readMessages(t, single.Bytes(), jsonmsgs.WithMaxHeaderSize(40), jsonmsgs.WithMaxSize(1000)), []message{{string(wantMsg), string(wantHdr)}})

	// And to any number of writers.
	for range 2 {
		var out bytes.Buffer
		if err := newWriter(&out, headerOpts(40)...).WriteFragments(fr); err != nil {
			t.Fatal(err)
		}
	}
}

// TestFragmentsSetNoHeader checks that without a header the message is the frame
// that a browser receives and sends.
func TestFragmentsSetNoHeader(t *testing.T) {
	fr, err := jsonmsgs.NewFragments(nil, map[string]any{"a": 1, "b": []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := newWriter(&out).WriteFragments(fr); err != nil {
		t.Fatal(err)
	}
	want := `{"a":1,"b":["x"]}`
	if !bytes.Equal(out.Bytes(), joinFrames(want)) || fr.Header() != nil {
		t.Errorf("wrote %q, header %q, want %q", out.Bytes(), fr.Header(), joinFrames(want))
	}
}

func TestFragmentsSetTypes(t *testing.T) {
	for name, tc := range map[string]struct {
		message, header  any
		wantMsg, wantHdr string
	}{
		"string":         {"a \"quoted\" é", nil, `"a \"quoted\" é"`, ""},
		"number":         {42, 1.5, `42`, `1.5`},
		"bool":           {true, false, `true`, `false`},
		"slice":          {[]int{1, 2}, []string{"h"}, `[1,2]`, `["h"]`},
		"jsontext.Value": {jsontext.Value(`{ "a" : 1 }`), jsontext.Value(`{ "to" : 2 }`), `{"a":1}`, `{"to":2}`},
		"RawMessage":     {stdjson.RawMessage(`[ 1, 2 ]`), stdjson.RawMessage(`"x"`), `[1,2]`, `"x"`},
		"pointer":        {&reply{Status: "ok"}, &route{To: 1}, `{"status":"ok"}`, `{"to":1}`},
		"bytes":          {[]byte("hi"), nil, `"aGk="`, ""},
		"HTML":           {"a<b>&c", "<h>", `"a<b>&c"`, `"<h>"`},
		"empty string":   {"", nil, `""`, ""},
	} {
		t.Run(name, func(t *testing.T) {
			fr, err := jsonmsgs.NewFragments(tc.header, tc.message)
			if err != nil {
				t.Fatal(err)
			}
			if string(fr.Payload()) != tc.wantMsg || string(fr.Header()) != tc.wantHdr {
				t.Errorf("payload %q header %q, want %q and %q", fr.Payload(), fr.Header(), tc.wantMsg, tc.wantHdr)
			}
		})
	}
}

func TestFragmentsSetErrors(t *testing.T) {
	var unsupported *stdjson.UnsupportedTypeError
	for name, tc := range map[string]struct {
		message, header any
		as              bool
	}{
		"no message":              {nil, route{}, false},
		"message not marshalable": {make(chan int), nil, true},
		"header not marshalable":  {"x", func() {}, true},
		"invalid raw message":     {stdjson.RawMessage(`{`), nil, false},
		"invalid raw header":      {"x", stdjson.RawMessage(`{`), false},
		"invalid UTF-8":           {stdjson.RawMessage("\"\xff\""), nil, false},
		"invalid UTF-8 header":    {"x", stdjson.RawMessage("\"\xff\""), false},
	} {
		t.Run(name, func(t *testing.T) {
			// Empties what was there.
			fr, err := jsonmsgs.NewFragments(route{To: 1}, "before")
			if err != nil {
				t.Fatal(err)
			}
			err = fr.Set(tc.header, tc.message)
			if !errors.Is(err, jsonmsgs.ErrInvalidFrame) || errors.Is(err, jsonmsgs.ErrPermanent) {
				t.Fatalf("got %v, want an ErrInvalidFrame that can be retried", err)
			}
			if tc.as && !errors.As(err, &unsupported) {
				t.Errorf("the error from marshalling is not wrapped: %v", err)
			}
			if fr.Len() != 0 || fr.Header() != nil || fr.Frames() != 0 || len(fr.Payload()) != 0 {
				t.Errorf("len %d, header %q, %d frames, %d bytes of payload were left", fr.Len(), fr.Header(), fr.Frames(), len(fr.Payload()))
			}
			if err := newWriter(io.Discard, headerOpts(40)...).WriteFragments(fr); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
				t.Errorf("an empty Fragments was written: %v", err)
			}
			if _, err := jsonmsgs.NewFragments(tc.header, tc.message); err == nil {
				t.Errorf("NewFragments did not fail")
			}
		})
	}
}

// TestFragmentsSetReused checks that Fragments that were read, or set, can be set
// again, with nothing of what they held left over.
func TestFragmentsSetReused(t *testing.T) {
	opts := headerOpts(40, jsonmsgs.WithMaxSize(100))
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{{strings.Repeat("a\"b", 200), `{"to":1,"via":["x","y","z"]}`}})
	fr, err := newReader(in.Bytes(), opts...).ReadFragments(nil)
	if err != nil || fr.Frames() < 3 {
		t.Fatalf("got %v, %d frames", err, fr.Frames())
	}
	if err := fr.Set(nil, "ok"); err != nil {
		t.Fatal(err)
	}
	checkAccessors(t, fr, "", true, 4, 1)
	var out bytes.Buffer
	if err := newWriter(&out, opts...).WriteFragments(fr); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), joinFrames(`"ok"`)) {
		t.Errorf("wrote %q", out.Bytes())
	}
	// And the other way, with a header where there was none.
	if err := fr.Set(route{To: 2}, "again"); err != nil {
		t.Fatal(err)
	}
	checkAccessors(t, fr, `{"to":2}`, true, 7, 1)
	if _, err := newReader(in.Bytes(), opts...).ReadFragments(fr); err != nil {
		t.Fatal(err)
	}
	if fr.Bare() || fr.Len() != 600 || string(fr.Header()) != `{"to":1,"via":["x","y","z"]}` {
		t.Errorf("bare %v, len %d, header %q", fr.Bare(), fr.Len(), fr.Header())
	}
}

// TestFragmentsSetWriterLimits checks that a Writer applies its limits to a message
// that is set, as it does to one that was read.
func TestFragmentsSetWriterLimits(t *testing.T) {
	fr, err := jsonmsgs.NewFragments(route{To: 1, Via: []string{"a"}}, strings.Repeat("a", 500))
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		opts []jsonmsgs.Option
		want error
	}{
		"header too large":    {headerOpts(4, jsonmsgs.WithMaxSize(100)), jsonmsgs.ErrMessageTooLarge},
		"headers not enabled": {[]jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(100)}, jsonmsgs.ErrInvalidFrame},
		"message too large":   {headerOpts(40, jsonmsgs.WithMaxSize(100), jsonmsgs.WithMaxFragmentedMessageSize(300)), jsonmsgs.ErrMessageTooLarge},
		"not in one frame":    {[]jsonmsgs.Option{jsonmsgs.WithMaxHeaderSize(40), jsonmsgs.WithMaxSize(300)}, jsonmsgs.ErrMessageTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			err := newWriter(&out, tc.opts...).WriteFragments(fr)
			if !errors.Is(err, tc.want) || errors.Is(err, jsonmsgs.ErrPermanent) || out.Len() != 0 {
				t.Fatalf("got %v, wrote %d bytes, want a %v that can be retried", err, out.Len(), tc.want)
			}
		})
	}
	// Without a header, a message that does not fit is too large unless it can be
	// fragmented.
	big, err := jsonmsgs.NewFragments(nil, strings.Repeat("a", 500))
	if err != nil {
		t.Fatal(err)
	}
	if err := newWriter(io.Discard, jsonmsgs.WithMaxSize(100)).WriteFragments(big); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("got %v", err)
	}
	var out bytes.Buffer
	if err := newWriter(&out, jsonmsgs.WithMaxSize(100), jsonmsgs.WithFragmentation(true)).WriteFragments(big); err != nil {
		t.Fatal(err)
	}
	equalMessages(t, readMessages(t, out.Bytes(), jsonmsgs.WithMaxSize(100), jsonmsgs.WithFragmentation(true)), []message{{`"` + strings.Repeat("a", 500) + `"`, ""}})
}

// TestFragmentsSetRouterReply is a router that replies to the messages that are
// addressed to it, and forwards the others.
func TestFragmentsSetRouterReply(t *testing.T) {
	opts := headerOpts(40, jsonmsgs.WithMaxSize(200))
	var in bytes.Buffer
	writeMessages(t, newWriter(&in, opts...), []message{
		{`{"op":"ping"}`, `{"to":0}`},               // for the router
		{strings.Repeat("x", 500), `{"to":1}`},      // to be forwarded
		{`{"op":"time"}`, `{"to":0,"reply":"abc"}`}, // for the router
	})
	rd, out := newReader(in.Bytes(), opts...), &bytes.Buffer{}
	wr := newWriter(out, opts...)
	var fr, response *jsonmsgs.Fragments
	for {
		var err error
		if fr, err = rd.ReadFragments(fr); err == io.EOF { //nolint:errorlint
			break
		} else if err != nil {
			t.Fatal(err)
		}
		var h route
		if err := stdjson.Unmarshal(fr.Header(), &h); err != nil {
			t.Fatal(err)
		}
		if h.To != 0 {
			if err := wr.WriteFragments(fr); err != nil {
				t.Fatal(err)
			}
			continue
		}
		var request struct{ Op string }
		if err := stdjson.NewDecoder(fr.Reader()).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if response == nil {
			response = &jsonmsgs.Fragments{}
		}
		if err := response.Set(route{To: 9}, reply{Status: request.Op + " ok"}); err != nil {
			t.Fatal(err)
		}
		if err := wr.WriteFragments(response); err != nil {
			t.Fatal(err)
		}
	}
	equalMessages(t, readMessages(t, out.Bytes(), opts...), []message{
		{`{"status":"ping ok"}`, `{"to":9}`},
		{strings.Repeat("x", 500), `{"to":1}`},
		{`{"status":"time ok"}`, `{"to":9}`},
	})
}

// setMessages are messages, as values, with the characters that are escaped in
// the JSON of the message, and again in the payload of an envelope.
func setMessages() []struct{ message, header any } {
	return []struct{ message, header any }{
		{"plain text", nil},
		{"quotes \" and \\ backslashes \\\" and \n\t\r control \u0001\u001f", route{To: 1}},
		{"unicode é 世界 😀     \x7f", route{To: 2, Via: []string{"é\"", " "}}},
		{map[string]any{"k\"ey": []any{"v\\", 1.5, nil, true}, "é": "\"", "z": strings500()}, route{To: 3}},
		{[]string{strings500(), strings500(), "\"\"\"\""}, nil},
		{stdjson.RawMessage(`{ "raw" : "literal é é  " }`), jsontext.Value(`{ "to" : 4 }`)},
		{12345.6789, true},
	}
}

func strings500() string {
	return strings.Repeat("ab\"c\\d\n é", 55)
}

// TestFragmentsSetEnvelopeMatchesWriter checks that a message that is set is
// written in the same frames, byte for byte, as WriteMessage writes the same
// JSON, since both escape and split it using the same code.
func TestFragmentsSetEnvelopeMatchesWriter(t *testing.T) {
	for i, tc := range setMessages() {
		fr, err := jsonmsgs.NewFragments(tc.header, tc.message)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		for _, size := range []uint32{100, 120, 200, 1000, 100_000} {
			opts := headerOpts(60, jsonmsgs.WithMaxFragmentedMessageSize(100_000), jsonmsgs.WithMaxSize(size))
			var set, direct bytes.Buffer
			if err := newWriter(&set, opts...).WriteFragments(fr); err != nil {
				t.Fatalf("%d, size %d: %v", i, size, err)
			}
			writeMessages(t, newWriter(&direct, opts...), []message{{string(fr.Payload()), string(fr.Header())}})
			if !bytes.Equal(set.Bytes(), direct.Bytes()) {
				t.Fatalf("%d, size %d: the frames differ:\nset    %q\ndirect %q", i, size, set.Bytes(), direct.Bytes())
			}
			equalMessages(t, readMessages(t, set.Bytes(), opts...), []message{{string(fr.Payload()), string(fr.Header())}})
		}
	}
}

// TestFragmentsSetNode has the JavaScript implementation, that is independent of
// this package, interpret messages that are set, in frames of many sizes, and
// writes them again for this package to read.
func TestFragmentsSetNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	var want []message
	for _, tc := range setMessages() {
		fr, err := jsonmsgs.NewFragments(tc.header, tc.message)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, message{string(fr.Payload()), string(fr.Header())})
	}
	for _, size := range []uint32{100, 150, 700, 100_000} {
		opts := headerOpts(60, jsonmsgs.WithMaxFragmentedMessageSize(100_000), jsonmsgs.WithMaxSize(size))
		var in bytes.Buffer
		wr := newWriter(&in, opts...)
		for _, tc := range setMessages() {
			fr, err := jsonmsgs.NewFragments(tc.header, tc.message)
			if err != nil {
				t.Fatal(err)
			}
			if err := wr.WriteFragments(fr); err != nil {
				t.Fatal(err)
			}
		}
		for _, jsBudget := range []int{int(size), 200, 100_000} {
			checkNodeFrames(t, runNode(t, node, jsBudget, in.Bytes()), jsBudget, want)
		}
	}
}

// checkNodeFrames checks the frames that the JavaScript wrote: that they are no
// larger than budget and give the messages.
func checkNodeFrames(t *testing.T, out []byte, budget int, want []message) {
	t.Helper()
	for i, f := range splitFrames(t, out) {
		if len(f) > budget {
			t.Fatalf("JavaScript wrote frame %d of %d bytes, over %d", i, len(f), budget)
		}
	}
	got := readMessages(t, out, headerOpts(60,
		jsonmsgs.WithMaxFragmentedMessageSize(100_000), jsonmsgs.WithMaxSize(uint32(budget)))...)
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	for i := range want {
		// The message is as it was, to the byte. The JavaScript serializes the
		// header again, which is the same JSON but is not necessarily the same
		// bytes, as it does not escape U+2028.
		if got[i].text != want[i].text || !sameJSON(t, got[i].header, want[i].header) {
			t.Fatalf("budget %d, message %d: got %q (header %q), want %q (header %q)",
				budget, i, got[i].text, got[i].header, want[i].text, want[i].header)
		}
	}
}

// sameJSON reports whether a and b, either of which may be empty, are the same JSON value.
func sameJSON(t *testing.T, a, b string) bool {
	t.Helper()
	if a == "" || b == "" {
		return a == b
	}
	var va, vb any
	if err := stdjson.Unmarshal([]byte(a), &va); err != nil {
		t.Fatal(err)
	}
	if err := stdjson.Unmarshal([]byte(b), &vb); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprint(va) == fmt.Sprint(vb)
}
