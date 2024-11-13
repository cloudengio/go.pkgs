// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"reflect"
	"slices"
	"strings"
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
		if err := when.Parse(2024, tc.val); err != nil {
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
		{"Feb-29"},
		{"Feb 02"},
		{"Jan Feb"},
		{"13-01"},
	} {
		var md datetime.Date
		if err := md.Parse(2023, tc.val); err == nil {
			t.Errorf("failed to return an error: %v", tc.val)
		}
	}

	var dl datetime.DateList
	if err := dl.Parse(2024, "01/02,02/29,11/4"); err != nil {
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
		{ndl(nd(1, 1), nd(1, 1)), ndrl(year, nd(1, 1), nd(1, 1))},
		{ndl(nd(1, 1), nd(1, 1), nd(1, 1)), ndrl(year, nd(1, 1), nd(1, 1))},
		{ndl(nd(1, 1), nd(1, 2)), ndrl(year, nd(1, 1), nd(1, 2))},
		{ndl(nd(1, 0), nd(1, 2)), ndrl(year, nd(1, 1), nd(1, 2))},
		{ndl(nd(1, 0), nd(1, 0)), ndrl(year, nd(1, 1), nd(1, 1))},
		{ndl(nd(0, 0), nd(0, 0)), ndrl(year, nd(1, 1), nd(1, 1))},
		{ndl(nd(2, 28), nd(2, 29)), ndrl(year, nd(2, 28), nd(2, 29))},
		{ndl(nd(1, 31), nd(2, 1), nd(2, 2)), ndrl(year, nd(1, 31), nd(2, 2))},
		{ndl(nd(1, 1), nd(1, 2), nd(1, 3)), ndrl(year, nd(1, 1), nd(1, 3))},
		{ndl(nd(1, 1), nd(1, 2), nd(3, 4)), ndrl(year, nd(1, 1), nd(1, 2), nd(3, 4), nd(3, 4))},
	} {
		slices.Sort(tc.dates)
		merged := tc.dates.Merge(year)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	year = 2023
	dates := ndl(nd(2, 28), nd(2, 29)) // 2/29 will be treated as 2/28.
	if got, want := dates.Merge(year), ndrl(year, nd(2, 28), nd(2, 28)); !slices.Equal(got, want) {
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

func TestTimeOfDayParse(t *testing.T) {
	for _, tc := range []struct {
		val  string
		when datetime.TimeOfDay
	}{
		{"08:12", datetime.NewTimeOfDay(8, 12, 0)},
		{"08-12", datetime.NewTimeOfDay(8, 12, 0)},
		{"20:01", datetime.NewTimeOfDay(20, 01, 0)},
		{"21-01", datetime.NewTimeOfDay(21, 01, 0)},
		{"08:12:13", datetime.NewTimeOfDay(8, 12, 13)},
		{"08-12-13", datetime.NewTimeOfDay(8, 12, 13)},
		{"20:01:13", datetime.NewTimeOfDay(20, 01, 13)},
		{"21-01-13", datetime.NewTimeOfDay(21, 01, 13)},
	} {
		var tod datetime.TimeOfDay
		if err := tod.Parse(tc.val); err != nil {
			t.Errorf("failed: %v: %v", tc.val, err)
		}
		if !reflect.DeepEqual(tod, tc.when) {
			t.Errorf("got %v, want %v", tod, tc.when)
		}
	}

	for _, tc := range []string{
		"",
		"08:61",
		"08 16",
		"08:61-15",
		"08-61:15",
	} {
		var tod datetime.TimeOfDay
		if err := tod.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}

	tods := datetime.TimeOfDayList{}
	examples := []string{"08:13", "07:13", "09:14:12", "09:14:9", "09:14"}
	for _, s := range examples {
		var tod datetime.TimeOfDay
		if err := tod.Parse(s); err != nil {
			t.Errorf("failed: %v", err)
		}
		tods = append(tods, tod)
	}
	slices.Sort(tods)

	nt := datetime.NewTimeOfDay
	expected := newTimeOfDayList(
		nt(7, 13, 0),
		nt(8, 13, 0),
		nt(9, 14, 0),
		nt(9, 14, 9),
		nt(9, 14, 12))

	if got, want := tods, expected; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	tods = datetime.TimeOfDayList{}
	if err := tods.Parse(strings.Join(examples, ",")); err != nil {
		t.Errorf("failed: %v", err)
	}

	if got, want := tods, expected; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
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
