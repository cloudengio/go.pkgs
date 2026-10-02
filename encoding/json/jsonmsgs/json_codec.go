// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
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

// appendEscape appends the escape sequence for the ASCII character c, which
// has an escape sequence, ie. escLen[c] is not 1.
func appendEscape(dst []byte, c byte) []byte {
	switch c {
	case '"', '\\':
		return append(dst, '\\', c)
	case '\b':
		return append(dst, '\\', 'b')
	case '\f':
		return append(dst, '\\', 'f')
	case '\n':
		return append(dst, '\\', 'n')
	case '\r':
		return append(dst, '\\', 'r')
	case '\t':
		return append(dst, '\\', 't')
	}
	const hex = "0123456789abcdef"
	return append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
}

// appendEscaped appends the characters of the valid UTF-8 text src to dst,
// escaped as the contents of a JSON string, for as long as the escaped form
// fits in room bytes, and returns the extended slice and the number of bytes
// of src it consumed. A character, and so its escape sequence, is written
// whole or not at all: it stops before the first that does not fit in what
// remains of room. This is what guarantees that a fragment fits in the
// budget, see the package documentation.
//
// Runs of characters that need no escape are copied in one operation, rather
// than character by character; start is the beginning of the run not yet
// copied.
func appendEscaped(dst, src []byte, room int) (out []byte, consumed int) {
	used, i, start := 0, 0, 0
	for i < len(src) {
		// Fast path for a run of characters that are themselves, each of which
		// is a single byte, bounded by what remains of room.
		if end := min(len(src), i+room-used); i < end {
			j := i
			for j < end && src[j] < utf8.RuneSelf && escLen[src[j]] == 1 {
				j++
			}
			used += j - i
			i = j
			if i == len(src) {
				break
			}
		}
		c := src[i]
		var width, esc int
		if c < utf8.RuneSelf {
			width, esc = 1, int(escLen[c])
		} else {
			// The text is valid UTF-8, which the caller has checked, so
			// the width of a character is that of its leading byte.
			switch {
			case c < 0xE0:
				width = 2
			case c < 0xF0:
				width = 3
			default:
				width = 4
			}
			esc = width
		}
		if used+esc > room {
			break
		}
		used += esc
		if c < utf8.RuneSelf && esc != 1 {
			dst = append(dst, src[start:i]...)
			dst = appendEscape(dst, c)
			start = i + 1
		}
		i += width
	}
	return append(dst, src[start:i]...), i
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
	// The common case, no escapes: validate and move.
	plain := true
	for _, c := range buf[from:to] {
		if c < 0x20 {
			return 0, errPayload("unescaped control character")
		}
		if c == '\\' {
			plain = false
		}
		// A quote is escaped, and so preceded by a backslash, or is an error.
		if c == '"' && plain {
			return 0, errPayload("unescaped quote")
		}
	}
	if plain {
		w += copy(buf[w:], buf[from:to])
	} else {
		var err error
		if w, err = unescapeEscapes(buf, from, to, w); err != nil {
			return 0, err
		}
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

// unescapeEscapes is unescapeChunk for a payload that contains a backslash.
func unescapeEscapes(buf []byte, from, to, w int) (int, error) {
	for r := from; r < to; {
		c := buf[r]
		if c == '"' {
			return 0, errPayload("unescaped quote")
		}
		if c != '\\' {
			buf[w] = c
			w++
			r++
			continue
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
	return w, nil
}

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
		if i+2 >= len(b) || b[i] != '\\' || b[i+1] != 'u' {
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

// unescapedLen validates the contents of a JSON string, that is the payload of
// a fragment, as unescapeChunk does, but without decoding it, and returns the
// number of bytes that it stands for.
func unescapedLen(b []byte) (int, error) {
	if !utf8.Valid(b) {
		return 0, errPayload("not valid UTF-8")
	}
	n := 0
	for r := 0; r < len(b); {
		c := b[r]
		switch {
		case c < 0x20:
			return 0, errPayload("unescaped control character")
		case c == '"':
			return 0, errPayload("unescaped quote")
		case c != '\\':
			n++
			r++
			continue
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
