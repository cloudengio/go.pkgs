// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs_test

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"math/rand/v2"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
)

func naivePlainRun(b []byte, i, end int, ascii bool) int {
	for ; i < end; i++ {
		c := b[i]
		if c < 0x20 || c == '"' || c == '\\' || (ascii && c >= 0x80) {
			break
		}
	}
	return i
}

// checkWords checks that the result of plainWords, or asciiWords, is the start
// of the first word that has a byte that is not plain, or the end.
func checkWords(t *testing.T, name string, b []byte, i, end, got int, ascii bool) {
	t.Helper()
	first := naivePlainRun(b, i, end, ascii)
	// Words are whole, and are plain, and the next is not, if it fits.
	want := i + (first-i)/8*8
	if got != want {
		t.Fatalf("%s(%q, %d, %d) = %d, want %d", name, b, i, end, got, want)
	}
}

// TestScannersExhaustive puts every value of a byte at every position in a run
// of plain bytes, for every starting offset and end, so that every position in
// a word, and every word boundary, is covered.
func TestScannersExhaustive(t *testing.T) {
	const size = 27
	for v := range 256 {
		for pos := range size {
			b := bytes.Repeat([]byte("a"), size)
			b[pos] = byte(v)
			for i := 0; i <= size; i++ {
				if got, want := jsonmsgs.PlainRunForTests(b, i), naivePlainRun(b, i, len(b), false); got != want {
					t.Fatalf("plainRun(value %#x at %d, %d) = %d, want %d", v, pos, i, got, want)
				}
				for _, end := range []int{i, min(i+1, size), min(i+7, size), min(i+8, size), min(i+9, size), size} {
					checkWords(t, "plainWords", b, i, end, jsonmsgs.PlainWordsForTests(b, i, end), false)
					checkWords(t, "asciiWords", b, i, end, jsonmsgs.ASCIIWordsForTests(b, i, end), true)
				}
			}
		}
	}
}

func TestScannersRandom(t *testing.T) {
	r := rand.New(rand.NewPCG(11, 12)) //nolint:gosec
	specials := []byte{0, 1, 0x1f, 0x20, 0x21, 0x22, 0x23, 0x5b, 0x5c, 0x5d, 0x7f, 0x80, 0x81, 0xc3, 0xff}
	for range 20000 {
		b := make([]byte, r.IntN(100))
		for i := range b {
			if r.IntN(25) == 0 {
				b[i] = specials[r.IntN(len(specials))]
			} else {
				b[i] = byte('a' + r.IntN(26))
			}
		}
		i := r.IntN(len(b) + 1)
		end := i + r.IntN(len(b)-i+1)
		if got, want := jsonmsgs.PlainRunForTests(b, i), naivePlainRun(b, i, len(b), false); got != want {
			t.Fatalf("plainRun(%q, %d) = %d, want %d", b, i, got, want)
		}
		checkWords(t, "plainWords", b, i, end, jsonmsgs.PlainWordsForTests(b, i, end), false)
		checkWords(t, "asciiWords", b, i, end, jsonmsgs.ASCIIWordsForTests(b, i, end), true)
	}
}

// longRunText returns text with runs of characters that need no escaping,
// of many lengths, between the characters that do.
func longRunText(r *rand.Rand, n int) []byte {
	specials := []string{"\"", "\\", "\n", "\t", "\x01", "\x1f", "é", "世", "😀", " ", "\x7f"}
	var b []byte
	for len(b) < n {
		for range r.IntN(300) {
			b = append(b, byte('a'+r.IntN(26)))
		}
		b = append(b, specials[r.IntN(len(specials))]...)
	}
	return b
}

