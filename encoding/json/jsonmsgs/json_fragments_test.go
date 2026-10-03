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
	"slices"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"cloudeng.io/encoding/json/jsonmsgs"
)

// splitFrames splits a stream into the bodies of its frames.
func splitFrames(t testing.TB, raw []byte) [][]byte {
	t.Helper()
	var out [][]byte
	for len(raw) > 0 {
		if len(raw) < 4 {
			t.Fatalf("truncated frame header")
		}
		n := int(binary.LittleEndian.Uint32(raw))
		if len(raw) < 4+n {
			t.Fatalf("truncated frame body")
		}
		out = append(out, raw[4:4+n])
		raw = raw[4+n:]
	}
	return out
}

func joinFrames(bodies ...string) []byte {
	var out []byte
	for _, b := range bodies {
		out = binary.LittleEndian.AppendUint32(out, uint32(len(b)))
		out = append(out, b...)
	}
	return out
}

func newWriter(w io.Writer, opts ...jsonmsgs.Option) *jsonmsgs.Writer {
	return jsonmsgs.NewWriter(w, opts...)
}

func newReader(stream []byte, opts ...jsonmsgs.Option) *jsonmsgs.Reader {
	return jsonmsgs.NewReader(io.NopCloser(bytes.NewReader(stream)), opts...)
}

// readRaw returns the next message, as bytes.
func readRaw(m *jsonmsgs.Reader) ([]byte, error) {
	dec, err := m.ReadMessage()
	if err != nil {
		return nil, err
	}
	defer m.ReleaseDecoder(dec)
	return bytes.Clone(jsonmsgs.DecoderBytesForTests(dec)), nil
}

func TestMinFragmentSize(t *testing.T) {
	for _, tc := range []struct{ max, want uint32 }{
		{1, 24}, {9, 24}, {10, 25}, {100, 26}, {1 << 20, 30},
		{16 << 20, 31}, {1<<32 - 1, 33},
	} {
		if got := jsonmsgs.MinFragmentSize(tc.max); got != tc.want {
			t.Errorf("MinFragmentSize(%d) = %d, want %d", tc.max, got, tc.want)
		}
	}
}

func TestBareFrameIsBrowserFraming(t *testing.T) {
	for _, frag := range []bool{false, true} {
		var buf bytes.Buffer
		nm := newWriter(&buf, jsonmsgs.WithFragmentation(frag))
		enc := nm.NewEncoder()
		if err := enc.WriteValue(jsontext.Value(`{"a":1}`)); err != nil {
			t.Fatal(err)
		}
		if err := nm.WriteMessage(enc); err != nil {
			t.Fatal(err)
		}
		if got, want := buf.Bytes(), []byte("\x07\x00\x00\x00{\"a\":1}"); !bytes.Equal(got, want) {
			t.Errorf("fragmentation %v: got %q, want %q", frag, got, want)
		}
	}
}

// TestFragmentGolden checks the exact bytes of a fragmented message, which
// also serve as the worked example in the package documentation: a budget of
// 30 bytes forces each of the escape sequences to be accounted for.
func TestFragmentGolden(t *testing.T) {
	const msg = `{"key":"abcdefghijklmnopqrstuvwxyz"}`
	var buf bytes.Buffer
	nm := newWriter(&buf, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(100), jsonmsgs.WithFragmentSize(30))
	if err := jsonmsgs.WriteRawForTests(nm, []byte(msg)); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`{"~":[0,36],"p":"{\"key\":\""}`,
		`{"~":[1],"p":"abcdefghijklmn"}`,
		`{"~":[2],"p":"opqrstuvwxyz\""}`,
		`{"~":[3],"p":"}"}`,
	}
	got := splitFrames(t, buf.Bytes())
	if len(got) != len(want) {
		t.Fatalf("got %d frames, want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if string(got[i]) != want[i] {
			t.Errorf("frame %d: got %s, want %s", i, got[i], want[i])
		}
	}
	rd, err := readRaw(newReader(buf.Bytes(), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(100)))
	if err != nil || string(rd) != msg {
		t.Errorf("got %q, %v, want %q", rd, err, msg)
	}
}

// randomText returns text with a mix of every kind of character that is
// treated differently when escaped.
func randomText(r *rand.Rand, n int) []byte {
	alphabet := []rune{'a', 'b', ' ', '{', '}', '"', '\\', '\n', '\t', '\r', '\b', '\f', 0, 1, 0x1f,
		0x7f, 0x80, 'é', 0x2028, 0x2029, '世', 0xFFFD, '😀', 0x10FFFF, '~', '/', '<', '&'}
	var b []byte
	for range n {
		b = utf8.AppendRune(b, alphabet[r.IntN(len(alphabet))])
	}
	return b
}

