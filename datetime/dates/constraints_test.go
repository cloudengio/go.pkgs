// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates_test

import (
	"testing"
	"time"

	"cloudeng.io/datetime/dates"
)

func TestConstraints(t *testing.T) {
	dt := newDate
	md := func(m, d int) time.Time {
		return time.Date(2024, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	}
	ct := func(weekday, weekend, exclude bool, custom ...dates.Date) dates.Constraints {
		return dates.Constraints{
			Weekdays: weekday,
			Weekends: weekend,
			Custom:   custom,
		}
	}

	for i, tc := range []struct {
		when       time.Time
		constraint dates.Constraints
		result     bool
	}{
		{md(1, 2), ct(false, false, false), true},
		{md(1, 2), ct(false, false, true), true},

		{md(1, 2), ct(true, false, false), true},
		{md(1, 3), ct(true, false, false), true},
		{md(1, 4), ct(true, false, false), true},
		{md(1, 5), ct(true, false, false), true},
		{md(1, 6), ct(true, false, false), false},
		{md(1, 7), ct(true, false, false), false},

		{md(1, 3), ct(false, true, false), false},
		{md(1, 4), ct(false, true, false), false},
		{md(1, 5), ct(false, true, false), false},
		{md(1, 6), ct(false, true, false), true},
		{md(1, 7), ct(false, true, false), true},

		{md(1, 2), ct(false, false, false, dt(1, 2)), true},
		{md(1, 3), ct(false, false, false, dt(1, 2)), false},
		{md(3, 4), ct(false, false, false, dt(1, 2), dt(3, 4)), true},
		{md(3, 4), ct(false, false, true, dt(1, 2), dt(3, 4)), false},
		{md(2, 5), ct(false, false, true, dt(1, 2), dt(3, 4)), true},
	} {
		if got, want := tc.constraint.Include(tc.when), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	// todo(cnicolaou): test with calendar dates also.
}
