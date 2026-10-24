// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/envfile"
)

func TestExpandingMapping(t *testing.T) {
	t.Setenv("EXPAND_TEST_ENV", "from-env")
	t.Setenv("EXPAND_TEST_BOTH", "env-wins")
	t.Setenv("EXPAND_TEST_DOLLAR", "pa$$word")
	vars, err := envfile.Read(strings.NewReader(strings.Join([]string{
		"EXPAND_TEST_FILE=from-file",
		"EXPAND_TEST_BOTH=file-loses",
		"EXPAND_TEST_REF=$EXPAND_TEST_ENV/${EXPAND_TEST_FILE}/x",
		"EXPAND_TEST_UNRESOLVED=a/${EXPAND_TEST_NOPE}/b",
		"EXPAND_TEST_SELF=$EXPAND_TEST_SELF",
		"EXPAND_TEST_DOLLAR=file",
	}, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	m := envfile.ExpandingMapping(vars)
	for _, tc := range []struct{ name, want string }{
		{"EXPAND_TEST_ENV", "from-env"},
		{"EXPAND_TEST_FILE", "from-file"},
		{"EXPAND_TEST_BOTH", "env-wins"},
		// References in a value from the file are expanded.
		{"EXPAND_TEST_REF", "from-env/from-file/x"},
		// An unresolved reference is left visible.
		{"EXPAND_TEST_UNRESOLVED", "a/${EXPAND_TEST_NOPE}/b"},
		// Values from the environment are not expanded.
		{"EXPAND_TEST_DOLLAR", "pa$$word"},
		{"EXPAND_TEST_UNDEFINED", ""},
	} {
		if got := m(tc.name); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.name, got, tc.want)
		}
	}
	// A definition that refers to itself terminates.
	if got := m("EXPAND_TEST_SELF"); !strings.Contains(got, "EXPAND_TEST_SELF") {
		t.Errorf("got %q", got)
	}
}

func TestExpandingMappingNil(t *testing.T) {
	t.Setenv("EXPAND_TEST_ENV", "v")
	m := envfile.ExpandingMapping(nil)
	if got := m("EXPAND_TEST_ENV"); got != "v" {
		t.Errorf("got %q", got)
	}
	if got := m("EXPAND_TEST_UNDEFINED"); got != "" {
		t.Errorf("got %q", got)
	}
}
