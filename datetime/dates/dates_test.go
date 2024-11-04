// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cloudeng.io/datetime/dates"
)

func TestDateParse(t *testing.T) {
	nd := newDate
	for _, tc := range []struct {
		val  string
		when dates.Date
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
		var when dates.Date
		if err := when.Parse(2024, tc.val); err != nil {
			t.Errorf("failed: %v: %v", tc.val, err)
		}
		if !reflect.DeepEqual(when, tc.when) {
			t.Errorf("got %v, want %v", when, tc.when)
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
		var md dates.Date
		if err := md.Parse(2023, tc.val); err == nil {
			t.Errorf("failed to return an error: %v", tc.val)
		}
	}

	var dl dates.DateList
	if err := dl.Parse(2024, "01/02,02/29,11/4"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if got, want := dl, (dates.DateList{nd(1, 2), nd(2, 29), nd(11, 4)}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDates(t *testing.T) {
	nd := newDate

	for _, tc := range []struct {
		d  dates.Date
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
		dayOf := tc.d.DayOfYear(year)
		if got, want := dayOf, tc.dy; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := dates.DateFromDay(year, dayOf), tc.d; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		year = 2024
		if (tc.d.Month == 2 && tc.d.Day >= 29) || tc.d.Month > 2 {
			tc.dy++
		}
		dayOf = tc.d.DayOfYear(year)
		if got, want := dayOf, tc.dy; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
		if got, want := dates.DateFromDay(year, dayOf), tc.d; got != want {
			t.Errorf("%v (%v): got %v, want %v", tc.d, year, got, want)
		}
	}

	if got, want := nd(1, 60).DayOfYear(2023), 31; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(2, 60).DayOfYear(2023), 31+28; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(2, 60).DayOfYear(2024), 31+29; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(12, 60).DayOfYear(2023), 365; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(12, 60).DayOfYear(2024), 366; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(1, 0).DayOfYear(2023), 0; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(3, 0).DayOfYear(2023), 31+28; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}
	if got, want := nd(3, 0).DayOfYear(2024), 31+29; got != want {
		t.Errorf("%v: got %v, want %v", nd(1, 1), got, want)
	}

	if got, want := dates.DateFromDay(2023, 0), nd(1, 1); got != want {
		t.Errorf("%v: got %v, want %v", 2023, got, want)
	}
	if got, want := dates.DateFromDay(2023, 366), nd(12, 31); got != want {
		t.Errorf("%v: got %v, want %v", 2023, got, want)
	}

	if got, want := dates.DateFromDay(2024, 0), nd(1, 1); got != want {
		t.Errorf("%v: got %v, want %v", 2024, got, want)
	}
	if got, want := dates.DateFromDay(2024, 367), nd(12, 31); got != want {
		t.Errorf("%v: got %v, want %v", 2024, got, want)
	}

	for _, tc := range []struct {
		cd, nd, yd dates.Date
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
		a, b   dates.Date
		before bool
	}{
		{nd(1, 1), nd(1, 1), false},
		{nd(3, 1), nd(1, 2), false},
		{nd(2, 28), nd(3, 1), true},
		{nd(2, 29), nd(3, 1), true},
	} {
		if got, want := tc.a.Before(tc.b), tc.before; got != want {
			t.Errorf("%v - %v: got %v, want %v", tc.a, tc.b, got, want)
		}
	}

}

func TestYearAndPlace(t *testing.T) {
	for _, tc := range []struct {
		yp    dates.YearAndPlace
		isset bool
	}{
		{dates.YearAndPlace{}, false},
		{dates.YearAndPlace{2023, nil}, false},
		{dates.YearAndPlace{0, time.UTC}, false},
		{dates.YearAndPlace{2025, time.UTC}, true},
	} {
		if got, want := tc.yp.IsSet(), tc.isset; got != want {
			t.Errorf("%v: got %v, want %v", tc.yp, got, want)
		}
	}
}

func daysFromDatesString(year int, datelist string) []int {
	parts := strings.Split(datelist, ",")
	days := make([]int, 0, len(parts))
	for _, p := range parts {
		var date dates.Date
		if err := date.Parse(year, p); err != nil {
			panic(err)
		}
		days = append(days, date.DayOfYear(year))
	}
	return days
}

func TestMergeDatesAndRanges(t *testing.T) {
	nd := newDate
	ndr := newDateRange
	year := 2024
	for _, tc := range []struct {
		dates  []dates.Date
		merged dates.DateRangeList
	}{
		{[]dates.Date{nd(1, 1), nd(1, 1)}, ndr(nd(1, 1), nd(1, 1))},
		{[]dates.Date{nd(1, 1), nd(1, 2)}, ndr(nd(1, 1), nd(1, 2))},
		{[]dates.Date{nd(1, 31), nd(2, 1), nd(2, 2)}, ndr(nd(1, 31), nd(2, 2))},
		{[]dates.Date{nd(1, 1), nd(1, 2), nd(1, 3)}, ndr(nd(1, 1), nd(1, 3))},
		{[]dates.Date{nd(1, 1), nd(1, 2), nd(3, 4)}, ndr(nd(1, 1), nd(1, 2), nd(3, 4), nd(3, 4))},
	} {
		merged := dates.MergeDates(year, tc.dates)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	for _, tc := range []struct {
		ranges dates.DateRangeList
		merged dates.DateRangeList
	}{
		{ndr(nd(1, 1), nd(1, 2)), ndr(nd(1, 1), nd(1, 2))},
		{ndr(nd(1, 1), nd(1, 2), nd(1, 1), nd(1, 2)), ndr(nd(1, 1), nd(1, 2))},
		{ndr(nd(1, 1), nd(1, 2), nd(1, 3), nd(1, 10)), ndr(nd(1, 1), nd(1, 10))},
		{ndr(nd(1, 1), nd(1, 2), nd(1, 4), nd(1, 10)), ndr(nd(1, 1), nd(1, 2), nd(1, 4), nd(1, 10))},
		{ndr(nd(2, 27), nd(2, 29), nd(3, 1), nd(3, 10)), ndr(nd(2, 27), nd(3, 10))},
	} {
		merged := dates.MergeRanges(year, tc.ranges)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestMirrorMonths(t *testing.T) {
	solstice := []int{}
	for i := 1; i <= 12; i++ {
		if dates.MirrorMonth(dates.Month(i)) == dates.Month(i) {
			solstice = append(solstice, i)
		}
	}
	if got, want := solstice, []int{6, 12}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for i := 1; i <= 11; i++ {
		if got, want := dates.MirrorMonth(dates.Month(i)), dates.Month(12-i); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestTimeOfDayParse(t *testing.T) {
	for _, tc := range []struct {
		val  string
		when dates.TimeOfDay
	}{
		{"08:12", dates.TimeOfDay{8, 12, 0}},
		{"08-12", dates.TimeOfDay{8, 12, 0}},
		{"20:01", dates.TimeOfDay{20, 01, 0}},
		{"21-01", dates.TimeOfDay{21, 01, 0}},
		{"08:12:13", dates.TimeOfDay{8, 12, 13}},
		{"08-12-13", dates.TimeOfDay{8, 12, 13}},
		{"20:01:13", dates.TimeOfDay{20, 01, 13}},
		{"21-01-13", dates.TimeOfDay{21, 01, 13}},
	} {
		var tod dates.TimeOfDay
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
		var tod dates.TimeOfDay
		if err := tod.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}

	tods := dates.TimeOfDayList{}
	for _, s := range []string{"08:13", "07:13", "09:14:12", "09:14:9", "09:14"} {
		var tod dates.TimeOfDay
		if err := tod.Parse(s); err != nil {
			t.Errorf("failed: %v", err)
		}
		tods = append(tods, tod)
	}
	tods.Sort()

	if got, want := tods, []dates.TimeOfDay{{7, 13, 0}, {8, 13, 0}, {9, 14, 0}, {9, 14, 9}, {9, 14, 12}}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMonthRangeParse(t *testing.T) {
	months := "Dec,Jan,12,Novem,12"
	var ml dates.MonthList
	if err := ml.Parse(months); err != nil {
		t.Errorf("failed: %v", err)
	}

	want := dates.MonthList{
		dates.Month(1),
		dates.Month(11),
		dates.Month(12),
	}
	if got := ml; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, tc := range []string{
		"",
		"Decx",
		"jan,fex",
	} {
		var ml dates.MonthList
		if err := ml.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}
}

func stringSlice(s ...string) []string {
	return s
}

func newDateRange(d ...dates.Date) []dates.DateRange {
	r := make([]dates.DateRange, 0, len(d)/2)
	for i := 0; i < len(d); i += 2 {
		r = append(r, dates.DateRange{d[i], d[i+1]})
	}
	return r
}

func newDate(m, d int) dates.Date {
	return dates.Date{
		Month: dates.Month(m),
		Day:   d,
	}
}

type dateList []dates.Date

func (dr *dateList) String() string {
	var out strings.Builder
	for _, d := range *dr {
		out.WriteString(fmt.Sprintf("%02d/%02d,", d.Month, d.Day))
	}
	if out.Len() == 0 {
		return ""
	}
	return out.String()[:out.Len()-1]
}

func TestMonthAndRangeMerge(t *testing.T) {
	nd := newDate
	dr := newDateRange
	sl := stringSlice
	year := 2021
	for _, tc := range []struct {
		months   string
		ranges   []string
		expected dates.DateRangeList
	}{
		{"jan,dec",
			nil,
			dr(nd(1, 1), nd(1, 31), nd(12, 1), nd(12, 31))},
		{"",
			sl("aug-02:sep-03", "jan-01:jan-02"),
			dr(nd(1, 1), nd(1, 2), nd(8, 2), nd(9, 3))},
		{"feb,apr",
			sl("aug-02:sep-03", "jan-01:jan-02"),
			dr(nd(1, 1), nd(1, 2), nd(2, 1), nd(2, 28), nd(4, 1), nd(4, 30), nd(8, 2), nd(9, 3))},
	} {
		var months dates.MonthList
		var ranges dates.DateRangeList
		if len(tc.months) > 0 {
			if err := months.Parse(tc.months); err != nil {
				t.Errorf("failed: %v", err)
			}
		}
		if len(tc.ranges) > 0 {
			if err := ranges.Parse(year, tc.ranges); err != nil {
				t.Errorf("failed: %v", err)
			}
		}
		merged := dates.MergeMonthsAndRanges(year, months, ranges)
		if got, want := merged, tc.expected; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