// worstCaseText is text that is as large as it can be once escaped.
func worstCaseText(n int) []byte {
	return bytes.Repeat([]byte{0x01}, n)
}

// checkFragments verifies the properties that the package documentation
// promises of the frames in raw, which carries msg: each is within budget, is
// valid JSON of the specified form and, other than the last, is as full as it
// can be.
func checkFragments(t *testing.T, raw []byte, msg []byte, budget int) {
	t.Helper()
	frames := splitFrames(t, raw)
	var joined []byte
	for i, f := range frames {
		if len(f) > budget {
			t.Fatalf("frame %d is %d bytes, over the budget of %d", i, len(f), budget)
		}
		if !stdjson.Valid(f) {
			t.Fatalf("frame %d is not valid JSON: %q", i, f)
		}
		var env struct {
			H []uint64 `json:"~"`
			P string   `json:"p"`
		}
		d := stdjson.NewDecoder(bytes.NewReader(f))
		d.DisallowUnknownFields()
		if err := d.Decode(&env); err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		wantH := 1
		if i == 0 {
			wantH = 2
		}
		if len(env.H) != wantH || env.H[0] != uint64(i) || (i == 0 && env.H[1] != uint64(len(msg))) {
			t.Fatalf("frame %d: bad header %v", i, env.H)
		}
		if env.P == "" {
			t.Fatalf("frame %d: empty payload", i)
		}
		// Maximal: a frame ends only when the next character does not fit,
		// and the widest, 4 bytes of UTF-8 or the 6 of an escape, does not
		// fit in at most 5 free bytes.
		if i < len(frames)-1 && len(f) < budget-(jsonmsgs.MaxEscapedLenForTests-1) {
			t.Fatalf("frame %d of %d is %d bytes, not full for a budget of %d", i, len(frames), len(f), budget)
		}
		joined = append(joined, env.P...)
	}
	if !bytes.Equal(joined, msg) {
		t.Fatalf("fragments carry %q, not %q", joined, msg)
	}
}

func TestFragmentProperties(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec
	const maxFrag = 1 << 20
	for _, budget := range []int{int(jsonmsgs.MinFragmentSize(maxFrag)), 31, 32, 37, 40, 64, 100, 257} {
		for i := range 60 {
			var msg []byte
			switch i % 3 {
			case 0:
				msg = randomText(r, 1+r.IntN(400))
			case 1:
				msg = worstCaseText(1 + r.IntN(100))
			default:
				msg = bytes.Repeat([]byte("é\"世😀"), 1+r.IntN(40))
			}
			var buf bytes.Buffer
			nm := newWriter(&buf, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(maxFrag), jsonmsgs.WithFragmentSize(uint32(budget)),
				jsonmsgs.WithMaxSize(uint32(budget)))
			if err := jsonmsgs.WriteRawForTests(nm, msg); err != nil {
				t.Fatalf("budget %d, %q: %v", budget, msg, err)
			}
			raw := bytes.Clone(buf.Bytes())
			if len(msg) > budget {
				checkFragments(t, raw, msg, budget)
			}
			got, err := readRaw(newReader(raw, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(maxFrag), jsonmsgs.WithMaxSize(uint32(budget))))
			if err != nil {
				t.Fatalf("budget %d, %q: read: %v", budget, msg, err)
			}
			if !bytes.Equal(got, msg) {
				t.Fatalf("budget %d: got %q, want %q", budget, got, msg)
			}
		}
	}
}

// TestFragmentDigitBoundaries sizes messages around the points at which the
// number of digits in the header changes.
func TestFragmentDigitBoundaries(t *testing.T) {
	const budget = 40
	for _, n := range []int{99, 100, 101, 999, 1000, 1001, 9999, 10000, 10001} {
		msg := worstCaseText(n)
		var buf bytes.Buffer
		nm := newWriter(&buf, jsonmsgs.WithFragmentation(true), jsonmsgs.WithFragmentSize(budget), jsonmsgs.WithMaxSize(budget))
		if err := jsonmsgs.WriteRawForTests(nm, msg); err != nil {
			t.Fatal(err)
		}
		checkFragments(t, buf.Bytes(), msg, budget)
	}
}

func TestFragmentedJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(64)}
	nm := newWriter(&buf, opts...)
	var want []string
	for i := range 20 {
		v := fmt.Sprintf(`{"id":%d,"text":%q,"list":[1,2.5,null,true]}`, i, strings.Repeat("héllo \"wörld\" 😀\n", i*3))
		want = append(want, v)
		enc := nm.NewEncoder()
		if err := enc.WriteValue(jsontext.Value(v)); err != nil {
			t.Fatal(err)
		}
		if err := nm.WriteMessage(enc); err != nil {
			t.Fatal(err)
		}
	}
	rd := newReader(buf.Bytes(), opts...)
	for i := range want {
		dec, err := rd.ReadMessage()
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		val, err := dec.ReadValue()
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		var a, b any
		if err := stdjson.Unmarshal(val, &a); err != nil {
			t.Fatal(err)
		}
		_ = stdjson.Unmarshal([]byte(want[i]), &b)
		if fmt.Sprint(a) != fmt.Sprint(b) {
			t.Errorf("%d: got %s, want %s", i, val, want[i])
		}
		rd.ReleaseDecoder(dec)
	}
	if _, err := rd.ReadMessage(); err != io.EOF {
		t.Errorf("got %v, want EOF", err)
	}
}

func TestReservedPrefix(t *testing.T) {
	for _, msg := range []string{`{"~":[1,2]}`, `{"~":[0,3],"p":"abc"}`, `{"~":[` + strings.Repeat("1,", 100) + `1]}`} {
		// Whether or not fragmentation is enabled the message is sent in an
		// envelope, that the reader unwraps.
		for _, frag := range []bool{false, true} {
			var buf bytes.Buffer
			opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(frag), jsonmsgs.WithMaxSize(300)}
			if err := jsonmsgs.WriteRawForTests(newWriter(&buf, opts...), []byte(msg)); err != nil {
				t.Fatalf("%q, fragmentation %v: %v", msg, frag, err)
			}
			if frames := splitFrames(t, buf.Bytes()); !bytes.HasPrefix(frames[0], []byte(`{"~":[0,`)) {
				t.Errorf("%q was not wrapped: %q", msg, frames[0])
			}
			got, err := readRaw(newReader(buf.Bytes(), opts...))
			if err != nil || string(got) != msg {
				t.Errorf("%q, fragmentation %v: got %q, %v", msg, frag, got, err)
			}
		}
	}
	// If it does not fit in an envelope, it is an error unless it can be
	// fragmented, and nothing is written.
	big := `{"~":[1,` + strings.Repeat("1,", 200) + `1]}`
	var buf bytes.Buffer
	nm := newWriter(&buf, jsonmsgs.WithMaxSize(100))
	if err := jsonmsgs.WriteRawForTests(nm, []byte(big)); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) || buf.Len() != 0 {
		t.Errorf("got %v, wrote %d bytes, want ErrMessageTooLarge and nothing written", err, buf.Len())
	}
	if err := jsonmsgs.WriteRawForTests(nm, []byte(`{}`)); err != nil {
		t.Errorf("write after rejected message: %v", err)
	}
	// An envelope has to be valid UTF-8.
	bad := append([]byte(`{"~":[1,`), 0xff, ']', '}')
	if err := jsonmsgs.WriteRawForTests(newWriter(&buf), bad); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Errorf("got %v, want ErrInvalidFrame", err)
	}
	// A message that merely resembles the prefix is a message.
	for _, msg := range []string{`{"~": [1]}`, `{ "~":[1]}`, `{"~":1}`, `{"a":[1]}`, `["~"]`} {
		var buf bytes.Buffer
		if err := jsonmsgs.WriteRawForTests(newWriter(&buf), []byte(msg)); err != nil {
			t.Errorf("%q: %v", msg, err)
		}
		if got, err := readRaw(newReader(buf.Bytes())); err != nil || string(got) != msg {
			t.Errorf("%q: got %q, %v", msg, got, err)
		}
	}
}

func TestEmptyMessage(t *testing.T) {
	for _, frag := range []bool{false, true} {
		var buf bytes.Buffer
		nm := newWriter(&buf, jsonmsgs.WithFragmentation(frag))
		if err := jsonmsgs.WriteRawForTests(nm, nil); !errors.Is(err, jsonmsgs.ErrInvalidFrame) || buf.Len() != 0 {
			t.Errorf("fragmentation %v: got %v, wrote %d bytes", frag, err, buf.Len())
		}
	}
}

