// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"fmt"
	"slices"
	"unicode/utf8"
)

// escLen is the length in bytes of each ASCII character once escaped for a
// JSON string, see the table in the package documentation.
var escLen = func() (t [utf8.RuneSelf]uint8) {
	for i := range t {
		t[i] = 1
	}
	for i := range 0x20 {
		t[i] = maxEscapedLen
	}
	for _, c := range []byte{'\b', '\f', '\n', '\r', '\t', '"', '\\'} {
		t[c] = 2
	}
	return t
}()

// escLetter is the character that follows the backslash in the escape sequence
// of each ASCII character that has a two byte one, and is zero for any other.
var escLetter = func() (t [utf8.RuneSelf]byte) {
	for c, l := range map[byte]byte{'"': '"', '\\': '\\', '\b': 'b', '\f': 'f', '\n': 'n', '\r': 'r', '\t': 't'} {
		t[c] = l
	}
	return t
}()

// shortRun is the length of run of characters that need no escape, in the text
// to be escaped or unescaped, after which they are processed 8 at a time.
// Runs in JSON are mostly shorter than this, for which processing the
// characters one at a time is faster than setting up to do 8 at a time.
const shortRun = 8

// appendEscaped appends the characters of the valid UTF-8 text src to dst,
// escaped as the contents of a JSON string, for as long as the escaped form
// fits in room bytes, and returns the extended slice and the number of bytes
// of src it consumed. A character, and so its escape sequence, is written
// whole or not at all: it stops before the first that does not fit in what
// remains of room. This is what guarantees that a fragment fits in the
// budget, see the package documentation.
//
// The output is written directly to the space that dst has for it, which is
// room bytes at most, rather than by appending. Characters are written one at
// a time until there is a run of them that need no escape of more than
// shortRun, which is then copied 8 at a time.
func appendEscaped(dst, src []byte, room int) (out []byte, consumed int) {
	if room <= 0 {
		return dst, 0
	}
	dst = slices.Grow(dst, room)
	o := dst[len(dst) : len(dst)+room]
	n, i, run := 0, 0, 0
	for i < len(src) {
		c := src[i]
		if c >= utf8.RuneSelf {
			// The text is valid UTF-8, which the caller has checked, so the
			// width of a character is that of its leading byte.
			width := 4
			switch {
			case c < 0xE0:
				width = 2
			case c < 0xF0:
				width = 3
			}
			if n+width > room {
				break
			}
			n += copy(o[n:], src[i:i+width])
			i += width
			run = 0
			continue
		}
		e := int(escLen[c])
		if n+e > room {
			break
		}
		if e == 1 {
			o[n] = c
			n++
			i++
			if run++; run >= shortRun {
				j := asciiWords(src, i, min(len(src), i+room-n))
				n += copy(o[n:], src[i:j])
				i, run = j, 0
			}
			continue
		}
		run = 0
		i++
		o[n] = '\\'
		if l := escLetter[c]; l != 0 {
			o[n+1] = l
			n += 2
			continue
		}
		const hex = "0123456789abcdef"
		o[n+1], o[n+2], o[n+3], o[n+4], o[n+5] = 'u', '0', '0', hex[c>>4], hex[c&0xf]
		n += 6
	}
	return dst[:len(dst)+n], i
}

// parseUint parses the decimal number that starts at b[i], with no sign and no
// leading zeros as JSON requires, and of at most maxDigits digits, returning
// it and the index of the byte after it.
func parseUint(b []byte, i int) (v uint64, next int, ok bool) {
	start := i
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		if i-start >= maxDigits {
			return 0, 0, false
		}
		v = v*10 + uint64(b[i]-'0')
		i++
	}
	if i == start || (b[start] == '0' && i-start > 1) {
		return 0, 0, false
	}
	return v, i, true
}

// fragmentHeader is what precedes the payload of a fragment.
type fragmentHeader struct {
	seq, total uint64 // total is zero except for the first fragment
	hdr        []byte // the user header, a slice of the fragment, or nil
	payloadAt  int    // the index at which the escaped payload begins
}

