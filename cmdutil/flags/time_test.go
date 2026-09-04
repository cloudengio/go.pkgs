// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
	"cloudeng.io/cmdutil/flags"
)

func TestTimeFlag(t *testing.T) {
	tp := func(f, v string) time.Time {
		tv, err := time.Parse(f, v)
		if err != nil {
			t.Fatal(err)
		}
		return tv
	}

	for i, tc := range []struct {
		in     string
		format string
	}{
		{"2021-10-10", time.DateOnly},
		{"2021-10-10T03:03:03-07:00", time.RFC3339},
		{"03:03:05", time.TimeOnly},
		{"2021-10-10 03:03:05", time.DateTime},
	} {
		tf := &flags.Time{}
		if err := tf.Set(tc.in); err != nil {
			t.Errorf("%v: %v", i, err)
		}
		if got, want := tf.Get().(time.Time), tp(tc.format, tc.in); !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestTimeFlagErrors(t *testing.T) {
	for _, in := range []string{"not-a-time", "", "2021-13-45", "25:00:00"} {
		tf := &flags.Time{}
		err := tf.Set(in)
		if err == nil {
			t.Errorf("%q: expected an error, got %v", in, tf.Time())
			continue
		}
		if !strings.Contains(err.Error(), "invalid time") {
			t.Errorf("%q: got %v, want it to report an invalid time", in, err)
		}
		// A rejected value leaves the flag unset.
		if !tf.IsDefault() {
			t.Errorf("%q: a rejected value should leave the flag unset", in)
		}
	}
}

// TestTimeFlagMatchesCmdtypes verifies that the flag parses exactly as
// cmdtypes.FlexTime does, so a time given on the command line is the same
// value as one read from JSON or YAML.
func TestTimeFlagMatchesCmdtypes(t *testing.T) {
	for _, in := range []string{
		"2021-10-10",
		"2021-10-10T03:03:03-07:00",
		"03:03:05",
		"2021-10-10 03:03:05",
	} {
		tf := &flags.Time{}
		if err := tf.Set(in); err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		want, err := cmdtypes.ParseFlexTime(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got := tf.FlexTime(); got != want {
			t.Errorf("%q: got %v, want %v", in, got.Time(), want.Time())
		}
		if got := tf.Time(); !got.Equal(want.Time()) {
			t.Errorf("%q: got %v, want %v", in, got, want.Time())
		}
		// String reports the value as supplied, so a default keeps the
		// format it was written in.
		if got := tf.String(); got != in {
			t.Errorf("%q: String returned %q", in, got)
		}
		if tf.IsDefault() {
			t.Errorf("%q: a set value should not report itself as the default", in)
		}
	}
}