func TestInvalidUTF8NotWritten(t *testing.T) {
	var buf bytes.Buffer
	nm := newWriter(&buf, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(40))
	msg := append(bytes.Repeat([]byte("a"), 500), 0xff, 'b')
	if err := jsonmsgs.WriteRawForTests(nm, msg); !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
		t.Fatalf("got %v, want ErrInvalidFrame", err)
	}
	if buf.Len() != 0 {
		t.Errorf("%d bytes written before the invalid UTF-8 was found", buf.Len())
	}
	if err := jsonmsgs.WriteRawForTests(nm, []byte(`{}`)); err != nil {
		t.Errorf("write after rejected message: %v", err)
	}
}

// failAfter fails every write after n bytes have been written.
type failAfter struct {
	n, written int
	calls      int
}

var errDisk = errors.New("disk full")

func (f *failAfter) Write(p []byte) (int, error) {
	f.calls++
	if f.written+len(p) > f.n {
		return 0, errDisk
	}
	f.written += len(p)
	return len(p), nil
}

func TestWriterPoisoned(t *testing.T) {
	fw := &failAfter{n: 100}
	nm := newWriter(fw, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(40))
	if err := jsonmsgs.WriteRawForTests(nm, bytes.Repeat([]byte("a"), 1000)); !errors.Is(err, errDisk) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Fatalf("got %v, want a permanent %v", err, errDisk)
	}
	calls := fw.calls
	// A message that would fit, whatever the writer would now do.
	fw.n = 1 << 20
	if err := jsonmsgs.WriteRawForTests(nm, []byte(`{}`)); !errors.Is(err, errDisk) || !errors.Is(err, jsonmsgs.ErrPermanent) {
		t.Errorf("got %v, want the earlier error", err)
	}
	if fw.calls != calls {
		t.Errorf("a poisoned writer wrote")
	}
}

func TestReaderPoisoned(t *testing.T) {
	stream := append(joinFrames(`{"~":[0,5],"p":"abc"}`, `{"a":1}`), joinFrames(`{"b":2}`)...)
	rd := newReader(stream, jsonmsgs.WithFragmentation(true))
	_, err1 := rd.ReadMessage()
	if !errors.Is(err1, jsonmsgs.ErrInvalidFrame) || !errors.Is(err1, jsonmsgs.ErrPermanent) {
		t.Fatalf("got %v, want a permanent ErrInvalidFrame", err1)
	}
	// The frames that follow must not be taken for the start of a message.
	for range 3 {
		if dec, err := rd.ReadMessage(); err != err1 || dec != nil { //nolint:errorlint
			t.Fatalf("got %v, %v, want %v", dec, err, err1)
		}
	}
}

const rejectOK0 = `{"~":[0,6],"p":"abc"}`

