// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	goflag "flag"
	"io"
	"io/fs"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func TestPermissionsSet(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want fs.FileMode
	}{
		{"0700", 0700},
		{"700", 0700},
		{"0o700", 0700},
		{"0x1c0", 0700},
		{"rwx------", 0700},
		{"-rwx------", 0700},
		{"rwx", 0700},
		{"u=rwx,go=", 0700},
		{"rwxr-xr-x", 0755},
		{"u=rwx,go=rx", 0755},
		{"0644", 0644},
	} {
		var p flags.Permissions
		if err := p.Set(tc.in); err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if got := p.FileMode(); got != tc.want {
			t.Errorf("%v: got %04o, want %04o", tc.in, got, tc.want)
		}
		// Get returns the mode for flag.Getter.
		if got, ok := p.Get().(fs.FileMode); !ok || got != tc.want {
			t.Errorf("%v: Get returned %v (%T), want %04o", tc.in, p.Get(), p.Get(), tc.want)
		}
		// String reports the value as supplied, so that a default keeps the
		// notation it was written in.
		if got := p.String(); got != tc.in {
			t.Errorf("%v: String returned %v", tc.in, got)
		}
	}
}

func TestPermissionsOctal(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"rwxr-xr-x", "0755"},
		{"u=rwx,go=", "0700"},
		{"644", "0644"},
		{"0o700", "0700"},
	} {
		var p flags.Permissions
		if err := p.Set(tc.in); err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if got := p.Octal(); got != tc.want {
			t.Errorf("%v: got %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestPermissionsIsDefault(t *testing.T) {
	var p flags.Permissions
	if !p.IsDefault() {
		t.Error("a new value should report itself as the default")
	}
	if got, want := p.FileMode(), fs.FileMode(0); got != want {
		t.Errorf("got %04o, want %04o before being set", got, want)
	}
	if got := p.String(); got != "" {
		t.Errorf("got %q, want an empty string before being set", got)
	}
	if err := p.Set("0700"); err != nil {
		t.Fatal(err)
	}
	if p.IsDefault() {
		t.Error("a set value should not report itself as the default")
	}
}

func TestPermissionsErrors(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"not-a-permission", "invalid permissions format"},
		{"999", "invalid permissions format"},
		{"", "empty permissions"},
		{"rwxr-xr", "invalid permissions format"},
	} {
		var p flags.Permissions
		err := p.Set(tc.in)
		if err == nil {
			t.Errorf("%v: expected an error, got %04o", tc.in, p.FileMode())
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want it to contain %q", tc.in, err, tc.want)
		}
		// A rejected value leaves the flag unset.
		if !p.IsDefault() {
			t.Errorf("%v: a rejected value should leave the flag unset", tc.in)
		}
	}
}

// TestPermissionsAsFlag verifies that it behaves as a flag.Value when
// registered with the standard flag package, including its default.
func TestPermissionsAsFlag(t *testing.T) {
	var p flags.Permissions
	if err := p.Set("0755"); err != nil {
		t.Fatal(err)
	}
	fset := goflag.NewFlagSet("test", goflag.ContinueOnError)
	fset.Var(&p, "perm", "file permissions")

	// The default is reported in the notation it was set in.
	if got, want := fset.Lookup("perm").DefValue, "0755"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if err := fset.Parse([]string{"--perm=u=rwx,go="}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, want := p.FileMode(), fs.FileMode(0700); got != want {
		t.Errorf("got %04o, want %04o", got, want)
	}

	// An invalid value is rejected by the flag package rather than accepted.
	var q flags.Permissions
	fset = goflag.NewFlagSet("test", goflag.ContinueOnError)
	fset.SetOutput(io.Discard)
	fset.Var(&q, "perm", "file permissions")
	if err := fset.Parse([]string{"--perm=nonsense"}); err == nil {
		t.Error("expected an error for an invalid permission, got nil")
	}
}
