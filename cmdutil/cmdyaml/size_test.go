// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"testing"

	"cloudeng.io/cmdutil/cmdtypes"
	"gopkg.in/yaml.v3"
)

func TestByteSizeYAML(t *testing.T) {
	type cfg struct {
		Size cmdtypes.ByteSize `yaml:"size"`
	}

	for _, tc := range []struct {
		yaml string
		want cmdtypes.ByteSize
	}{
		// bare integer in YAML
		{"size: 100", 100},
		// decimal units
		{"size: 1KB", cmdtypes.KB},
		{"size: 256MB", 256 * cmdtypes.MB},
		// binary units
		{"size: 1KiB", cmdtypes.KiB},
		{"size: 512MiB", 512 * cmdtypes.MiB},
		{"size: 1GiB", cmdtypes.GiB},
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

// TestByteSizeYAMLMarshal verifies that a size is written in the units it
// divides evenly into and reads back as the same number of bytes, so a value
// can be written to a configuration file and read from it again.
func TestByteSizeYAMLMarshal(t *testing.T) {
	type spec struct {
		Size cmdtypes.ByteSize `yaml:"size"`
	}
	for _, tc := range []struct {
		size cmdtypes.ByteSize
		want string
	}{
		{cmdtypes.KiB, "size: 1KiB\n"},
		{2 * cmdtypes.MiB, "size: 2MiB\n"},
		{10 * cmdtypes.MB, "size: 10MB\n"},
		{0, "size: 0B\n"},
		{1023, "size: 1023B\n"},
		{-cmdtypes.KiB, "size: -1KiB\n"},
	} {
		buf, err := yaml.Marshal(spec{Size: tc.size})
		if err != nil {
			t.Errorf("%v: %v", int64(tc.size), err)
			continue
		}
		if got := string(buf); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
		var back spec
		if err := yaml.Unmarshal(buf, &back); err != nil {
			t.Errorf("%v: %v", int64(tc.size), err)
			continue
		}
		if back.Size != tc.size {
			t.Errorf("got %v, want %v", int64(back.Size), int64(tc.size))
		}
	}
}

func TestByteSizeYAMLErrors(t *testing.T) {
	type spec struct {
		Size cmdtypes.ByteSize `yaml:"size"`
	}
	for _, in := range []string{
		"size: not-a-size\n",
		"size: 10XB\n",
		"size: \"\"\n",
		"size: []\n",
	} {
		var got spec
		if err := yaml.Unmarshal([]byte(in), &got); err == nil {
			t.Errorf("%q: expected an error, got %v", in, int64(got.Size))
		}
	}
}
