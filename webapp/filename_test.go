// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"testing"

	"cloudeng.io/webapp"
)

func TestSafePaths(t *testing.T) {
	for _, tc := range []struct {
		path string
		err  string
	}{
		{"../y", "contains relative path components"},
		{"./y", "contains relative path components"},
		{"x/../y", "contains relative path components"},
		{"x/./y", "contains relative path components"},
		{"x/?/y", "contains unix reserved characters"},
		{"con", "contains windows reserved characters"},
		{"a/b", "contains unix reserved characters"},
	} {
		err := webapp.SafePath(tc.path)
		if got, want := err.Error(), tc.err; got != want {
			t.Errorf("%v: got %v, want %v", tc.path, got, want)
		}
	}
}

func TestSafePathsGemini(t *testing.T) {
	for _, tc := range []struct {
		path string
		err  string // empty string means no error expected
	}{
		// Invalid paths - should return errors
		{"../y", "contains relative path components"},
		{"./y", "contains relative path components"},
		{"x/../y", "contains relative path components"},
		{"x/./y", "contains relative path components"},
		{"...", "contains relative path components"},
		{"x/.../y", "contains relative path components"},
		{"x/?/y", "contains unix reserved characters"},
		{"file:text.txt", "contains unix reserved characters"},
		{"file<.txt", "contains unix reserved characters"},
		{"file>.txt", "contains unix reserved characters"},
		{"file\".txt", "contains unix reserved characters"},
		{"file|.txt", "contains unix reserved characters"},
		{"file*.txt", "contains unix reserved characters"},
		{"\u0000file.txt", "contains control characters"},
		{"\u001ftext.txt", "contains control characters"},
		{"\u0080file.txt", "contains control characters"},
		{"\u009ftext.txt", "contains control characters"},
		{"con", "contains windows reserved characters"},
		{"prn", "contains windows reserved characters"},
		{"aux", "contains windows reserved characters"},
		{"nul", "contains windows reserved characters"},
		{"com1", "contains windows reserved characters"},
		{"COM9", "contains windows reserved characters"},
		{"lpt1", "contains windows reserved characters"},
		{"LPT9", "contains windows reserved characters"},

		// Valid paths - should not return errors
		{"file-123", ""},
		{"file_123", ""},
		{"file 123", ""},
		{"file+123", ""},
		{"file,123", ""},
		{"file;123", ""},
		{"file=123", ""},
		{"file@123", ""},
		{"mycon", ""},
		{"привет", ""}, // Unicode is fine as long as it's not control chars
	} {
		err := webapp.SafePath(tc.path)

		if tc.err == "" {
			if err != nil {
				t.Errorf("%q: expected no error, but got: %v", tc.path, err)
			}
		} else {
			if err == nil {
				t.Errorf("%q: expected error %q, but got nil", tc.path, tc.err)
				continue
			}
			if got, want := err.Error(), tc.err; got != want {
				t.Errorf("%q: got error %q, want %q", tc.path, got, want)
			}
		}
	}
}

func TestSafePathsClaude(t *testing.T) {
	for _, tc := range []struct {
		path string
		err  string // empty string means no error expected
	}{
		// Invalid paths - should return errors
		{"../y", "contains relative path components"},
		{"./y", "contains relative path components"},
		{"x/../y", "contains relative path components"},
		{"x/./y", "contains relative path components"},
		{"...", "contains relative path components"},
		{"x/.../y", "contains relative path components"},
		{"x/?/y", "contains unix reserved characters"},
		{"file:text.txt", "contains unix reserved characters"},
		{"file<.txt", "contains unix reserved characters"},
		{"file>.txt", "contains unix reserved characters"},
		{"file:.txt", "contains unix reserved characters"},
		{"file\".txt", "contains unix reserved characters"},
		{"file|.txt", "contains unix reserved characters"},
		{"file*.txt", "contains unix reserved characters"},
		{"\u0000file.txt", "contains control characters"},
		{"\u001ftext.txt", "contains control characters"},
		{"\u0080file.txt", "contains control characters"},
		{"\u009ftext.txt", "contains control characters"},
		{"con", "contains windows reserved characters"},
		{"prn", "contains windows reserved characters"},
		{"aux", "contains windows reserved characters"},
		{"nul", "contains windows reserved characters"},
		{"com1", "contains windows reserved characters"},
		{"COM9", "contains windows reserved characters"},
		{"lpt1", "contains windows reserved characters"},
		{"LPT9", "contains windows reserved characters"},

		// Valid paths - should not return errors
		{"file.txt", ""},
		// {"path/to/file.txt", ""}, should not be allowed.
		{"file-123.txt", ""},
		{"file_123.txt", ""},
		{"file 123.txt", ""},
		{"file+123.txt", ""},
		{"file,123.txt", ""},
		{"file;123.txt", ""},
		{"file=123.txt", ""},
		{"file@123.txt", ""},
		{"con.txt", ""}, // Only exact matches of reserved names should fail
		{"mycon", ""},
		{"привет.txt", ""}, // Unicode is fine as long as it's not control chars
	} {
		err := webapp.SafePath(tc.path)

		if tc.err == "" {
			// Expect no error
			if err != nil {
				t.Errorf("%q: expected no error, but got %v", tc.path, err)
			}
		} else {
			// Expect specific error
			if err == nil {
				t.Errorf("%q: expected error %q, but got nil", tc.path, tc.err)
				continue
			}
			if got, want := err.Error(), tc.err; got != want {
				t.Errorf("%q: got error %q, want %q", tc.path, got, want)
			}
		}
	}
}
