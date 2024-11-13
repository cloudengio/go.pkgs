// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"testing"
	"time"

	"cloudeng.io/datetime"
)

func TestConstraints(t *testing.T) {
	nd := newDate
	ncd := newCalendarDate
	md := func(y, m, d int) time.Time {
		return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	}

	ct := func(weekday, weekend bool, custom ...datetime.Date) datetime.Constraints {
		return datetime.Constraints{
			Weekdays: weekday,
			Weekends: weekend,
			Custom:   custom,
		}
	}

	ctc := func(weekday, weekend bool, custom ...datetime.CalendarDate) datetime.Constraints {
		return datetime.Constraints{
			Weekdays:       weekday,
			Weekends:       weekend,
			CustomCalendar: custom,
		}
	}

	for i, tc := range []struct {
		when       time.Time
		constraint datetime.Constraints
		result     bool
	}{
		{md(2024, 1, 2), ct(false, false), true},

		{md(2024, 1, 2), ct(true, false), true},
		{md(2024, 1, 3), ct(true, false), true},
		{md(2024, 1, 4), ct(true, false), true},
		{md(2024, 1, 5), ct(true, false), true},
		{md(2024, 1, 6), ct(true, false), false},
		{md(2024, 1, 7), ct(true, false), false},

		{md(2024, 1, 3), ct(false, true), false},
		{md(2024, 1, 4), ct(false, true), false},
		{md(2024, 1, 5), ct(false, true), false},
		{md(2024, 1, 6), ct(false, true), true},
		{md(2024, 1, 7), ct(false, true), true},

		{md(2024, 1, 2), ct(false, false, nd(1, 2)), false},
		{md(2024, 1, 3), ct(false, false, nd(1, 2)), true},
		{md(2024, 3, 4), ct(false, false, nd(1, 2), nd(3, 4)), false},
		{md(2024, 2, 5), ct(false, false, nd(1, 2), nd(3, 4)), true},

		{md(2024, 3, 4), ctc(false, false, ncd(2024, 1, 2), ncd(2024, 3, 4)), false},
		{md(2024, 3, 4), ctc(false, false, ncd(2023, 1, 2), ncd(2023, 3, 4)), true},
		{md(2024, 2, 5), ctc(false, false, ncd(2024, 1, 2), ncd(2024, 3, 4)), true},
	} {
		if got, want := tc.constraint.Include(tc.when), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}

	// todo(cnicolaou): test with calendar dates also.
}
