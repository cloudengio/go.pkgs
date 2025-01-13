// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"slices"
	"testing"
	"time"

	"cloudeng.io/datetime"
	"cloudeng.io/datetime/schedule"
)

type schoolHolidays struct {
}

func (sh schoolHolidays) Name() string {
	return "school-holidays"
}

func (sh schoolHolidays) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(
		datetime.NewCalendarDate(year, 6, 10),
		datetime.NewCalendarDate(year, 8, 20))
}

func TestDynamicDates(t *testing.T) {
	nd := datetime.NewDate
	ndr := datetime.NewDateRange
	d := schedule.Dates{
		Months:  datetime.MonthList{1, 2},
		Dynamic: []datetime.DynamicDateRange{schoolHolidays{}},
		Ranges: datetime.DateRangeList{
			ndr(
				datetime.NewDate(9, 1), datetime.NewDate(9, 10))},
	}
	e := d.EvaluateDateRanges(2024, datetime.DateRangeYear())
	if got, want := e, (datetime.DateRangeList{
		ndr(nd(1, 1), nd(2, 29)),
		ndr(nd(6, 10), nd(8, 20)),
		ndr(nd(9, 1), nd(9, 10))}); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type DynamicTimeOfDay struct {
	name string
	val  datetime.TimeOfDay
}

func (d DynamicTimeOfDay) Name() string {
	return d.name
}

func (d DynamicTimeOfDay) Evaluate(_ datetime.CalendarDate, _ datetime.Place) datetime.TimeOfDay {
	return d.val
}

func TestDynamic(t *testing.T) {
	breakfast := DynamicTimeOfDay{name: "breakfast", val: datetime.NewTimeOfDay(8, 0, 0)}
	a := schedule.ActionSpecs[int]{
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "a", T: 1},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "b", T: 2},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 3},
		{Dynamic: schedule.DynamicTimeOfDaySpec{Due: breakfast, Offset: time.Minute * 30}, Name: "d", T: 4},
	}
	b := a.Evaluate(datetime.NewCalendarDate(2024, 1, 1), datetime.Place{TimeLocation: time.Local})
	b.Sort()

	if got, want := b, []schedule.ActionSpec[int]{
		{Due: datetime.NewTimeOfDay(8, 30, 0), Name: "d", T: 4},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 3},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "b", T: 2},
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "a", T: 1},
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