var rejectCases = []struct {
	name   string
	frames []string
}{
	{"first is not seq 0", []string{`{"~":[1],"p":"abc"}`}},
	{"seq 0 without total", []string{`{"~":[0],"p":"abc"}`}},
	{"seq 1 with total", []string{rejectOK0, `{"~":[1,3],"p":"abc"}`}},
	{"zero total", []string{`{"~":[0,0],"p":"abc"}`}},
	{"leading zero total", []string{`{"~":[0,06],"p":"abc"}`}},
	{"leading zero seq", []string{rejectOK0, `{"~":[01],"p":"abc"}`}},
	{"eleven digits", []string{`{"~":[0,10000000000],"p":"abc"}`}},
	{"negative total", []string{`{"~":[0,-6],"p":"abc"}`}},
	{"fractional total", []string{`{"~":[0,6.0],"p":"abc"}`}},
	{"exponent total", []string{`{"~":[0,6e0],"p":"abc"}`}},
	{"no digits", []string{`{"~":[,6],"p":"abc"}`}},
	{"whitespace in header", []string{`{"~":[0, 6],"p":"abc"}`}},
	{"whitespace after colon", []string{`{"~":[0,6],"p": "abc"}`}},
	{"whitespace at end", []string{`{"~":[0,6],"p":"abc"} `}},
	{"extra member", []string{`{"~":[0,6],"p":"abc","x":1}`}},
	{"extra header element", []string{`{"~":[0,6,7],"p":"abc"}`}},
	{"wrong key", []string{`{"~":[0,6],"q":"abc"}`}},
	{"empty payload", []string{`{"~":[0,6],"p":""}`}},
	{"payload not a string", []string{`{"~":[0,6],"p":123}`}},
	{"truncated header", []string{`{"~":[0,6`}},
	{"no closing", []string{`{"~":[0,6],"p":"abc`}},
	{"escape cut short", []string{`{"~":[0,6],"p":"abc\"}`}},
	{"unescaped quote", []string{`{"~":[0,6],"p":"a"c"}`}},
	{"unescaped newline", []string{"{\"~\":[0,6],\"p\":\"a\nc\"}"}},
	{"unescaped control", []string{"{\"~\":[0,6],\"p\":\"a\x01c\"}"}},
	{"bad escape", []string{`{"~":[0,6],"p":"a\xc"}`}},
	{"bad unicode escape", []string{`{"~":[0,6],"p":"a\u12g4"}`}},
	{"short unicode escape", []string{`{"~":[0,6],"p":"a\u12"}`}},
	{"lone high surrogate", []string{`{"~":[0,6],"p":"a\ud83dbc"}`}},
	{"lone high surrogate at end", []string{`{"~":[0,6],"p":"abc\ud83d"}`}},
	{"truncated low surrogate 2", []string{`{"~":[0,6],"p":"\ud83d\u"}`}},
	{"truncated low surrogate 3", []string{`{"~":[0,6],"p":"\ud83d\u1"}`}},
	{"truncated low surrogate 4", []string{`{"~":[0,6],"p":"\ud83d\u12"}`}},
	{"truncated low surrogate 5", []string{`{"~":[0,6],"p":"\ud83d\u123"}`}},
	{"lone low surrogate", []string{`{"~":[0,6],"p":"a\ude00bc"}`}},
	{"high surrogate then non low", []string{`{"~":[0,6],"p":"\ud83dA"}`}},
	{"invalid utf8", []string{"{\"~\":[0,6],\"p\":\"a\xffc\"}"}},
	{"overlong utf8", []string{"{\"~\":[0,6],\"p\":\"\xc0\x80\"}"}},
	{"too much data", []string{`{"~":[0,2],"p":"abc"}`}},
	{"too much data in later fragment", []string{`{"~":[0,4],"p":"ab"}`, `{"~":[1],"p":"cde"}`}},
	{"seq skipped", []string{rejectOK0, `{"~":[2],"p":"abc"}`}},
	{"seq repeated", []string{rejectOK0, `{"~":[0,6],"p":"abc"}`}},
	{"seq 0 repeated", []string{`{"~":[0,6],"p":"abc"}`, `{"~":[0,6],"p":"abc"}`}},
	{"bare frame mid message", []string{rejectOK0, `{"a":1}`}},
	{"empty frame mid message", []string{rejectOK0, ``}},
	{"empty frame", []string{``}},
	{"total over limit", []string{`{"~":[0,1000001],"p":"abc"}`}},
	{"huge total", []string{`{"~":[0,4294967295],"p":"abc"}`}},
}

func TestReaderRejects(t *testing.T) {
	for _, tc := range rejectCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := newReader(joinFrames(tc.frames...), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(1_000_000))
			dec, err := rd.ReadMessage()
			if err == nil {
				t.Fatalf("accepted %q: %q", tc.frames, jsonmsgs.DecoderBytesForTests(dec))
			}
			if !errors.Is(err, jsonmsgs.ErrInvalidFrame) && !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
				t.Errorf("got %v, want ErrInvalidFrame or ErrMessageTooLarge", err)
			}
			if !errors.Is(err, jsonmsgs.ErrPermanent) {
				t.Errorf("%v does not match ErrPermanent", err)
			}
			if _, err2 := rd.ReadMessage(); err2 != err { //nolint:errorlint
				t.Errorf("not sticky: got %v, want %v", err2, err)
			}
		})
	}
}

// TestFragmentReaderRejects checks that ReadFragment rejects what ReadMessage does.
func TestFragmentReaderRejects(t *testing.T) {
	for _, tc := range rejectCases {
		t.Run(tc.name, func(t *testing.T) {
			rd := newReader(joinFrames(tc.frames...), jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(1_000_000))
			var err error
			for err == nil {
				_, err = rd.ReadFragment()
			}
			if !errors.Is(err, jsonmsgs.ErrInvalidFrame) && !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
				t.Fatalf("got %v, want ErrInvalidFrame or ErrMessageTooLarge", err)
			}
			if !errors.Is(err, jsonmsgs.ErrPermanent) {
				t.Errorf("%v does not match ErrPermanent", err)
			}
			if _, err2 := rd.ReadFragment(); err2 != err { //nolint:errorlint
				t.Errorf("not sticky: got %v, want %v", err2, err)
			}
		})
	}
}

