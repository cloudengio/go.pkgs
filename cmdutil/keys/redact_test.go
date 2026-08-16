// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import (
	"strings"
	"testing"
)

func TestClearString(t *testing.T) {
	s := strings.Clone("secret-key")
	ClearString(s)
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			t.Errorf("byte %d not zeroed: got %q", i, s[i])
		}
	}
}

func TestClearBytes(t *testing.T) {
	b := []byte("secret-key")
	ClearBytes(b)
	for i, v := range b {
		if v != 0 {
			t.Errorf("byte %d not zeroed: got %q", i, v)
		}
	}
}

func TestRedactKeyString(t *testing.T) {
	// positive keep -> prefix visible
	if got, want := RedactKeyString("abcdefgh", 3), "abc******"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// negative keep -> suffix visible
	if got, want := RedactKeyString("abcdefgh", -3), "******fgh"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactStringHead(t *testing.T) {
	tests := []struct {
		s    string
		keep int
		want string
	}{
		{"abcdefgh", 3, "abc******"},
		{"abcdefgh", 0, "********"},
		{"abcdefgh", 8, "********"},
		{"abcdefgh", 9, "********"},
		{"abcdefgh", -1, "********"},
	}
	for _, tc := range tests {
		if got := RedactStringHead(tc.s, tc.keep); got != tc.want {
			t.Errorf("RedactStringHead(%q, %d) = %q, want %q", tc.s, tc.keep, got, tc.want)
		}
	}
}

func TestRedactStringTail(t *testing.T) {
	tests := []struct {
		s    string
		keep int
		want string
	}{
		{"abcdefgh", 3, "******fgh"},
		{"abcdefgh", 0, "********"},
		{"abcdefgh", 8, "********"},
		{"abcdefgh", 9, "********"},
		{"abcdefgh", -1, "********"},
	}
	for _, tc := range tests {
		if got := RedactStringTail(tc.s, tc.keep); got != tc.want {
			t.Errorf("RedactStringTail(%q, %d) = %q, want %q", tc.s, tc.keep, got, tc.want)
		}
	}
}

func TestRedactKeyBytes(t *testing.T) {
	// positive keep -> prefix visible
	if got, want := string(RedactKeyBytes([]byte("abcdefgh"), 3)), "abc******"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// negative keep -> suffix visible
	if got, want := string(RedactKeyBytes([]byte("abcdefgh"), -3)), "******fgh"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactBytesHead(t *testing.T) {
	tests := []struct {
		b    []byte
		keep int
		want string
	}{
		{[]byte("abcdefgh"), 3, "abc******"},
		{[]byte("abcdefgh"), 0, "********"},
		{[]byte("abcdefgh"), 8, "********"},
		{[]byte("abcdefgh"), 9, "********"},
		{[]byte("abcdefgh"), -1, "********"},
	}
	for _, tc := range tests {
		if got := string(RedactBytesHead(tc.b, tc.keep)); got != tc.want {
			t.Errorf("RedactBytesHead(%q, %d) = %q, want %q", tc.b, tc.keep, got, tc.want)
		}
	}
}

func TestRedactBytesTail(t *testing.T) {
	tests := []struct {
		b    []byte
		keep int
		want string
	}{
		{[]byte("abcdefgh"), 3, "******fgh"},
		{[]byte("abcdefgh"), 0, "********"},
		{[]byte("abcdefgh"), 8, "********"},
		{[]byte("abcdefgh"), 9, "********"},
		{[]byte("abcdefgh"), -1, "********"},
	}
	for _, tc := range tests {
		if got := string(RedactBytesTail(tc.b, tc.keep)); got != tc.want {
			t.Errorf("RedactBytesTail(%q, %d) = %q, want %q", tc.b, tc.keep, got, tc.want)
		}
	}
}
