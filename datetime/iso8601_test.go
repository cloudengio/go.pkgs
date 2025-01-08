// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"testing"
	"time"

	"cloudeng.io/datetime"
)

func TestISO8601Duration(t *testing.T) {
	year := time.Hour * 24 * 365
	month := (time.Hour * 24 * 365) / 12
	week := time.Hour * 24 * 7
	day := time.Hour * 24
	for i, tc := range []struct {
		input  string
		output time.Duration
	}{
		{"P", 0},
		{"-P", 0},
		{"P1Y", year},
		{"-P1Y", -year},
		{"P1M", month},
		{"P1W", week},
		{"P1D", day},
		{"PT1H", time.Hour},
		{"PT1M", time.Minute},
		{"PT1.5M", time.Minute + 30*time.Second},
		{"PT10S", time.Second * 10},
		{"PT1.5S", time.Second + 500*time.Millisecond},
		{"P2MT1M", month*2 + (time.Minute)},
		{"P1Y1M1W1DT1H1M1S", year + month + week + day + time.Hour + time.Minute + time.Second},
	} {
		d, err := datetime.ParseISO8601Duration(tc.input)
		if err != nil {
			t.Errorf("%v: %v", i, err)
			continue
		}
		if got, want := d, tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
}
