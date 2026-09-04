// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math"
	"testing"

	"cloudeng.io/cmdutil/cmdtypes"
)

func TestParseByteSizeValid(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want cmdtypes.ByteSize
	}{
		// bare integers (bytes)
		{"0", 0},
		{"1", 1},
		{"100", 100},
		{"1B", cmdtypes.Byte},
		// decimal units
		{"1KB", cmdtypes.KB},
		{"2KB", 2 * cmdtypes.KB},
		{"1MB", cmdtypes.MB},
		{"1GB", cmdtypes.GB},
		{"1TB", cmdtypes.TB},
		// binary units
		{"1KiB", cmdtypes.KiB},
		{"2KiB", 2 * cmdtypes.KiB},
		{"1MiB", cmdtypes.MiB},
		{"1GiB", cmdtypes.GiB},
		{"1TiB", cmdtypes.TiB},
		// floating-point
		{"1.5KB", 1500},
		{"0.5MiB", cmdtypes.MiB / 2},
		// optional space between number and unit
		{"1 KiB", cmdtypes.KiB},
		{"512 MiB", 512 * cmdtypes.MiB},
		// case-insensitive
		{"1kib", cmdtypes.KiB},
		{"1KIB", cmdtypes.KiB},
		{"1kb", cmdtypes.KB},
		{"1mib", cmdtypes.MiB},
		{"1gib", cmdtypes.GiB},
		{"1tib", cmdtypes.TiB},
	} {
		got, err := cmdtypes.ParseByteSize(tc.in)
		if err != nil {
			t.Errorf("ParseByteSize(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseByteSize(%q): got %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestParseByteSizeErrors(t *testing.T) {
	for _, tc := range []string{
		"",      // empty
		"abc",   // no numeric part
		"1XB",   // unknown unit
		"1 ZiB", // unknown binary-style unit
	} {
		if _, err := cmdtypes.ParseByteSize(tc); err == nil {
			t.Errorf("ParseByteSize(%q): expected error, got nil", tc)
		}
	}
}

func TestByteSizeString(t *testing.T) {
	for _, tc := range []struct {
		v    cmdtypes.ByteSize
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{999, "999B"},
		{cmdtypes.KB, "1KB"},
		{2 * cmdtypes.KB, "2KB"},
		{cmdtypes.MB, "1MB"},
		{cmdtypes.GB, "1GB"},
		{cmdtypes.TB, "1TB"},
		{cmdtypes.KiB, "1KiB"},
		{cmdtypes.MiB, "1MiB"},
		{cmdtypes.GiB, "1GiB"},
		{cmdtypes.TiB, "1TiB"},
		// binary preferred over decimal when both divide evenly
		// 1024 % 1024 == 0 → KiB wins over KB (1024 % 1000 == 24)
		{1024, "1KiB"},
		// 1000 is not divisible by any binary unit, but is divisible by KB
		{1000, "1KB"},
		// value not divisible by any unit → bytes
		{1500, "1500B"},
		{1536, "1536B"},
		// 512 MiB: divisible by MiB
		{512 * cmdtypes.MiB, "512MiB"},
		// negative
		{-cmdtypes.KiB, "-1KiB"},
		{-100, "-100B"},
	} {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("ByteSize(%d).String(): got %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestByteSizeRoundTrip(t *testing.T) {
	for _, s := range []string{
		"0B", "1B", "100B",
		"1KB", "512KB",
		"1MB", "256MB",
		"1GB", "2GB",
		"1TB",
		"1KiB", "4KiB",
		"1MiB", "128MiB", "512MiB",
		"1GiB", "8GiB",
		"1TiB",
	} {
		v, err := cmdtypes.ParseByteSize(s)
		if err != nil {
			t.Errorf("ParseByteSize(%q): unexpected error: %v", s, err)
			continue
		}
		if got := v.String(); got != s {
			t.Errorf("round-trip %q → parse → String: got %q", s, got)
		}
	}
}

func TestByteSizeEdgeCases(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		for _, tc := range []struct {
			v    cmdtypes.ByteSize
			want string
		}{
			// MaxInt64 = 2^63-1 is odd, so no unit divides it evenly.
			{math.MaxInt64, "9223372036854775807B"},
			// MinInt64 = -2^63 = -8388608 TiB (2^63 / 2^40 = 2^23 = 8388608).
			{math.MinInt64, "-8388608TiB"},
		} {
			got := tc.v.String()
			if got != tc.want {
				t.Errorf("ByteSize(%d).String(): got %q, want %q", tc.v, got, tc.want)
			}
		}
	})

	t.Run("parse_minint64_roundtrip", func(t *testing.T) {
		// The canonical string for MinInt64 must parse back exactly.
		// 8388608 = 2^23 and TiB = 2^40, so the product is 2^63 which is
		// exactly representable as float64 (a power of 2).
		got, err := cmdtypes.ParseByteSize("-8388608TiB")
		if err != nil {
			t.Fatalf("ParseByteSize(\"-8388608TiB\"): unexpected error: %v", err)
		}
		if got != math.MinInt64 {
			t.Errorf("got %d, want MinInt64 (%d)", int64(got), int64(math.MinInt64))
		}
	})

	t.Run("parse_overflow", func(t *testing.T) {
		// Values whose byte count exceeds int64 range must be rejected.
		for _, s := range []string{
			// 8388608 TiB = 2^63, exactly one TiB above MaxInt64.
			"8388608TiB",
			// One TiB below MinInt64 = -2^63.
			"-8388609TiB",
			// 9223372037 GB > MaxInt64 (9223372036854775807).
			"9223372037GB",
		} {
			if _, err := cmdtypes.ParseByteSize(s); err == nil {
				t.Errorf("ParseByteSize(%q): expected overflow error, got nil", s)
			}
		}
	})
}

// TestByteSizeAllEncodings verifies that the one type decodes from JSON and
// JSON v2, both of which route through UnmarshalText, and that a value written
// by either reads back unchanged. The YAML equivalent is in cmdyaml, which is
// where this package's YAML behaviour is verified.
func TestByteSizeAllEncodings(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want cmdtypes.ByteSize
	}{
		{"1024", 1024},
		{"1KiB", cmdtypes.KiB},
		{"1 MiB", cmdtypes.MiB},
		{"10MB", 10 * cmdtypes.MB},
		{"1.5GiB", cmdtypes.ByteSize(1.5 * float64(cmdtypes.GiB))},
		{"0B", 0},
	} {
		for _, enc := range []struct {
			name      string
			unmarshal func([]byte, any) error
			marshal   func(any) ([]byte, error)
		}{
			{"json", json.Unmarshal, json.Marshal},
			{"json/v2", func(b []byte, v any) error { return jsonv2.Unmarshal(b, v) }, func(v any) ([]byte, error) { return jsonv2.Marshal(v) }},
		} {
			var got cmdtypes.ByteSize
			if err := enc.unmarshal([]byte(fmt.Sprintf("%q", tc.in)), &got); err != nil {
				t.Errorf("%v: %v: %v", tc.in, enc.name, err)
				continue
			}
			if got != tc.want {
				t.Errorf("%v: %v: got %v, want %v", tc.in, enc.name, int64(got), int64(tc.want))
			}
			// What it writes reads back as the same value.
			buf, err := enc.marshal(got)
			if err != nil {
				t.Errorf("%v: %v: %v", tc.in, enc.name, err)
				continue
			}
			var back cmdtypes.ByteSize
			if err := enc.unmarshal(buf, &back); err != nil {
				t.Errorf("%v: %v: %v", tc.in, enc.name, err)
				continue
			}
			if back != got {
				t.Errorf("%v: %v: round trip gave %v, want %v", tc.in, enc.name, int64(back), int64(got))
			}
		}
	}
}

func TestByteSizeText(t *testing.T) {
	var b cmdtypes.ByteSize
	if err := b.UnmarshalText([]byte("2 GiB")); err != nil {
		t.Fatal(err)
	}
	if got, want := b, 2*cmdtypes.GiB; got != want {
		t.Errorf("got %v, want %v", int64(got), int64(want))
	}
	text, err := b.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(text), "2GiB"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err := b.UnmarshalText([]byte("not-a-size")); err == nil {
		t.Error("expected an error for an unrecognised size")
	}
}
