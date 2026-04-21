// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"flag"
	"testing"

	"cloudeng.io/cmdutil"
)

func newParsedFlagSet(t *testing.T, args []string) (*flag.FlagSet, *string, *int, *bool) {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	s := fs.String("str", "default", "a string flag")
	n := fs.Int("num", 0, "an int flag")
	b := fs.Bool("bool", false, "a bool flag")
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return fs, s, n, b
}

func TestIsExplicitlySetBasic(t *testing.T) {
	fs, _, _, _ := newParsedFlagSet(t, []string{"-str=hello", "-bool"})

	if !cmdutil.IsExplicitlySet(fs, "str") {
		t.Error("str: expected explicitly set")
	}
	if !cmdutil.IsExplicitlySet(fs, "bool") {
		t.Error("bool: expected explicitly set")
	}
	if cmdutil.IsExplicitlySet(fs, "num") {
		t.Error("num: expected not explicitly set (only default)")
	}
}

func TestIsExplicitlySetNoneProvided(t *testing.T) {
	fs, _, _, _ := newParsedFlagSet(t, nil)

	for _, name := range []string{"str", "num", "bool"} {
		if cmdutil.IsExplicitlySet(fs, name) {
			t.Errorf("%s: expected not explicitly set when no args parsed", name)
		}
	}
}

func TestIsExplicitlySetAllProvided(t *testing.T) {
	fs, _, _, _ := newParsedFlagSet(t, []string{"-str=x", "-num=7", "-bool=true"})

	for _, name := range []string{"str", "num", "bool"} {
		if !cmdutil.IsExplicitlySet(fs, name) {
			t.Errorf("%s: expected explicitly set", name)
		}
	}
}

func TestIsExplicitlySetMatchesDefault(t *testing.T) {
	// Setting a flag to its default value still counts as explicitly set.
	fs, _, _, _ := newParsedFlagSet(t, []string{"-str=default", "-num=0"})

	if !cmdutil.IsExplicitlySet(fs, "str") {
		t.Error("str set to its default value should still be explicitly set")
	}
	if !cmdutil.IsExplicitlySet(fs, "num") {
		t.Error("num set to its default value should still be explicitly set")
	}
}

func TestIsExplicitlySetUnknownName(t *testing.T) {
	fs, _, _, _ := newParsedFlagSet(t, []string{"-str=val"})

	if cmdutil.IsExplicitlySet(fs, "nonexistent") {
		t.Error("nonexistent flag should never be reported as explicitly set")
	}
}

func TestIsExplicitlySetBeforeParse(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("str", "default", "usage")

	// Visit on an unparsed FlagSet visits nothing.
	if cmdutil.IsExplicitlySet(fs, "str") {
		t.Error("str: should not be set before Parse is called")
	}
}
