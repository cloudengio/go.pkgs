// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import (
	"bytes"
	"strings"
	"unsafe"
)

// ClearString zeroes the underlying bytes of s in place. The caller must
// ensure s was not created from a string literal (read-only memory).
func ClearString(s string) {
	b := unsafe.Slice(unsafe.StringData(s), len(s))
	clear(b)
}

// ClearBytes zeroes every byte of b in place.
func ClearBytes(b []byte) {
	clear(b)
}

// RedactKeyString redacts s, keeping keep visible characters. A positive keep
// preserves a prefix (RedactStringHead); a negative keep preserves a suffix
// (RedactStringTail).
func RedactKeyString(s string, keep int) string {
	if keep < 0 {
		return RedactStringTail(s, -keep)
	}
	return RedactStringHead(s, keep)
}

// RedactStringTail returns a redacted copy of s that hides everything except
// the last keep bytes, which are shown as a suffix after "******". If keep is
// zero or >= len(s), all characters are replaced with "*".
func RedactStringTail(s string, keep int) string {
	if keep <= 0 || keep >= len(s) {
		return strings.Repeat("*", len(s))
	}
	return "******" + s[len(s)-keep:]
}

// RedactStringHead returns a redacted copy of s that shows the first keep
// bytes followed by "******". If keep is zero or >= len(s), all characters
// are replaced with "*".
func RedactStringHead(s string, keep int) string {
	if keep <= 0 || keep >= len(s) {
		return strings.Repeat("*", len(s))
	}
	return s[:keep] + "******"
}

// RedactKeyBytes redacts b, keeping keep visible bytes. A positive keep
// preserves a prefix (RedactBytesHead); a negative keep preserves a suffix
// (RedactBytesTail).
func RedactKeyBytes(b []byte, keep int) []byte {
	if keep < 0 {
		return RedactBytesTail(b, -keep)
	}
	return RedactBytesHead(b, keep)
}

// RedactBytesHead returns a redacted copy of b that shows the first keep bytes
// followed by "******". If keep is zero or >= len(b), all bytes are replaced
// with '*'.
func RedactBytesHead(b []byte, keep int) []byte {
	if keep <= 0 || keep >= len(b) {
		return bytes.Repeat([]byte("*"), len(b))
	}
	out := make([]byte, keep+6)
	copy(out, b[:keep])
	copy(out[keep:], "******")
	return out
}

// RedactBytesTail returns a redacted copy of b that hides everything except
// the last keep bytes, shown as a suffix after "******". If keep is zero or
// >= len(b), all bytes are replaced with '*'.
func RedactBytesTail(b []byte, keep int) []byte {
	if keep <= 0 || keep >= len(b) {
		return bytes.Repeat([]byte("*"), len(b))
	}
	out := make([]byte, 6+keep)
	copy(out, "******")
	copy(out[6:], b[len(b)-keep:])
	return out
}