// TestUnescapeLongRuns checks the reader against encoding/json for strings with
// runs of every length, so that a run can end anywhere in a word.
func TestUnescapeLongRuns(t *testing.T) {
	r := rand.New(rand.NewPCG(13, 14)) //nolint:gosec
	for i := range 300 {
		text := longRunText(r, 1+r.IntN(2000))
		quoted, err := stdjson.Marshal(string(text))
		if err != nil {
			t.Fatal(err)
		}
		body := fmt.Sprintf(`{"~":[0,%d],"p":%s}`, len(text), quoted)
		got, err := readRaw(newReader(joinFrames(body), jsonmsgs.WithFragmentation(true)))
		if err != nil || !bytes.Equal(got, text) {
			t.Fatalf("%d: got %q, %v, want %q", i, got, err, text)
		}
		// The same, as the later frame of a message, which is moved to follow the first.
		first := `{"~":[0,` + fmt.Sprint(len(text)+1) + `],"p":"x"}`
		second := fmt.Sprintf(`{"~":[1],"p":%s}`, quoted)
		got, err = readRaw(newReader(joinFrames(first, second), jsonmsgs.WithFragmentation(true)))
		if err != nil || !bytes.Equal(got, append([]byte("x"), text...)) {
			t.Fatalf("%d: later frame: got %q, %v", i, got, err)
		}
	}
}

// TestFragmentLongRuns round trips text with long runs at many frame sizes.
func TestFragmentLongRuns(t *testing.T) {
	r := rand.New(rand.NewPCG(15, 16)) //nolint:gosec
	for _, budget := range []int{int(jsonmsgs.MinFragmentSize(1 << 20)), 40, 77, 100, 333, 1000} {
		for range 40 {
			msg := longRunText(r, 1+r.IntN(5000))
			var buf bytes.Buffer
			opts := []jsonmsgs.Option{jsonmsgs.WithFragmentation(true), jsonmsgs.WithMaxFragmentedMessageSize(1 << 20), jsonmsgs.WithMaxSize(uint32(budget))}
			if err := jsonmsgs.WriteRawForTests(newWriter(&buf, opts...), msg); err != nil {
				t.Fatal(err)
			}
			if len(msg) > budget {
				checkFragments(t, buf.Bytes(), msg, budget)
			}
			got, err := readRaw(newReader(buf.Bytes(), opts...))
			if err != nil || !bytes.Equal(got, msg) {
				t.Fatalf("budget %d: %v", budget, err)
			}
		}
	}
}

// referenceEscape returns the escaped form of the prefix of src, which is made
// of whole characters, that is the longest to fit in room bytes, and the length
// of that prefix.
func referenceEscape(src []byte, room int) (out []byte, consumed int) {
	for _, r := range string(src) {
		var esc []byte
		switch {
		case r == '"' || r == '\\':
			esc = []byte{'\\', byte(r)}
		case r == '\n':
			esc = []byte(`\n`)
		case r == '\t':
			esc = []byte(`\t`)
		case r == '\r':
			esc = []byte(`\r`)
		case r == '\b':
			esc = []byte(`\b`)
		case r == '\f':
			esc = []byte(`\f`)
		case r < 0x20:
			esc = []byte(fmt.Sprintf(`\u%04x`, r))
		default:
			esc = []byte(string(r))
		}
		if len(out)+len(esc) > room {
			break
		}
		out = append(out, esc...)
		consumed += len(string(r))
	}
	return out, consumed
}

// TestAppendEscapedFits checks that exactly as many characters are escaped
// as fit in the room, for every room, including exactly full, and for runs
// that are shorter and longer than a word.
func TestAppendEscapedFits(t *testing.T) {
	r := rand.New(rand.NewPCG(17, 18)) //nolint:gosec
	for i := range 3000 {
		var src []byte
		if i%2 == 0 {
			src = randomText(r, 1+r.IntN(40))
		} else {
			src = longRunText(r, 1+r.IntN(120))
		}
		for room := 0; room <= len(src)*2+8 && room < 120; room++ {
			prefix := []byte("hdr")
			got, n := jsonmsgs.AppendEscapedForTests(bytes.Clone(prefix), src, room)
			want, wantN := referenceEscape(src, room)
			if n != wantN || !bytes.Equal(got, append(bytes.Clone(prefix), want...)) {
				t.Fatalf("room %d, %q: got %q, %d; want %q, %d", room, src, got, n, want, wantN)
			}
		}
	}
}
