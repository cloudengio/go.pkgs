// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes_test

import (
	"io/fs"
	"testing"

	"cloudeng.io/cmdutil/cmdtypes"
)

func TestParsePermissions(t *testing.T) {
	tests := []struct {
		input   string
		want    fs.FileMode
		wantErr bool
	}{
		// Octal strings
		{"0700", 0700, false},
		{"700", 0700, false},
		{"0o700", 0700, false},
		{"0O700", 0700, false},
		{"0755", 0755, false},
		{"755", 0755, false},
		{"0o755", 0755, false},
		{"0644", 0644, false},
		{"644", 0644, false},
		{"0600", 0600, false},
		{"600", 0600, false},
		{"0777", 0777, false},
		{"777", 0777, false},
		{"0x1c0", 0700, false},

		// 9-character rwx format
		{"rwx------", 0700, false},
		{"rwxr-xr-x", 0755, false},
		{"rw-r--r--", 0644, false},
		{"rw-------", 0600, false},
		{"rwxrwxrwx", 0777, false},
		{"rwxr-x---", 0750, false},
		{"rwsr-xr-x", fs.ModeSetuid | 0755, false},
		{"rwxr-sr-x", fs.ModeSetgid | 0755, false},
		{"rwxr-xr-t", fs.ModeSticky | 0755, false},

		// 10-character format (with type prefix)
		{"-rwx------", 0700, false},
		{"-rwxr-xr-x", 0755, false},
		{"drwxr-xr-x", 0755, false},
		{"-rw-r--r--", 0644, false},

		// 3-character user shorthand
		{"rwx", 0700, false},
		{"r-x", 0500, false},
		{"rw-", 0600, false},

		// Symbolic chmod format
		{"u=rwx,go=", 0700, false},
		{"u=rwx,go=rx", 0755, false},
		{"u=rwx,g=rx,o=rx", 0755, false},
		{"a=rwx", 0777, false},
		{"a=rx", 0555, false},
		{"u=rw,go=r", 0644, false},
		{"u=rwx", 0700, false},

		// Whitespace handling
		{"  0700  ", 0700, false},
		{"  rwxr-xr-x  ", 0755, false},

		// Errors
		{"", 0, true},
		{"   ", 0, true},
		{"invalid", 0, true},
		{"rwxr-xr-z", 0, true},
		{"899", 0, true},
		{"u=rwx,invalid", 0, true},
		{"99999", 0, true},
	}

	for _, tc := range tests {
		got, err := cmdtypes.ParsePermissions(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParsePermissions(%q) expected error, got %v (%04o)", tc.input, got, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePermissions(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got.FileMode() != tc.want {
			t.Errorf("ParsePermissions(%q) = %04o, want %04o", tc.input, got.FileMode(), tc.want)
		}
	}
}
