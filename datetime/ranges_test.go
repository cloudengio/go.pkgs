// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
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
		"feb-29:feb-29", // normalizes to 2/28 for non-leap years.
	}

	nd := newDate
	ndr := datetime.NewDateRange
	want := datetime.DateRangeList{
		ndr(nd(1, 2), nd(3, 4)),
		ndr(nd(1, 2), nd(3, 4)),
		ndr(nd(1, 1), nd(3, 31)),
		ndr(nd(1, 1), nd(3, 31)),
		ndr(nd(1, 1), nd(3, 31)),
		ndr(nd(1, 1), nd(3, 31)),
		ndr(nd(11, 1), nd(12, 31)),
		ndr(nd(11, 1), nd(12, 20)),
		ndr(nd(2, 1), nd(2, 29)),  // normalizes to 2/28 for non-leap years.
		ndr(nd(2, 29), nd(2, 29)), // normalizes to 2/28 for non-leap years.
	}

	for i, tc := range ranges {
		var dr datetime.DateRange
		if err := dr.Parse(tc); err != nil {
			t.Errorf("failed: %v: %v", tc, err)
		}

		// The dates will be normalized for the equality test,
		// so 2/29 will be treated as 2/28 in 2023 and any zero
		// days will be treated as either the first or last day of
		// the month.
		if got, want := dr, want[i]; !got.Equal(2024, want) {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := dr, want[i]; !got.Equal(2023, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	// non-leap year
	var ldc datetime.DateRange
	if err := ldc.Parse("02:feb"); err != nil {
		t.Errorf("failed: %v", err)
	}
	if got, want := ldc, ndr(nd(2, 0), nd(2, 0)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := ldc.Normalize(2023), ndr(nd(2, 1), nd(2, 28)); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, tc := range []string{
		"",
		"xxx",
		"feb-20:feb:02",
		"feb-20:feb:29",
	} {
		var dr datetime.DateRange
		if err := dr.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}
}

func TestDateRanges(t *testing.T) {
	nd := newDate
	ncd := newCalendarDate
	ndr := datetime.NewDateRange
	ncdr := newCalendarDateRange

	dra := ndr(nd(1, 0), nd(2, 0))
	if got, want := dra, ndr(nd(1, 0), nd(2, 0)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// The to/from dates are swapped when created.
	dra = ndr(nd(2, 0), nd(1, 0))
	if got, want := dra, ndr(nd(1, 0), nd(2, 0)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// test normalization.
	if got, want := dra.Normalize(2024), ndr(nd(1, 1), nd(2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := dra.Normalize(2023), ndr(nd(1, 1), nd(2, 28)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// date ranges are normalized before creating calendar date ranges.
	dra = ndr(nd(2, 0), nd(2, 0))
	cdra := dra.CalendarDateRange(2023)
	if got, want := cdra, ncdr(ncd(2023, 2, 1), ncd(2023, 2, 28)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cdra = dra.CalendarDateRange(2024)
	if got, want := cdra, ncdr(ncd(2024, 2, 1), ncd(2024, 2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCalendarDateRanges(t *testing.T) {
	ncd := newCalendarDate
	ncdr := datetime.NewCalendarDateRange

	// from, to are swapped and then normalized.
	cdra := ncdr(ncd(2024, 1, 0), ncd(2024, 2, 0))
	if got, want := cdra, ncdr(ncd(2024, 1, 1), ncd(2024, 2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cdra = ncdr(ncd(2024, 2, 0), ncd(2024, 0, 0))
	if got, want := cdra, ncdr(ncd(2024, 1, 1), ncd(2024, 2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// test non-leap year.
	cdra = ncdr(ncd(2023, 1, 0), ncd(2023, 2, 0))
	if got, want := cdra, ncdr(ncd(2023, 1, 1), ncd(2023, 2, 28)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDateRangeSorting(t *testing.T) {
	ndrl := newDateRangeList
	nd := newDate
	r1 := ndrl(
		nd(1, 2), nd(2, 11),
		nd(1, 2), nd(1, 13),
		nd(3, 1), nd(3, 10),
		nd(11, 20), nd(11, 30),
		nd(1, 1), nd(1, 31),
		nd(3, 1), nd(10, 10),
		nd(1, 2), nd(1, 11),
		nd(3, 1), nd(3, 9),
	)
	slices.Sort(r1)
	if got, want := r1, ndrl(
		nd(1, 1), nd(1, 31),
		nd(1, 2), nd(1, 11),
		nd(1, 2), nd(1, 13),
		nd(1, 2), nd(2, 11),
		nd(3, 1), nd(3, 9),
		nd(3, 1), nd(3, 10),
		nd(3, 1), nd(10, 10),
		nd(11, 20), nd(11, 30),
	); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCalendarDateRangeSorting(t *testing.T) {
	ncdrl := newCalendarDateRangeList
	ncd := newCalendarDate
	r1 := ncdrl(
		ncd(2024, 1, 2), ncd(2024, 2, 11),
		ncd(2024, 1, 2), ncd(2024, 1, 13),
		ncd(2024, 3, 1), ncd(2024, 3, 10),
		ncd(2023, 11, 20), ncd(2023, 11, 30),
		ncd(2024, 11, 20), ncd(2024, 11, 30),
		ncd(2024, 1, 1), ncd(2024, 1, 31),
		ncd(2024, 3, 1), ncd(2024, 10, 10),
		ncd(2024, 1, 2), ncd(2024, 1, 11),
		ncd(2025, 1, 2), ncd(2026, 1, 11),
		ncd(2024, 3, 1), ncd(2024, 3, 9),
		ncd(2023, 11, 20), ncd(2025, 11, 30),
	)
	slices.Sort(r1)
	if got, want := r1, ncdrl(
		ncd(2023, 11, 20), ncd(2023, 11, 30),
		ncd(2023, 11, 20), ncd(2025, 11, 30),
		ncd(2024, 1, 1), ncd(2024, 1, 31),
		ncd(2024, 1, 2), ncd(2024, 1, 11),
		ncd(2024, 1, 2), ncd(2024, 1, 13),
		ncd(2024, 1, 2), ncd(2024, 2, 11),
		ncd(2024, 3, 1), ncd(2024, 3, 9),
		ncd(2024, 3, 1), ncd(2024, 3, 10),
		ncd(2024, 3, 1), ncd(2024, 10, 10),
		ncd(2024, 11, 20), ncd(2024, 11, 30),
		ncd(2025, 1, 2), ncd(2026, 1, 11),
	); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDateRangeEquality(t *testing.T) {
	ndrl := newDateRangeList
	nd := newDate
	r1 := ndrl(
		nd(2, 1), nd(2, 29),
		nd(1, 1), nd(2, 31),
	)
	r2 := ndrl(
		nd(2, 1), nd(2, 29),
		nd(1, 1), nd(2, 31),
	)

	if got, want := r1.Equal(2024, r2), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := r1.Equal(2023, r2), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	r2 = ndrl(
		nd(2, 1), nd(2, 28),
		nd(1, 1), nd(2, 31),
	)
	// 2/28 is not the same as 2/29 in 2024, but it is in 2023.
	if got, want := r1.Equal(2024, r2), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := r1.Equal(2023, r2), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestDateRangeIterator(t *testing.T) {
	ds := datesAsString
	year := 2024
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"01/01:01/03", "01/01,01/02,01/03"},
		{"01/02:01/03", "01/02,01/03"},
		{"01/30:02/02", "01/30,01/31,02/01,02/02"},
		{"02/27:02", "02/27,02/28,02/29"},
		{"03/30:05/02", "03/30,03/31," + ds(4, 30) + "05/01,05/02"},
	} {
		var dr datetime.DateRange
		var dates dateList
		var yearDay []datetime.YearDay
		if err := dr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v: %v", tc.input, err)
		}
		for cd := range dr.Dates(year) {
			dates = append(dates, cd)
		}
		if got, want := len(dates), strings.Count(tc.output, ",")+1; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := dates.String(), appendYearToDates(year, tc.output); got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		for yd := range dr.Days(year) {
			yearDay = append(yearDay, yd)
		}
		if got, want := yearDay, daysFromDatesString(year, tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
	// non-leap year
	year = 2023
	var dr datetime.DateRange
	if err := dr.Parse("02/27:03/02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var dates dateList
	for d := range dr.Dates(year) {
		dates = append(dates, d)
	}
	if got, want := dates.String(), appendYearToDates(year, "02/27,02/28,03/01,03/02"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	var yearDay []datetime.YearDay
	for yd := range dr.Days(year) {
		yearDay = append(yearDay, yd)
	}
	if got, want := yearDay, daysFromDatesString(year, "02/27,02/28,03/01,03/02"); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCalendarDateRangeIterator(t *testing.T) {
	cds := calendarDatesAsString
	cms := calendarMonthsAsString
	for _, tc := range []struct {
		input  string
		output string
	}{
		{"01/01/2024:01/03/2024", "01/01/2024,01/02/2024,01/03/2024"},
		{"01/02/2024:01/03/2024", "01/02/2024,01/03/2024"},
		{"01/30/2024:02/02/2024", "01/30/2024,01/31/2024,02/01/2024,02/02/2024"},
		{"03/30/2024:05/02/2024", "03/30/2024,03/31/2024," + cms(2024, 4) + "05/01/2024,05/02/2024"},
		{"02/27/2024:03/02/2026", "02/27/2024,02/28/2024,02/29/2024," +
			cms(2024, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12) +
			cms(2025, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12) +
			cms(2026, 1, 2) + cds(2026, 3, 2)},
	} {
		var cdr datetime.CalendarDateRange
		var dates dateList
		var yearDay []datetime.YearDay
		if err := cdr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v: %v", tc.input, err)
		}
		for cd := range cdr.Dates() {
			dates = append(dates, cd)
		}
		tc.output = strings.TrimSuffix(tc.output, ",")
		if got, want := len(dates), strings.Count(tc.output, ",")+1; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := dates.String(), tc.output; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		for yd := range cdr.Days() {
			yearDay = append(yearDay, yd)
		}
		if got, want := yearDay, daysFromCalendarDatesString(tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}
}

func TestDataRangeIteratorConstrained(t *testing.T) {
	ndl := newDateList
	nd := newDate
	year := 2024
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}
	custom := datetime.Constraints{Custom: ndl(nd(2, 2), nd(2, 5), nd(2, 20))}

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
		var yearDay []datetime.YearDay
		if err := dr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v", err)
		}
		for d := range dr.DatesConstrained(year, tc.constraint) {
			dates = append(dates, d)
		}
		if got, want := dates.String(), appendYearToDates(year, tc.output); got != want {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}
		for yd := range dr.DaysConstrained(year, tc.constraint) {
			yearDay = append(yearDay, yd)
		}
		if got, want := yearDay, daysFromDatesString(year, tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}

	}
	// non-leap year
	year = 2023
	var dr datetime.DateRange
	if err := dr.Parse("02/27:03/02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var out dateList
	for d := range dr.Dates(year) {
		out = append(out, d)
	}
	if got, want := out.String(), appendYearToDates(year, "02/27,02/28,03/01,03/02"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCalendarDataRangeIteratorConstrained(t *testing.T) {
	ndl := newDateList
	nd := newDate
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}
	custom := datetime.Constraints{Custom: ndl(nd(2, 2), nd(2, 5), nd(2, 20))}

	for _, tc := range []struct {
		input      string
		constraint datetime.Constraints
		output     string
	}{
		{"02/2024:02/2024", weekdays, appendYearToDates(2024, "02/01,02/02,02/05,02/06,02/07,02/08,02/09,02/12,02/13,02/14,02/15,02/16,02/19,02/20,02/21,02/22,02/23,02/26,02/27,02/28,02/29")},
		{"02/2024:02/2024", weekends, appendYearToDates(2024, "02/03,02/04,02/10,02/11,02/17,02/18,02/24,02/25")},
		{"02/2024:02/2024", custom, appendYearToDates(2024, "02/01,02/03,02/04,02/06,02/07,02/08,02/09,02/10,02/11,02/12,02/13,02/14,02/15,02/16,02/17,02/18,02/19,02/21,02/22,02/23,02/24,02/25,02/26,02/27,02/28,02/29")},
		{"12/2024:01/08/2025", weekends, "12/01/2024,12/07/2024,12/08/2024,12/14/2024,12/15/2024,12/21/2024,12/22/2024,12/28/2024,12/29/2024,01/04/2025,01/05/2025"},
	} {
		var cdr datetime.CalendarDateRange
		var dates dateList
		var yearDay []datetime.YearDay
		if err := cdr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v", err)
		}
		for d := range cdr.DatesConstrained(tc.constraint) {
			dates = append(dates, d)
		}
		if got, want := dates.String(), tc.output; got != want {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}
		for yd := range cdr.DaysConstrained(tc.constraint) {
			yearDay = append(yearDay, yd)
		}
		if got, want := yearDay, daysFromCalendarDatesString(tc.output); !slices.Equal(got, want) {
			t.Errorf("%v: %#v:got %v, want %v", tc.input, tc.constraint, got, want)
		}
	}
}

func TestDataRangeRangesIterator(t *testing.T) {
	year := 2024
	ndl := newDateList
	nd := newDate
	unconstrained := datetime.Constraints{}
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}
	customExclusion := datetime.Constraints{Custom: ndl(nd(2, 2), nd(2, 5), nd(2, 20))}

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
		var ranges datetime.CalendarDateRangeList
		if err := dr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v", err)
			continue
		}
		for r := range dr.RangesConstrained(year, tc.constraint) {
			ranges = append(ranges, r)
		}

		var expected datetime.CalendarDateRangeList
		withYear := appendYearToRanges(year, tc.output)
		if err := expected.Parse(strings.Split(withYear, ",")); err != nil {
			t.Errorf("failed: %v: %v", tc.output, err)
		}
		if got, want := ranges, expected; !slices.Equal(got, want) {
			t.Errorf("%q: %#v: got %v, want %v", tc.input, tc.constraint, got, want)
		}
	}

	// non-leap year
	year = 2023
	var dr datetime.DateRange
	if err := dr.Parse("02:02"); err != nil {
		t.Fatalf("failed: %v", err)
	}
	var ranges datetime.CalendarDateRangeList
	for r := range dr.RangesConstrained(year, weekdays) {
		ranges = append(ranges, r)
	}
	var expected datetime.CalendarDateRangeList
	expectedStr := appendYearToRanges(year, "2/1:2/3,2/6:2/10,2/13:2/17,2/20:2/24,2/27:2/28")
	if err := expected.Parse(strings.Split(expectedStr, ",")); err != nil {
		t.Errorf("failed: %v", err)
	}
	if got, want := ranges, expected; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestCalendarDataRangeRangesIterator(t *testing.T) {
	unconstrained := datetime.Constraints{}
	weekdays := datetime.Constraints{Weekdays: true}
	weekends := datetime.Constraints{Weekends: true}

	for _, tc := range []struct {
		input      string
		constraint datetime.Constraints
		output     string
	}{
		{"02/2024:02/2024", unconstrained, appendYearToRanges(2024, "2/1:2/29")},
		{"02/2023:02/2024", unconstrained, "2/1/2023:2/29/2024"},
		{"02/2024:02/2024", weekdays, appendYearToRanges(2024, "2/1:2/2,2/5:2/9,2/12:2/16,2/19:2/23,2/26:2/29")},
		{"12/2023:03/02/2024", weekends, "12/2/2023:12/3/2023,12/9/2023:12/10/2023,12/16/2023:12/17/2023,12/23/2023:12/24/2023,12/30/2023:12/31/2023,1/6/2024:1/7/2024,1/13/2024:1/14/2024,1/20/2024:1/21/2024,1/27/2024:1/28/2024,2/3/2024:2/4/2024,2/10/2024:2/11/2024,2/17/2024:2/18/2024,2/24/2024:2/25/2024,3/2/2024:3/2/2024"},
	} {
		var dr datetime.CalendarDateRange
		var ranges datetime.CalendarDateRangeList
		if err := dr.Parse(tc.input); err != nil {
			t.Errorf("failed: %v", err)
			continue
		}
		for r := range dr.RangesConstrained(tc.constraint) {
			ranges = append(ranges, r)
		}
		var expected datetime.CalendarDateRangeList
		if err := expected.Parse(strings.Split(tc.output, ",")); err != nil {
			t.Errorf("failed: %v: %v", tc.output, err)
		}
		if got, want := ranges, expected; !slices.Equal(got, want) {
			t.Errorf("%q: %#v: got %v, want %v", tc.input, tc.constraint, got, want)
		}
	}
}

func TestMergeDateRanges(t *testing.T) {
	nd := newDate
	ndrl := newDateRangeList
	year := 2024
	for _, tc := range []struct {
		ranges datetime.DateRangeList
		merged datetime.DateRangeList
	}{
		{ndrl(nd(1, 1), nd(1, 2)), ndrl(nd(1, 1), nd(1, 2))},
		{ndrl(nd(1, 1), nd(1, 2), nd(1, 1), nd(1, 2)), ndrl(nd(1, 1), nd(1, 2))},
		{ndrl(nd(1, 1), nd(1, 2), nd(1, 3), nd(1, 10)), ndrl(nd(1, 1), nd(1, 10))},
		{ndrl(nd(1, 1), nd(1, 2), nd(1, 4), nd(1, 10)), ndrl(nd(1, 1), nd(1, 2), nd(1, 4), nd(1, 10))},
		{ndrl(nd(2, 27), nd(2, 29), nd(3, 1), nd(3, 10)), ndrl(nd(2, 27), nd(3, 10))},
		{ndrl(nd(1, 1), nd(12, 1), nd(2, 2), nd(4, 2)), ndrl(nd(1, 1), nd(12, 1))},
		{ndrl(nd(1, 1), nd(11, 1), nd(1, 1), nd(12, 1)), ndrl(nd(1, 1), nd(12, 1))},
	} {
		if got, want := tc.ranges.Merge(year), tc.merged; !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

}

func TestMergeCalendarDateRanges(t *testing.T) {
	ncd := newCalendarDate
	ncdrl := newCalendarDateRangeList
	for _, tc := range []struct {
		ranges datetime.CalendarDateRangeList
		merged datetime.CalendarDateRangeList
	}{
		{ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2))},
		{ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2), ncd(2024, 1, 1), ncd(2024, 1, 2)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2))},
		{ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2), ncd(2024, 1, 3), ncd(2024, 1, 10)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 10))},
		{ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2), ncd(2024, 1, 4), ncd(2024, 1, 10)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2), ncd(2024, 1, 4), ncd(2024, 1, 10))},
		{ncdrl(ncd(2024, 2, 27), ncd(2024, 2, 29), ncd(2024, 3, 1), ncd(2024, 3, 10)),
			ncdrl(ncd(2024, 2, 27), ncd(2024, 3, 10))},
		{ncdrl(ncd(2023, 12, 1), ncd(2023, 12, 11), ncd(2023, 12, 10), ncd(2024, 1, 10)),
			ncdrl(ncd(2023, 12, 1), ncd(2024, 1, 10))},
		{ncdrl(ncd(2023, 12, 30), ncd(2023, 12, 31), ncd(2024, 1, 1), ncd(2024, 1, 2)),
			ncdrl(ncd(2023, 12, 30), ncd(2024, 1, 2))},
		{ncdrl(ncd(2023, 12, 30), ncd(2024, 12, 31), ncd(2024, 1, 1), ncd(2024, 1, 2)),
			ncdrl(ncd(2023, 12, 30), ncd(2024, 12, 31))},
	} {
		slices.Sort(tc.ranges)
		if got, want := tc.ranges.Merge(), tc.merged; !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestDatesMonthsMerge(t *testing.T) {
	nd := newDate
	ndrl := newDateRangeList
	year := 2024
	for _, tc := range []struct {
		months datetime.MonthList
		ranges datetime.DateRangeList
		merged datetime.DateRangeList
	}{
		{datetime.MonthList{1}, ndrl(nd(2, 1), nd(2, 28)), ndrl(nd(1, 1), nd(2, 28))},
		{datetime.MonthList{}, ndrl(nd(2, 1), nd(2, 28)), ndrl(nd(2, 1), nd(2, 28))},
		{datetime.MonthList{1}, ndrl(nd(2, 2), nd(2, 28)), ndrl(nd(1, 1), nd(1, 31), nd(2, 2), nd(2, 28))},
	} {
		merged := tc.ranges.MergeMonths(year, tc.months)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestCalendarMonthsMerge(t *testing.T) {
	ncd := newCalendarDate
	ncdrl := newCalendarDateRangeList
	for _, tc := range []struct {
		months     datetime.MonthList
		mergeMonth int
		ranges     datetime.CalendarDateRangeList
		merged     datetime.CalendarDateRangeList
	}{
		{datetime.MonthList{1}, 2024,
			ncdrl(ncd(2024, 2, 1), ncd(2024, 2, 28)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 2, 28))},
		{datetime.MonthList{}, 2024,
			ncdrl(ncd(2024, 2, 1), ncd(2024, 2, 28)),
			ncdrl(ncd(2024, 2, 1), ncd(2024, 2, 28))},
		{datetime.MonthList{1}, 2024,
			ncdrl(ncd(2024, 2, 2), ncd(2024, 2, 28)),
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 31), ncd(2024, 2, 2), ncd(2024, 2, 28))},
		{datetime.MonthList{1}, 2023,
			ncdrl(ncd(2024, 2, 2), ncd(2024, 2, 28)),
			ncdrl(ncd(2023, 1, 1), ncd(2023, 1, 31), ncd(2024, 2, 2), ncd(2024, 2, 28))},
		{datetime.MonthList{1, 2, 12}, 2023,
			ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 5), ncd(2023, 11, 12), ncd(2023, 11, 31)),
			ncdrl(ncd(2023, 1, 1), ncd(2023, 2, 28), ncd(2023, 11, 12), ncd(2024, 1, 5))},
	} {
		merged := tc.ranges.MergeMonths(tc.mergeMonth, tc.months)
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
