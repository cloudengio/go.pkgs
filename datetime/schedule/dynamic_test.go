// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"slices"
	"testing"

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
		For:     datetime.MonthList{1, 2},
		Dynamic: []datetime.DynamicDateRange{schoolHolidays{}},
		Ranges: datetime.DateRangeList{
			ndr(
				datetime.NewDate(9, 1), datetime.NewDate(9, 10))},
	}
	e := d.EvaluateDateRanges(2024)
	if got, want := e, (datetime.DateRangeList{
		ndr(nd(1, 1), nd(2, 29)),
		ndr(nd(6, 10), nd(8, 20)),
		ndr(nd(9, 1), nd(9, 10))}); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

/*
type sunSetRise struct {
	rise bool
}

func (sr sunSetRise) Name() string {
	if sr.rise {
		return "sunrise"
	}
	return "sunset"
}

func (sr sunSetRise) Evaluate(cd datetime.CalendarDate, _ *time.Location) datetime.TimeOfDay {
	if sr.rise {
		return datetime.NewTimeOfDay(7, 12, 13)
	}
	return datetime.NewTimeOfDay(17, 0, 33)
}

func TestDynamicActions(t *testing.T) {
	sunset := sunSetRise{rise: false}
	sunrise := sunSetRise{rise: true}

	ta := schedule.Action[int]{DueDynamic: sunrise, Name: "1", Action: 1}

	if got, want := ta.Evaluate(datetime.NewCalendarDate(2024, 1, 1), nil), datetime.NewTimeOfDay(7, 12, 13); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	tl := schedule.Actions[int]{
		ta,
		{DueDynamic: sunset, Name: "2", Action: 2},
		{Due: datetime.NewTimeOfDay(12, 0, 0), Name: "3", Action: 3},
		{DueDynamic: sunset, Name: "4", Action: 4},
	}

	tl.Evaluate(datetime.NewCalendarDate(2024, 1, 1), nil)

	if got, want := tl, (schedule.Actions[int]{
		{Due: datetime.NewTimeOfDay(7, 12, 13), DueDynamic: sunrise, Name: "1", Action: 1},
		{Due: datetime.NewTimeOfDay(17, 0, 33), DueDynamic: sunset, Name: "2", Action: 2},
		{Due: datetime.NewTimeOfDay(12, 0, 0), Name: "3", Action: 3},
		{Due: datetime.NewTimeOfDay(17, 0, 33), DueDynamic: sunset, Name: "4", Action: 4},
	}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
*/
