// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"cloudeng.io/datetime"
)

func TestDateParse(t *testing.T) {
	nd := newDate
	for _, tc := range []struct {
		val  string
		when datetime.Date
	}{
		{"01/02", nd(1, 2)},
		{"1/2", nd(1, 2)},
		{"1/02", nd(1, 2)},
		{"01/2", nd(1, 2)},
		{"Jan-02", nd(1, 2)},
		{"01", nd(1, 0)},
		{"Dec", nd(12, 0)},
		{"Feb-29", nd(2, 29)},
		{"FEB-29", nd(2, 29)},
		{"FeB-29", nd(2, 29)},
	} {
		var when datetime.Date
		if err := when.Parse(tc.val); err != nil {
			t.Errorf("failed: %v: %v", tc.val, err)
			continue
		}
		if got, want := when, tc.when; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	for _, tc := range []struct {
		val string
	}{
		{""},
		{"Jan/31"},
		{"01-02"},
		{"01-02-03"},
		{"Jan-32"},
		{"Feb 02"},
		{"Jan Feb"},
		{"13-01"},
	} {
		var md datetime.Date
		if err := md.Parse(tc.val); err == nil {
			t.Errorf("failed to return an error: %v", tc.val)
		}
	}

	var dl datetime.DateList
	if err := dl.Parse("01/02,02/29,11/4"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if got, want := dl, (datetime.DateList{nd(1, 2), nd(2, 29), nd(11, 4)}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDates(t *testing.T) {
	nd := newDate

	for _, tc := range []struct {
		d  datetime.Date
		dy int
	}{
		{nd(1, 1), 1},
		{nd(2, 2), 31 + 2},
		{nd(3, 1), 31 + 28 + 1},
		{nd(4, 2), 31 + 28 + 31 + 2},
		{nd(5, 2), 31 + 28 + 31 + 30 + 2},
		{nd(6, 2), 31 + 28 + 31 + 30 + 31 + 2},
		{nd(7, 2), 31 + 28 + 31 + 30 + 31 + 30 + 2},
		{nd(8, 2), 31 + 28 + 31 + 30 + 31 + 30 + 31 + 2},
		{nd(9, 2), 31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 2},
		{nd(10, 2), 31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 2},
		{nd(11, 2), 31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 2},
		{nd(12, 2), 31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 2},
	} {
		year := 2023
		yd := tc.d.YearDay(year)

		if got, want := yd.Day(), tc.dy; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := yd.Date(), tc.d; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := yd.CalendarDate(), tc.d.CalendarDate(year); got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}

		year = 2024
		if (tc.d.Month() == 2 && tc.d.Day() >= 29) || tc.d.Month() > 2 {
			tc.dy++
		}
		yd = tc.d.YearDay(year)
		if got, want := yd.Day(), tc.dy; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := yd.Date(), tc.d; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := yd.CalendarDate(), tc.d.CalendarDate(year); got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
	}

}

func TestDatesYearDay(t *testing.T) {
	nd := newDate

	for _, tc := range []struct {
		month, day, year int
		dayOfYear        int
	}{

		{1, 1, 2023, 1},
		{1, 60, 2023, 31},
		{2, 60, 2023, 31 + 28},
		{2, 60, 2024, 31 + 29},
		{12, 60, 2023, 365},
		{12, 60, 2024, 366},
		{1, 0, 2023, 0},
		{3, 0, 2023, 31 + 28},
		{3, 0, 2024, 31 + 29},
	} {

		yd := datetime.NewYearDay(tc.month, tc.dayOfYear)
		if got, want := nd(tc.month, tc.day).DayOfYear(tc.year), yd.Day(); got != want {
			t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
		}
	}

	for _, tc := range []struct {
		year, dayOfYear int
		date            datetime.Date
	}{
		{2023, 0, nd(1, 1)},
		{2023, 366, nd(12, 31)},
		{2024, 0, nd(1, 1)},
		{2024, 367, nd(12, 31)},
		{2024, 4000, nd(12, 31)},
		{2023, 367, nd(12, 31)},
		{2023, 366, nd(12, 31)},
		{2023, 365, nd(12, 31)},
	} {
		yd := datetime.NewYearDay(tc.year, tc.dayOfYear)
		if got, want := yd.Date(), tc.date; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := yd.CalendarDate(), yd.Date().CalendarDate(tc.year); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	for _, tc := range []struct {
		cd, nd, yd datetime.Date
		year       int
	}{
		{nd(1, 1), nd(1, 2), nd(1, 1), 2023},
		{nd(12, 31), nd(1, 1), nd(12, 31), 2023},
		{nd(2, 28), nd(3, 1), nd(2, 28), 2023},
		{nd(2, 28), nd(2, 29), nd(2, 28), 2024},
		{nd(3, 31), nd(4, 1), nd(3, 31), 2023},
		{nd(3, 31), nd(4, 1), nd(3, 31), 2024},
		{nd(2, 28), nd(2, 29), nd(2, 28), 2024},
		{nd(2, 29), nd(3, 1), nd(2, 28), 2023}, // yesterday will return the valid previous day
		{nd(2, 29), nd(3, 1), nd(2, 29), 2024},
	} {
		if got, want := tc.cd.Tomorrow(tc.year), tc.nd; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.cd, tc.year, got, want)
		}
		if got, want := tc.nd.Yesterday(tc.year), tc.yd; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.cd, tc.year, got, want)
		}
	}

	for _, tc := range []struct {
		a, b   datetime.Date
		before bool
	}{
		{nd(1, 1), nd(1, 1), false},
		{nd(3, 1), nd(1, 2), false},
		{nd(2, 28), nd(3, 1), true},
		{nd(2, 29), nd(3, 1), true},
	} {
		if got, want := tc.a < tc.b, tc.before; got != want {
			t.Errorf("%v - %v: got %v, want %v", tc.a, tc.b, got, want)
		}
	}

}

