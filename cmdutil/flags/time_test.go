// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"testing"
	"time"

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