// parseFragmentHeader parses the header of the fragment in body, which is
// known to begin with envelopePrefix, and checks the rest of its form, which
// is fixed. The payload ends 2 bytes before the end of body. No variation of
// the form is accepted: see the package documentation. A user header is
// accepted only if maxHeader is not zero and it is not larger than that.
func parseFragmentHeader(body []byte, maxHeader uint32) (fh fragmentHeader, err error) {
	bad := func(what string) (fragmentHeader, error) {
		return fragmentHeader{}, fmt.Errorf("%w: malformed fragment: %s", ErrInvalidFrame, what)
	}
	if !bytes.HasPrefix(body, []byte(envelopePrefix)) {
		return bad("does not begin with " + envelopePrefix)
	}
	seq, i, ok := parseUint(body, len(envelopePrefix))
	if !ok {
		return bad("invalid sequence number")
	}
	fh.seq = seq
	if seq == 0 {
		if i >= len(body) || body[i] != ',' {
			return bad("the first fragment has no total")
		}
		fh.total, i, ok = parseUint(body, i+1)
		if !ok || fh.total == 0 {
			return bad("invalid total")
		}
	}
	if fh.hdr, i, err = parseUserHeader(body, i, maxHeader); err != nil {
		return fragmentHeader{}, err
	}
	// A payload is at least one byte, then the closing "}.
	if len(body)-i < 1+len(envelopeSuffix) || !bytes.HasSuffix(body, []byte(envelopeSuffix)) {
		return bad("empty payload or missing closing")
	}
	fh.payloadAt = i
	return fh, nil
}

// parseUserHeader parses what follows the numbers of a fragment's header,
// which begins at body[i]: either the end of the header and the start of the
// payload, or the length of a user header, then the header, then the start of
// the payload. It returns the user header, or nil, and the index of the start
// of the payload.
func parseUserHeader(body []byte, i int, maxHeader uint32) (hdr []byte, payloadAt int, err error) {
	bad := func(what string) ([]byte, int, error) {
		return nil, 0, fmt.Errorf("%w: malformed fragment: %s", ErrInvalidFrame, what)
	}
	if i >= len(body) || body[i] != ',' {
		if !bytes.HasPrefix(body[i:], []byte(envelopeMid)) {
			return bad("invalid header")
		}
		return nil, i + len(envelopeMid), nil
	}
	if maxHeader == 0 {
		return bad("a user header was received but they are not enabled")
	}
	hlen, j, ok := parseUint(body, i+1)
	if !ok || hlen == 0 || hlen > uint64(maxHeader) {
		return bad("invalid user header length")
	}
	if !bytes.HasPrefix(body[j:], []byte(envelopeHdrMid)) {
		return bad("invalid header")
	}
	j += len(envelopeHdrMid)
	if uint64(len(body)-j) < hlen {
		return bad("user header cut short")
	}
	hdr = body[j : j+int(hlen)]
	j += int(hlen)
	if !bytes.HasPrefix(body[j:], []byte(envelopeHdrEnd)) {
		return bad("invalid header")
	}
	if !jsontext.Value(hdr).IsValid() {
		return bad("user header is not valid JSON")
	}
	return hdr, j + len(envelopeHdrEnd), nil
}

// hexValue returns the value of the hex digit c, or -1.
func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// hex4 returns the value of the 4 hex digits at b[i:], or -1.
func hex4(b []byte, i int) rune {
	if i+4 > len(b) {
		return -1
	}
	var v rune
	for _, c := range b[i : i+4] {
		d := hexValue(c)
		if d < 0 {
			return -1
		}
		v = v<<4 | rune(d)
	}
	return v
}

// unescapeChunk decodes the contents of the JSON string at buf[from:to] and
// writes the result to buf at w, returning the index after what was written.
// The destination may be, and is, the same slice as the source, with w no
// greater than from: decoding never lengthens the data, so what is written is
// never ahead of what is read, and the data is decoded where it lies. It
// accepts every form of escape that a JSON encoder may produce, and rejects
// anything that is not a valid JSON string or is not valid UTF-8 once decoded
// (including a surrogate that is not part of a pair), a quote or control
// character that is not escaped, and an escape that is cut short.
func unescapeChunk(buf []byte, from, to, w int) (int, error) {
	start := w
	run := 0
	for r := from; r < to; {
		c := buf[r]
		if chunkOK[c] {
			buf[w] = c
			w++
			r++
			if run++; run >= shortRun {
				j := plainWords(buf, r, to)
				w += copy(buf[w:], buf[r:j])
				r, run = j, 0
			}
			continue
		}
		run = 0
		switch {
		case c < 0x20:
			return 0, errPayload("unescaped control character")
		case c == '"':
			return 0, errPayload("unescaped quote")
		}
		r++
		if r >= to {
			return 0, errPayload("escape cut short")
		}
		if e := simpleEscapes[buf[r]]; e != 0 {
			buf[w] = e
			w++
			r++
			continue
		}
		if buf[r] != 'u' {
			return 0, errPayload("invalid escape")
		}
		v, next, err := decodeUnicodeEscape(buf[:to], r+1)
		if err != nil {
			return 0, err
		}
		w += utf8.EncodeRune(buf[w:], v)
		r = next
	}
	if !utf8.Valid(buf[start:w]) {
		return 0, errPayload("not valid UTF-8")
	}
	return w, nil
}