func TestYearAndPlace(t *testing.T) {
	for _, tc := range []struct {
		yp    datetime.YearAndPlace
		isset bool
	}{
		{datetime.YearAndPlace{}, false},
		{datetime.YearAndPlace{2023, nil}, false},
		{datetime.YearAndPlace{0, time.UTC}, false},
		{datetime.YearAndPlace{2025, time.UTC}, true},
	} {
		if got, want := tc.yp.IsSet(), tc.isset; got != want {
			t.Errorf("%v: got %v, want %v", tc.yp, got, want)
		}
	}
}

func TestMergeDates(t *testing.T) {
	nd := newDate
	ndl := newDateList
	ndrl := newDateRangeList
	year := 2024
	for _, tc := range []struct {
		dates  datetime.DateList
		merged datetime.DateRangeList
	}{
		{ndl(nd(1, 1), nd(1, 1)), ndrl(nd(1, 1), nd(1, 1))},
		{ndl(nd(1, 1), nd(1, 1), nd(1, 1)), ndrl(nd(1, 1), nd(1, 1))},
		{ndl(nd(1, 1), nd(1, 2)), ndrl(nd(1, 1), nd(1, 2))},
		{ndl(nd(1, 0), nd(1, 2)), ndrl(nd(1, 1), nd(1, 2))},
		{ndl(nd(1, 0), nd(1, 0)), ndrl(nd(1, 1), nd(1, 1))},
		{ndl(nd(0, 0), nd(0, 0)), ndrl(nd(1, 1), nd(1, 1))},
		{ndl(nd(2, 28), nd(2, 29)), ndrl(nd(2, 28), nd(2, 29))},
		{ndl(nd(1, 31), nd(2, 1), nd(2, 2)), ndrl(nd(1, 31), nd(2, 2))},
		{ndl(nd(1, 1), nd(1, 2), nd(1, 3)), ndrl(nd(1, 1), nd(1, 3))},
		{ndl(nd(1, 1), nd(1, 2), nd(3, 4)), ndrl(nd(1, 1), nd(1, 2), nd(3, 4), nd(3, 4))},
	} {
		slices.Sort(tc.dates)
		merged := tc.dates.Merge(year)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	year = 2023
	dates := ndl(nd(2, 28), nd(2, 29)) // 2/29 will be treated as 2/28.
	if got, want := dates.Merge(year), ndrl(nd(2, 28), nd(2, 28)); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMirrorMonths(t *testing.T) {
	solstice := []int{}
	for i := 1; i <= 12; i++ {
		if datetime.MirrorMonth(datetime.Month(i)) == datetime.Month(i) {
			solstice = append(solstice, i)
		}
	}
	if got, want := solstice, []int{6, 12}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for i := 1; i <= 11; i++ {
		if got, want := datetime.MirrorMonth(datetime.Month(i)), datetime.Month(12-i); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestMonthRangeParse(t *testing.T) {
	months := "Dec,Jan,12,Novem,12"
	var ml datetime.MonthList
	if err := ml.Parse(months); err != nil {
		t.Errorf("failed: %v", err)
	}

	want := datetime.MonthList{
		datetime.Month(1),
		datetime.Month(11),
		datetime.Month(12),
	}
	if got := ml; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, tc := range []string{
		"",
		"Decx",
		"jan,fex",
	} {
		var ml datetime.MonthList
		if err := ml.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}
}

func TestDST(t *testing.T) {

	ncd := datetime.NewCalendarDate
	nd := newDate
	tod := datetime.NewTimeOfDay

	caTZ, _ := time.LoadLocation("America/Los_Angeles")
	saTZ, err := time.LoadLocation("Australia/South")
	if err != nil {
		t.Fatalf("failed to load Australia/South: %v", err)
	}
	for _, tc := range []struct {
		year, month, day int
		isDST            bool
		loc              *time.Location
	}{
		{2024, 3, 9, false, caTZ},
		{2024, 3, 10, true, caTZ},
		{2024, 11, 2, true, caTZ},
		{2024, 11, 3, false, caTZ},
		{2024, 10, 5, false, saTZ},
		{2024, 10, 6, true, saTZ},
		{2025, 4, 5, true, saTZ},
		{2025, 4, 6, false, saTZ},
	} {
		if got, want := ncd(tc.year, datetime.Month(tc.month), tc.day).IsDST(tc.loc), tc.isDST; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	for i, tc := range []struct {
		fromDate                 datetime.Date
		fromTime                 datetime.TimeOfDay
		toDate                   datetime.Date
		toTime                   datetime.TimeOfDay
		year                     int
		loc                      *time.Location
		same, stdToDST, dstToStd bool
	}{
		{nd(1, 1), tod(11, 0, 0), nd(1, 1), tod(11, 1, 0), 2024, caTZ, true, false, false},
		{nd(3, 10), tod(1, 59, 58), nd(3, 10), tod(2, 59, 59), 2024, caTZ, true, false, false}, {nd(3, 10), tod(2, 59, 58), nd(3, 10), tod(2, 59, 59), 2024, caTZ, true, false, false},
		{nd(3, 10), tod(1, 59, 59), nd(3, 10), tod(3, 0, 0), 2024, caTZ, false, true, false},
		{nd(3, 11), tod(1, 59, 59), nd(3, 11), tod(3, 0, 0), 2024, caTZ, true, false, false},
		{nd(11, 3), tod(1, 59, 59), nd(11, 3), tod(2, 0, 0), 2024, caTZ, false, false, true},
		{nd(11, 3), tod(2, 59, 59), nd(11, 3), tod(3, 0, 0), 2024, caTZ, true, false, false},
		{nd(3, 10), tod(1, 59, 59), nd(3, 10), tod(3, 0, 0), 2024, saTZ, true, false, false},
		{nd(11, 3), tod(1, 59, 59), nd(11, 3), tod(2, 0, 0), 2024, saTZ, true, false, false},
		{nd(10, 6), tod(1, 59, 59), nd(10, 6), tod(3, 0, 0), 2024, saTZ, false, true, false},
		{nd(4, 6), tod(1, 59, 59), nd(4, 6), tod(3, 0, 0), 2025, saTZ, false, false, true},
	} {
		yp := datetime.YearAndPlace{Year: tc.year, Place: tc.loc}

		now := datetime.Time(yp, tc.fromDate, tc.fromTime)
		same, stdToDST, dstToStd := datetime.DSTTransition(yp, now, tc.toDate, tc.toTime)
		if got, want := same, tc.same; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := stdToDST, tc.stdToDST; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := dstToStd, tc.dstToStd; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}

	}
}
