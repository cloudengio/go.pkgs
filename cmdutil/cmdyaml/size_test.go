// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

func TestParseByteSizeValid(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want cmdyaml.ByteSize
	}{
		// bare integers (bytes)
		{"0", 0},
		{"1", 1},
		{"100", 100},
		{"1B", cmdyaml.Byte},
		// decimal units
		{"1KB", cmdyaml.KB},
		{"2KB", 2 * cmdyaml.KB},
		{"1MB", cmdyaml.MB},
		{"1GB", cmdyaml.GB},
		{"1TB", cmdyaml.TB},
		// binary units
		{"1KiB", cmdyaml.KiB},
		{"2KiB", 2 * cmdyaml.KiB},
		{"1MiB", cmdyaml.MiB},
		{"1GiB", cmdyaml.GiB},
		{"1TiB", cmdyaml.TiB},
		// floating-point
		{"1.5KB", 1500},
		{"0.5MiB", cmdyaml.MiB / 2},
		// optional space between number and unit
		{"1 KiB", cmdyaml.KiB},
		{"512 MiB", 512 * cmdyaml.MiB},
		// case-insensitive
		{"1kib", cmdyaml.KiB},
		{"1KIB", cmdyaml.KiB},
		{"1kb", cmdyaml.KB},
		{"1mib", cmdyaml.MiB},
		{"1gib", cmdyaml.GiB},
		{"1tib", cmdyaml.TiB},
	} {
		got, err := cmdyaml.ParseByteSize(tc.in)
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
		"",       // empty
		"abc",    // no numeric part
		"1XB",    // unknown unit
		"1 ZiB",  // unknown binary-style unit
	} {
		if _, err := cmdyaml.ParseByteSize(tc); err == nil {
			t.Errorf("ParseByteSize(%q): expected error, got nil", tc)
		}
	}
}

func TestByteSizeString(t *testing.T) {
	for _, tc := range []struct {
		v    cmdyaml.ByteSize
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{999, "999B"},
		{cmdyaml.KB, "1KB"},
		{2 * cmdyaml.KB, "2KB"},
		{cmdyaml.MB, "1MB"},
		{cmdyaml.GB, "1GB"},
		{cmdyaml.TB, "1TB"},
		{cmdyaml.KiB, "1KiB"},
		{cmdyaml.MiB, "1MiB"},
		{cmdyaml.GiB, "1GiB"},
		{cmdyaml.TiB, "1TiB"},
		// binary preferred over decimal when both divide evenly
		// 1024 % 1024 == 0 → KiB wins over KB (1024 % 1000 == 24)
		{1024, "1KiB"},
		// 1000 is not divisible by any binary unit, but is divisible by KB
		{1000, "1KB"},
		// value not divisible by any unit → bytes
		{1500, "1500B"},
		{1536, "1536B"},
		// 512 MiB: divisible by MiB
		{512 * cmdyaml.MiB, "512MiB"},
		// negative
		{-cmdyaml.KiB, "-1KiB"},
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
		v, err := cmdyaml.ParseByteSize(s)
		if err != nil {
			t.Errorf("ParseByteSize(%q): unexpected error: %v", s, err)
			continue
		}
		if got := v.String(); got != s {
			t.Errorf("round-trip %q → parse → String: got %q", s, got)
		}
	}
}

func TestByteSizeYAML(t *testing.T) {
	type cfg struct {
		Size cmdyaml.ByteSize `yaml:"size"`
	}

	for _, tc := range []struct {
		yaml string
		want cmdyaml.ByteSize
	}{
		// bare integer in YAML
		{"size: 100", 100},
		// decimal units
		{"size: 1KB", cmdyaml.KB},
		{"size: 256MB", 256 * cmdyaml.MB},
		// binary units
		{"size: 1KiB", cmdyaml.KiB},
		{"size: 512MiB", 512 * cmdyaml.MiB},
		{"size: 1GiB", cmdyaml.GiB},
	} {
		var c cfg
		if err := yaml.Unmarshal([]byte(tc.yaml), &c); err != nil {
			t.Errorf("Unmarshal(%q): %v", tc.yaml, err)
			continue
		}
		if c.Size != tc.want {
			t.Errorf("Unmarshal(%q): got %v, want %v", tc.yaml, c.Size, tc.want)
		}

		// round-trip through marshal + unmarshal
		out, err := yaml.Marshal(c)
		if err != nil {
			t.Errorf("Marshal(%q): %v", tc.yaml, err)
			continue
		}
		var c2 cfg
		if err := yaml.Unmarshal(out, &c2); err != nil {
			t.Errorf("Unmarshal(roundtrip of %q): %v", tc.yaml, err)
			continue
		}
		if c2.Size != tc.want {
			t.Errorf("roundtrip %q: got %v, want %v", tc.yaml, c2.Size, tc.want)
		}
	}
}
