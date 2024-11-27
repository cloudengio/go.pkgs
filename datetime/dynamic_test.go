// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"testing"
	"time"

	"cloudeng.io/datetime"
)

type spring struct {
}

func (s *spring) Name() string {
	return "spring"
}

func (s *spring) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(
		datetime.NewCalendarDate(year, 3, 21),
		datetime.NewCalendarDate(year, 6, 21))
}

func TestDynamicDates(t *testing.T) {
	// exclude the spring.
	c := datetime.Constraints{
		Dynamic: []datetime.DynamicDateRange{&spring{}},
	}
	nd := datetime.NewDate
	ndr := datetime.NewDateRange

	for d := range ndr(nd(3, 21), nd(6, 21)).Dates(2024) {
		when := time.Date(d.Year(), time.Month(d.Month()), d.Day(), 0, 0, 0, 0, time.UTC)
		if c.Include(when) {
			t.Errorf("spring date included: %v", when)
		}
	}

	for _, tc := range []struct {
		m datetime.Month
		d int
	}{
		{1, 1},
		{11, 24},
		{11, 25},
	} {
		when := time.Date(2024, time.Month(tc.m), tc.d, 0, 0, 0, 0, time.UTC)
		if got, want := c.Include(when), true; got != want {
			t.Errorf("%v-%v: got %v, want %v", tc.m, tc.d, got, want)
		}
	}
}
