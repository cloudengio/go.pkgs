// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/datetime"
)

func TestDateRangeParse(t *testing.T) {
	ranges := []string{
		"01/02:03/04",
		"Jan-02:Mar-04",
		"01:03",
		"01:03",
		"Jan:Mar",
		"january:Mar",
		"nov:dec",
		"nov:dec-20",
		"feb:feb",
	}

	want := datetime.DateRangeList{
		{datetime.Date{1, 2}, datetime.Date{3, 4}},
		{datetime.Date{1, 2}, datetime.Date{3, 4}},
		{datetime.Date{1, 1}, datetime.Date{3, 31}},
		{datetime.Date{1, 1}, datetime.Date{3, 31}},
		{datetime.Date{1, 1}, datetime.Date{3, 31}},
		{datetime.Date{1, 1}, datetime.Date{3, 31}},
		{datetime.Date{11, 1}, datetime.Date{12, 31}},
		{datetime.Date{11, 1}, datetime.Date{12, 20}},
		{datetime.Date{2, 1}, datetime.Date{2, 29}},
	}

	for i, tc := range ranges {
		var dr datetime.DateRange
		if err := dr.Parse(2024, tc); err != nil {
			t.Errorf("failed: %v: %v", tc, err)
		}
		if got, want := dr, want[i]; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	var dr datetime.DateRangeList
	if err := dr.Parse(2024, ranges); err != nil {
		t.Errorf("failed: %v", err)
	}

	want = datetime.DateRangeList{
		{datetime.Date{1, 1}, datetime.Date{3, 31}},
		{datetime.Date{1, 2}, datetime.Date{3, 4}},
		{datetime.Date{2, 1}, datetime.Date{2, 29}},
		{datetime.Date{11, 1}, datetime.Date{12, 20}},
		{datetime.Date{11, 1}, datetime.Date{12, 31}},
	}

	if got := dr; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// non-leap year
	var ldc datetime.DateRange
	if err := ldc.Parse(2023, "02:feb"); err != nil {
		t.Errorf("failed: %v", err)
	}
	if got, want := ldc, (datetime.DateRange{datetime.Date{2, 1}, datetime.Date{2, 28}}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, tc := range []string{
		"xxx",
		"feb:jan",
		"feb-20:feb-02",
		"feb-20:feb:02",
		"feb-20:feb-29",
	} {
		var dr datetime.DateRange
		if err := dr.Parse(2023, tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}
}

func TestDateRange(t *testing.T) {
	nd := newDate
	dra := datetime.NewDateRange(2024, nd(1, 0), nd(3, 0))
	if got, want := dra.From, nd(1, 1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := dra.To, nd(3, 31); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	dra = datetime.NewDateRange(2024, nd(3, 0), nd(1, 0))
	if got, want := dra.From, nd(1, 1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := dra.To, nd(3, 31); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func datesAsString(m, d int) string {
	s := ""
	for i := 1; i <= d; i++ {
		s += fmt.Sprintf("%02d/%02d,", m, i)
	}
	return s
}

func TestDataRangeIterator(t *testing.T) {
	m := datesAsString
	year := 2024
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"01/01:01/03", "01/01,01/02,01/03"},
		{"01/02:01/03", "01/02,01/03"},
		{"01/30:02/02", "01/30,01/31,02/01,02/02"},
		{"02/27:02", "02/27,02/28,02/29"},
		{"03/30:05/02", "03/30,03/31," + m(4, 30) + "05/01,05/02"},
	} {
		var dr datetime.DateRange
		var dates dateList
		var days []int
		if err := dr.Parse(year, tc.input); err != nil {
			t.Errorf("failed: %v", err)
		}
		for d := range dr.Dates(year) {
			dates = append(dates, d)
		}
		if got, want := dates.String(), tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		for d := range dr.Days(year) {
			days = append(days, d)
		}
		if got, want := days, daysFromDatesString(year, tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
	// non-leap year
	year = 2023
	var dr datetime.DateRange
	if err := dr.Parse(year, "02/27:03/02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var dates dateList
	for d := range dr.Dates(year) {
		dates = append(dates, d)
	}
	if got, want := dates.String(), "02/27,02/28,03/01,03/02"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	var days []int
	for d := range dr.Days(year) {
		days = append(days, d)
	}
	if got, want := days, daysFromDatesString(year, "02/27,02/28,03/01,03/02"); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDataRangeIteratorConstrained(t *testing.T) {
	year := 2024
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}
	custom := datetime.Constraints{Custom: []datetime.Date{{2, 2}, {2, 5}, {2, 20}}}

	for _, tc := range []struct {
		input      string
		constraint datetime.Constraints
		output     string
	}{
		{"02:02", weekdays, "02/01,02/02,02/05,02/06,02/07,02/08,02/09,02/12,02/13,02/14,02/15,02/16,02/19,02/20,02/21,02/22,02/23,02/26,02/27,02/28,02/29"},
		{"02:02", weekends, "02/03,02/04,02/10,02/11,02/17,02/18,02/24,02/25"},
		{"02:02", custom, "02/01,02/03,02/04,02/06,02/07,02/08,02/09,02/10,02/11,02/12,02/13,02/14,02/15,02/16,02/17,02/18,02/19,02/21,02/22,02/23,02/24,02/25,02/26,02/27,02/28,02/29"},
	} {
		var dr datetime.DateRange
		var dates dateList
		var days []int
		if err := dr.Parse(year, tc.input); err != nil {
			t.Errorf("failed: %v", err)
		}
		for d := range dr.DatesConstrained(year, tc.constraint) {
			dates = append(dates, d)
		}
		if got, want := dates.String(), tc.output; got != want {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}
		for d := range dr.DaysConstrained(year, tc.constraint) {
			days = append(days, d)
		}
		if got, want := days, daysFromDatesString(year, tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}
	}
	// non-leap year
	var dr datetime.DateRange
	if err := dr.Parse(2023, "02/27:03/02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var out dateList
	for d := range dr.Dates(2023) {
		out = append(out, d)
	}
	if got, want := out.String(), "02/27,02/28,03/01,03/02"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDataRangeRangesIterator(t *testing.T) {
	year := 2024
	unconstrained := datetime.Constraints{}
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}
	customExclusion := datetime.Constraints{Custom: []datetime.Date{{2, 2}, {2, 5}, {2, 20}}}

	for _, tc := range []struct {
		input      string
		constraint datetime.Constraints
		output     string
	}{
		{"02/01:02/01", unconstrained, "2/1:2/1"},
		{"02:02", unconstrained, "2/1:2/29"},
		{"02:02", weekdays, "2/1:2/2,2/5:2/9,2/12:2/16,2/19:2/23,2/26:2/29"},
		{"02:02", weekends, "2/3:2/4,2/10:2/11,2/17:2/18,2/24:2/25"},
		{"02:02", customExclusion, "2/1:2/1,2/3:2/4,2/6:2/19,2/21:2/29"},
		{"01:02", unconstrained, "1/1:2/29"},
		{"01:01", unconstrained, "1/1:1/31"},
	} {
		var dr datetime.DateRange
		var ranges datetime.DateRangeList
		if err := dr.Parse(year, tc.input); err != nil {
			t.Errorf("failed: %v", err)
			continue
		}
		for r := range dr.RangesConstrained(year, tc.constraint) {
			ranges = append(ranges, r)
		}

		var expected datetime.DateRangeList
		if err := expected.Parse(year, strings.Split(tc.output, ",")); err != nil {
			t.Errorf("failed: %v: %v", tc.output, err)
		}
		if got, want := ranges, expected; !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %#v: got %v, want %v", tc.input, tc.constraint, got, want)
		}
	}

	// non-leap year
	var dr datetime.DateRange
	if err := dr.Parse(2023, "02:02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var ranges datetime.DateRangeList
	for r := range dr.RangesConstrained(2023, weekdays) {
		ranges = append(ranges, r)
	}
	var expected datetime.DateRangeList
	if err := expected.Parse(year, strings.Split("2/1:2/3,2/6:2/10,2/13:2/17,2/20:2/24,2/27:2/28", ",")); err != nil {
		t.Errorf("failed: %v", err)
	}
	if got, want := ranges, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}