func errPayload(what string) error {
	return fmt.Errorf("%w: malformed fragment payload: %s", ErrInvalidFrame, what)
}

// simpleEscapes maps the character following a backslash to the character it
// stands for, for the escapes that are a single character, and is zero for
// any other.
var simpleEscapes = func() (t [256]byte) {
	for k, v := range map[byte]byte{'"': '"', '\\': '\\', '/': '/', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t'} {
		t[k] = v
	}
	return t
}()

// decodeUnicodeEscape decodes the hex digits of a \u escape that begin at b[i],
// ie. just after the u, and, if they are the first half of a surrogate pair,
// the second half that must follow. It returns the character and the index
// of the byte after the escape or pair.
func decodeUnicodeEscape(b []byte, i int) (rune, int, error) {
	bad := func(what string) (rune, int, error) {
		return 0, 0, errPayload(what)
	}
	v := hex4(b, i)
	if v < 0 {
		return bad("invalid \\u escape")
	}
	i += 4
	switch {
	case v >= 0xDC00 && v <= 0xDFFF:
		return bad("unpaired low surrogate")
	case v >= 0xD800 && v <= 0xDBFF:
		// The first half of a pair; the second must follow.
		if i+6 > len(b) || b[i] != '\\' || b[i+1] != 'u' {
			return bad("unpaired high surrogate")
		}
		lo := hex4(b, i+2)
		if lo < 0xDC00 || lo > 0xDFFF {
			return bad("unpaired high surrogate")
		}
		return 0x10000 + (v-0xD800)<<10 + (lo - 0xDC00), i + 6, nil
	}
	return v, i, nil
}

const (
	lo8 = 0x0101010101010101
	hi8 = 0x8080808080808080
)

// special reports whether any of the 8 bytes of x is a control character, a
// quote or a backslash, which is to say whether it needs attention when it is
// the contents of a JSON string.
func special(x uint64) bool {
	ctl := (x - 0x20*lo8) &^ x & hi8
	q := x ^ ('"' * lo8)
	q = (q - lo8) &^ q & hi8
	bs := x ^ ('\\' * lo8)
	bs = (bs - lo8) &^ bs & hi8
	return ctl|q|bs != 0
}

// chunkOK is true for the bytes that stand for themselves in the contents of a
// JSON string: all but the control characters, the quote and the backslash.
var chunkOK = func() (t [256]bool) {
	for i := 0x20; i < len(t); i++ {
		t[i] = i != '"' && i != '\\'
	}
	return t
}()

// plainWords returns i advanced over whole words, of 8 bytes, of b[:end] that
// have no control character, quote or backslash in them.
func plainWords(b []byte, i, end int) int {
	for i+8 <= end && !special(binary.LittleEndian.Uint64(b[i:])) {
		i += 8
	}
	return i
}

// asciiWords is plainWords for words in which every byte is ASCII.
func asciiWords(b []byte, i, end int) int {
	for i+8 <= end {
		x := binary.LittleEndian.Uint64(b[i:])
		if x&hi8 != 0 || special(x) {
			break
		}
		i += 8
	}
	return i
}

// plainRun returns the index of the first byte of b at or after i that is a
// control character, a quote or a backslash, or len(b). The first shortRun
// bytes are checked one at a time and the rest 8 at a time.
func plainRun(b []byte, i int) int {
	lim := min(len(b), i+shortRun)
	for i < lim && chunkOK[b[i]] {
		i++
	}
	if i < lim {
		return i
	}
	i = plainWords(b, i, len(b))
	for i < len(b) && chunkOK[b[i]] {
		i++
	}
	return i
}

// unescapedLen validates the contents of a JSON string, that is the payload of
// a fragment, as unescapeChunk does, but without decoding it, and returns the
// number of bytes that it stands for.
func unescapedLen(b []byte) (int, error) {
	if !utf8.Valid(b) {
		return 0, errPayload("not valid UTF-8")
	}
	n := 0
	for r := 0; r < len(b); {
		// Fast path for runs of characters that need no escaping.
		if j := plainRun(b, r); j > r {
			n += j - r
			r = j
			if r == len(b) {
				break
			}
		}
		c := b[r]
		switch {
		case c < 0x20:
			return 0, errPayload("unescaped control character")
		case c == '"':
			return 0, errPayload("unescaped quote")
		}
		r++
		if r >= len(b) {
			return 0, errPayload("escape cut short")
		}
		if simpleEscapes[b[r]] != 0 {
			n++
			r++
			continue
		}
		if b[r] != 'u' {
			return 0, errPayload("invalid escape")
		}
		v, next, err := decodeUnicodeEscape(b, r+1)
		if err != nil {
			return 0, err
		}
		n += utf8.RuneLen(v)
		r = next
	}
	return n, nil
}