func TestReaderWithoutFragmentation(t *testing.T) {
	// A message in a single envelope is a message ...
	got, err := readRaw(newReader(joinFrames(`{"~":[0,3],"p":"abc"}`)))
	if err != nil || string(got) != "abc" {
		t.Errorf("got %q, %v", got, err)
	}
	// ... but one that is in more than one is a fragmented message.
	for _, via := range []string{"ReadMessage", "ReadFragment"} {
		rd := newReader(joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[1],"p":"def"}`))
		var err error
		if via == "ReadMessage" {
			_, err = rd.ReadMessage()
		} else {
			_, err = rd.ReadFragment()
		}
		if !errors.Is(err, jsonmsgs.ErrInvalidFrame) {
			t.Errorf("%s: got %v, want ErrInvalidFrame", via, err)
		}
	}
	// And its size is limited to that of a frame.
	rd := newReader(joinFrames(`{"~":[0,300],"p":"abc"}`), jsonmsgs.WithMaxSize(100))
	if _, err := rd.ReadMessage(); !errors.Is(err, jsonmsgs.ErrMessageTooLarge) {
		t.Errorf("got %v, want ErrMessageTooLarge", err)
	}
}

func TestReaderTruncation(t *testing.T) {
	full := joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[1],"p":"def"}`)
	if got, err := readRaw(newReader(full, jsonmsgs.WithFragmentation(true))); err != nil || string(got) != "abcdef" {
		t.Fatalf("got %q, %v", got, err)
	}
	// Every strict prefix is an error, other than the empty one which is a
	// stream that has ended, and one that ends between messages.
	for n := 1; n < len(full); n++ {
		_, err := readRaw(newReader(full[:n], jsonmsgs.WithFragmentation(true)))
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("truncated to %d: got %v, want ErrUnexpectedEOF", n, err)
		}
	}
	if _, err := readRaw(newReader(nil, jsonmsgs.WithFragmentation(true))); err != io.EOF { //nolint:errorlint
		t.Errorf("got %v, want EOF", err)
	}
	// A bare message that is truncated is also an error, not an EOF.
	if _, err := readRaw(newReader(joinFrames(`{"a":1}`)[:8])); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("got %v, want ErrUnexpectedEOF", err)
	}
}

// TestReaderUnescapeMatchesStdlib checks that whatever encoding/json produces
// for a string, and so any JSON encoder, is decoded to the same string.
func TestReaderUnescapeMatchesStdlib(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4)) //nolint:gosec
	for i := range 500 {
		text := randomText(r, 1+r.IntN(60))
		for _, html := range []bool{false, true} {
			var b bytes.Buffer
			e := stdjson.NewEncoder(&b)
			e.SetEscapeHTML(html)
			if err := e.Encode(string(text)); err != nil {
				t.Fatal(err)
			}
			quoted := bytes.TrimSuffix(b.Bytes(), []byte("\n"))
			body := fmt.Sprintf(`{"~":[0,%d],"p":%s}`, len(text), quoted)
			got, err := readRaw(newReader(joinFrames(body), jsonmsgs.WithFragmentation(true)))
			if err != nil || !bytes.Equal(got, text) {
				t.Fatalf("%d: %s: got %q, %v, want %q", i, body, got, err, text)
			}
		}
	}
	// Forms that this package does not itself produce.
	for _, tc := range []struct{ in, want string }{
		{`Aé世`, "Aé世"},
		{`😀`, "😀"},
		{`😀x`, "😀x"},
		{`\/\"\\\b\f\n\r\t`, "/\"\\\b\f\n\r\t"},
		{`a\u0000b`, "a\x00b"},
		{`\u007f `, "\x7f "},
	} {
		body := fmt.Sprintf(`{"~":[0,%d],"p":"%s"}`, len(tc.want), tc.in)
		got, err := readRaw(newReader(joinFrames(body), jsonmsgs.WithFragmentation(true)))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: got %q, %v, want %q", tc.in, got, err, tc.want)
		}
	}
}

