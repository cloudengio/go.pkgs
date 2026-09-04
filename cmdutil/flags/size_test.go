// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	goflag "flag"
	"io"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdtypes"
	"cloudeng.io/cmdutil/flags"
)

func TestByteSizeFlag(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int64
	}{
		{"1024", 1024},
		{"1KiB", 1024},
		{"1 KiB", 1024},
		{"1kib", 1024},
		{"10MB", 10_000_000},
		{"2MiB", 2 * 1024 * 1024},
		{"1.5GiB", 1610612736},
		{"0", 0},
		{"512B", 512},
	} {
		var b flags.ByteSize
		if err := b.Set(tc.in); err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if got := b.Bytes(); got != tc.want {
			t.Errorf("%v: got %v, want %v", tc.in, got, tc.want)
		}
		if got, ok := b.Get().(int64); !ok || got != tc.want {
			t.Errorf("%v: Get returned %v (%T), want %v", tc.in, b.Get(), b.Get(), tc.want)
		}
		// String reports the value as supplied, so a default keeps the units
		// it was written in.
		if got := b.String(); got != tc.in {
			t.Errorf("%v: String returned %v", tc.in, got)
		}
		if b.IsDefault() {
			t.Errorf("%v: a set value should not report itself as the default", tc.in)
		}
	}
}

// TestByteSizeFlagMatchesCmdtypes verifies that the flag parses exactly as
// cmdtypes.ByteSize does, so a size given on the command line is the same
// value as one read from JSON or YAML.
func TestByteSizeFlagMatchesCmdtypes(t *testing.T) {
	for _, in := range []string{"1024", "1KiB", "10MB", "1.5GiB", "2 MiB", "-1KiB"} {
		var b flags.ByteSize
		if err := b.Set(in); err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		want, err := cmdtypes.ParseByteSize(in)
		if err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		if got := b.ByteSize(); got != want {
			t.Errorf("%v: got %v, want %v", in, int64(got), int64(want))
		}
	}
}

func TestByteSizeFlagErrors(t *testing.T) {
	for _, in := range []string{"not-a-size", "", "10XB", "1.2.3KiB"} {
		var b flags.ByteSize
		err := b.Set(in)
		if err == nil {
			t.Errorf("%q: expected an error, got %v", in, b.Bytes())
			continue
		}
		// A rejected value leaves the flag unset.
		if !b.IsDefault() {
			t.Errorf("%q: a rejected value should leave the flag unset", in)
		}
		if got := b.Bytes(); got != 0 {
			t.Errorf("%q: got %v, want 0 after a rejected value", in, got)
		}
	}
}

func TestByteSizeFlagIsDefault(t *testing.T) {
	var b flags.ByteSize
	if !b.IsDefault() {
		t.Error("a new value should report itself as the default")
	}
	if got := b.String(); got != "" {
		t.Errorf("got %q, want an empty string before being set", got)
	}
	if err := b.Set("1KiB"); err != nil {
		t.Fatal(err)
	}
	if b.IsDefault() {
		t.Error("a set value should not report itself as the default")
	}
}

// TestByteSizeAsFlag verifies that it behaves as a flag.Value when registered
// with the standard flag package, including its default.
func TestByteSizeAsFlag(t *testing.T) {
	var b flags.ByteSize
	if err := b.Set("10MB"); err != nil {
		t.Fatal(err)
	}
	fset := goflag.NewFlagSet("test", goflag.ContinueOnError)
	fset.Var(&b, "size", "maximum size")
	if got, want := fset.Lookup("size").DefValue, "10MB"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err := fset.Parse([]string{"--size=2MiB"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, want := b.Bytes(), int64(2*1024*1024); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var q flags.ByteSize
	fset = goflag.NewFlagSet("test", goflag.ContinueOnError)
	fset.SetOutput(io.Discard)
	fset.Var(&q, "size", "maximum size")
	if err := fset.Parse([]string{"--size=nonsense"}); err == nil {
		t.Error("expected an error for an invalid size, got nil")
	} else if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("got %v, want it to report an invalid size", err)
	}
}