// TestReaderAllocatesAsDataArrives checks that the size that a frame declares
// does not cause memory to be allocated before the data arrives.
func TestReaderAllocatesAsDataArrives(t *testing.T) {
	const declared = 1 << 30
	hdr := binary.LittleEndian.AppendUint32(nil, declared)
	stream := slices.Concat(hdr, []byte(`{"a":"`))
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	rd := newReader(stream, jsonmsgs.WithMaxSize(declared))
	_, err := rd.ReadMessage()
	runtime.ReadMemStats(&after)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v", err)
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 4<<20 {
		t.Errorf("allocated %d bytes for a frame that declared %d and delivered %d", got, declared, len(stream)-4)
	}

	// The same for the total declared by a fragment.
	stream = joinFrames(`{"~":[0,16000000],"p":"abc"}`)
	runtime.GC()
	runtime.ReadMemStats(&before)
	rd = newReader(stream, jsonmsgs.WithFragmentation(true))
	_, err = rd.ReadMessage()
	runtime.ReadMemStats(&after)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v", err)
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 4<<20 {
		t.Errorf("allocated %d bytes for a fragment that declared 16000000", got)
	}
}

func TestNewMessagerFragmentSizePanics(t *testing.T) {
	for name, opts := range map[string][]jsonmsgs.Option{
		"too small":        {jsonmsgs.WithFragmentation(true), jsonmsgs.WithFragmentSize(30)},
		"too small, large": {jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(1<<32 - 1), jsonmsgs.WithFragmentSize(32)},
		"max too large":    {jsonmsgs.WithMaxFragmentedMessageSize(1 << 31)},
		"max size":         {jsonmsgs.WithMaxSize(1 << 31)},
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
	// Not an error if not fragmenting.
	jsonmsgs.NewMessager(nil, nil, jsonmsgs.WithFragmentSize(5))
	jsonmsgs.NewMessager(nil, nil, jsonmsgs.WithFragmentation(true), jsonmsgs.WithFragmentSize(30), jsonmsgs.WithMaxFragmentedMessageSize(1000))
}

func TestDefaultsAreBelowBrowserLimit(t *testing.T) {
	if jsonmsgs.DefaultMaxNativeMessageSize > 1_000_000 {
		t.Errorf("default frame size %d exceeds 1,000,000", jsonmsgs.DefaultMaxNativeMessageSize)
	}
	var buf bytes.Buffer
	nm := newWriter(&buf, jsonmsgs.WithFragmentation(true))
	msg := append([]byte(`{"a":"`), bytes.Repeat([]byte("\"\\\n"), 600_000)...)
	msg = append(msg, `"}`...)
	if err := jsonmsgs.WriteRawForTests(nm, msg); err != nil {
		t.Fatal(err)
	}
	for i, f := range splitFrames(t, buf.Bytes()) {
		if len(f) > 1_000_000 {
			t.Errorf("frame %d is %d bytes", i, len(f))
		}
	}
	got, err := readRaw(newReader(buf.Bytes(), jsonmsgs.WithFragmentation(true)))
	if err != nil || !bytes.Equal(got, msg) {
		t.Errorf("round trip failed: %v", err)
	}
}

func TestConcurrentFragmentedMessages(t *testing.T) {
	var sb safeBuffer
	opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(80)}
	nm := newWriter(&sb, opts...)
	const writers, perWriter = 8, 25
	var wg sync.WaitGroup
	want := map[string]bool{}
	var mu sync.Mutex
	for w := range writers {
		wg.Go(func() {
			for i := range perWriter {
				msg := fmt.Sprintf(`{"w":%d,"i":%d,"s":"%s"}`, w, i, strings.Repeat(`x\"é`, 20+w+i))
				mu.Lock()
				want[msg] = true
				mu.Unlock()
				if err := jsonmsgs.WriteRawForTests(nm, []byte(msg)); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	rd := newReader(sb.Bytes(), opts...)
	for range writers * perWriter {
		got, err := readRaw(rd)
		if err != nil {
			t.Fatal(err)
		}
		if !want[string(got)] {
			t.Fatalf("unexpected or interleaved message %q", got)
		}
		delete(want, string(got))
	}
	if len(want) != 0 {
		t.Errorf("%d messages missing", len(want))
	}
}

func TestInteropNode(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	const budget = 700
	r := rand.New(rand.NewPCG(5, 6)) //nolint:gosec
	msgs := []string{
		`{"a":1}`,
		`{"~":[1,2]}`,
		`{"~":[0,1],"p":"x"}`,
		`{"text":"` + strings.Repeat(`é"\世😀 `, 500) + `"}`,
		`{"k":"` + strings.Repeat("a", 5000) + `"}`,
		"{\n\t\"a\":\t[1,\n2]\n}" + strings.Repeat(" \n\t\r", 400),
		"\x01\x02\x7f" + strings.Repeat("\x1f\"\\\u2028\u2029", 300),
		string(randomText(r, 3000)),
	}
	// Every other message has a user header.
	headers := make([]string, len(msgs))
	for i := 1; i < len(msgs); i += 2 {
		headers[i] = fmt.Sprintf(`{"to":%d,"via":["a","b"]}`, i)
	}
	for _, repeatIn := range []bool{false, true} {
		for _, repeatOut := range []bool{false, true} {
			runNodeInterop(t, node, budget, msgs, headers, repeatIn, repeatOut)
		}
	}
}

// runNodeInterop sends msgs, with headers, from this package through the
// JavaScript implementation and back.
func runNodeInterop(t *testing.T, node string, budget int, msgs, headers []string, repeatIn, repeatOut bool) {
	t.Helper()
	opts := headerOpts(40, repeatIn, jsonmsgs.WithMaxSize(uint32(budget)))
	var in bytes.Buffer
	nm := newWriter(&in, opts...)
	for i, m := range msgs {
		var err error
		if headers[i] != "" {
			err = writeWithHeader(nm, m, headers[i])
		} else {
			err = jsonmsgs.WriteRawForTests(nm, []byte(m))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	args := []string{"testdata/fragments.js", fmt.Sprint(budget)}
	if repeatOut {
		args = append(args, "repeat")
	}
	cmd := exec.Command(node, args...)
	cmd.Stdin = &in
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node: %v\n%s", err, stderr.String())
	}
	for _, f := range splitFrames(t, out.Bytes()) {
		if len(f) > budget {
			t.Errorf("JavaScript wrote a frame of %d bytes, over %d", len(f), budget)
		}
	}
	rd := newReader(out.Bytes(), opts...)
	for i, m := range msgs {
		dec, err := rd.ReadMessage()
		if err != nil {
			t.Fatalf("repeat %v/%v: message %d: %v", repeatIn, repeatOut, i, err)
		}
		if got := string(jsonmsgs.DecoderBytesForTests(dec)); got != m || string(dec.Header()) != headers[i] {
			t.Fatalf("repeat %v/%v: message %d: got %q header %q, want %q header %q", repeatIn, repeatOut, i, got, dec.Header(), m, headers[i])
		}
	}
}

func FuzzReadMessage(f *testing.F) {
	f.Add(joinFrames(`{"a":1}`))
	f.Add(joinFrames(`{"~":[0,6],"p":"abc"}`, `{"~":[1],"p":"def"}`))
	f.Add(joinFrames(`{"~":[0,3],"p":"😀"}`))
	f.Add(joinFrames(`{"~":[0,3],"p":"a\u0001"}`))
	f.Fuzz(func(t *testing.T, stream []byte) {
		rd := newReader(stream, jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(1<<16), jsonmsgs.WithMaxFragmentedMessageSize(1<<20))
		for {
			dec, err := rd.ReadMessage()
			if err != nil {
				if _, again := rd.ReadMessage(); again == nil {
					t.Fatalf("read succeeded after error %v", err)
				}
				return
			}
			rd.ReleaseDecoder(dec)
		}
	})
}

func FuzzFragmentRoundTrip(f *testing.F) {
	f.Add([]byte(`{"a":"b"}`), uint8(0))
	f.Add([]byte("\x01\"\\é😀"), uint8(3))
	f.Fuzz(func(t *testing.T, msg []byte, extra uint8) {
		budget := jsonmsgs.MinFragmentSize(1<<20) + uint32(extra)
		opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxSize(budget), jsonmsgs.WithMaxFragmentedMessageSize(1 << 20)}
		var buf bytes.Buffer
		err := jsonmsgs.WriteRawForTests(newWriter(&buf, opts...), msg)
		switch {
		case len(msg) == 0 || (!utf8.Valid(msg) && len(msg) > int(budget)) || bytes.HasPrefix(msg, []byte(`{"~":[`)) && !utf8.Valid(msg):
			if !errors.Is(err, jsonmsgs.ErrInvalidFrame) || buf.Len() != 0 {
				t.Fatalf("got %v, %d bytes written", err, buf.Len())
			}
			return
		case err != nil:
			t.Fatal(err)
		}
		for _, fr := range splitFrames(t, buf.Bytes()) {
			if len(fr) > int(budget) {
				t.Fatalf("frame of %d over budget %d", len(fr), budget)
			}
		}
		got, err := readRaw(newReader(buf.Bytes(), opts...))
		if err != nil || !bytes.Equal(got, msg) {
			t.Fatalf("got %q, %v, want %q", got, err, msg)
		}
	})
}

// Bytes returns the data written to the buffer.
func (s *safeBuffer) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Bytes()
}
