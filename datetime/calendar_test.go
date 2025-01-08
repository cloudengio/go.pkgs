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

func TestParseCalendarDates(t *testing.T) {
	ncd := datetime.NewCalendarDate
	for _, tc := range []struct {
		input string
		cd    datetime.CalendarDate
	}{
		{"01/01/2024", ncd(2024, 1, 1)},
		{"01/2024", ncd(2024, 1, 0)},
		{"feb-2024", ncd(2024, 2, 0)},
		{"02/29/2024", ncd(2024, 2, 29)},
		{"02/28/2023", ncd(2023, 2, 28)},
		{"Jan-01-2024", ncd(2024, 1, 1)},
		{"Feb-29-2024", ncd(2024, 2, 29)},
		{"Feb-28-2023", ncd(2023, 2, 28)},
	} {
		var cd datetime.CalendarDate
		if err := cd.Parse(tc.input); err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if cd != tc.cd {
			t.Errorf("%v: got %v, want %v", tc.input, cd, tc.cd)
		}
		str := cd.String()
		if err := cd.Parse(str); err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if cd != tc.cd {
			t.Errorf("%v: got %v, want %v", tc.input, cd, tc.cd)
		}
	}

	for _, tc := range []string{
		"02/29/2023",
		"Feb-29-2023",
		"02-03",
		"Jan/03",
	} {
		var cd datetime.CalendarDate
		if err := cd.Parse(tc); err == nil {
			t.Errorf("%v: expected error", tc)
		}
	}
}

func TestParseAnyDates(t *testing.T) {
	ncd := datetime.NewCalendarDate
	for _, tc := range []struct {
		year  int
		input string
		cd    datetime.CalendarDate
	}{
		{2024, "01/01/2024", ncd(2024, 1, 1)},
		{2024, "02/29/2024", ncd(2024, 2, 29)},
		{2023, "02/28/2023", ncd(2023, 2, 28)},
		{2024, "Jan-01-2024", ncd(2024, 1, 1)},
		{2024, "Feb-29-2024", ncd(2024, 2, 29)},
		{2023, "Feb-28-2023", ncd(2023, 2, 28)},

		{2024, "01/01", ncd(2024, 1, 1)},
		{2024, "02/29", ncd(2024, 2, 29)},
		{2023, "02/28", ncd(2023, 2, 28)},
		{2024, "Jan-01", ncd(2024, 1, 1)},
		{2024, "Feb-29", ncd(2024, 2, 29)},
		{2023, "Feb-28", ncd(2023, 2, 28)},

		{2024, "01", ncd(2024, 1, 0)},
		{2024, "Febr", ncd(2024, 2, 0)},
	} {
		cd, err := datetime.ParseAnyDate(tc.year, tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if cd != tc.cd {
			t.Errorf("%v: got %v, want %v", tc.input, cd, tc.cd)
		}
	}

	for _, tc := range []string{
		"02/29/2023",
		"Feb-29-2023",
		"02-03",
		"Jan/03",
	} {
		var cd datetime.CalendarDate
		if err := cd.Parse(tc); err == nil {
			t.Errorf("%v: expected error", tc)
		}
	}
}

func TestCalendarDateRangeParse(t *testing.T) {
	ranges := []string{
		"01/02/2004:03/04/2006",
		"Jan-02-2004:Mar-04-2006",
		"01/1/2004:03/31/2006",
		"Jan-1-2004:Mar-31-2006",
		"january-1-2004:Mar-31-2006",
	}

	ncd := newCalendarDate
	ncdr := newCalendarDateRange
	expected := datetime.CalendarDateRangeList{
		ncdr(ncd(2004, 1, 2), ncd(2006, 3, 4)),
		ncdr(ncd(2004, 1, 2), ncd(2006, 3, 4)),
		ncdr(ncd(2004, 1, 1), ncd(2006, 3, 31)),
		ncdr(ncd(2004, 1, 1), ncd(2006, 3, 31)),
		ncdr(ncd(2004, 1, 1), ncd(2006, 3, 31)),
	}

	for i, tc := range ranges {
		var cdr datetime.CalendarDateRange
		if err := cdr.Parse(tc); err != nil {
			t.Errorf("failed: %v: %v", tc, err)
		}
		if got, want := cdr, expected[i]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	ranges = append(ranges,
		"01/02:03/04",
		"Jan-02:Mar-04",
		"01/1:03/31",
		"Jan-1:Mar-31",
		"january-1:Mar-31",
		"jan:feb",
	)

	expected = append(expected,
		ncdr(ncd(2004, 1, 2), ncd(2008, 3, 4)),
		ncdr(ncd(2004, 1, 2), ncd(2008, 3, 4)),
		ncdr(ncd(2004, 1, 1), ncd(2008, 3, 31)),
		ncdr(ncd(2004, 1, 1), ncd(2008, 3, 31)),
		ncdr(ncd(2004, 1, 1), ncd(2008, 3, 31)),
		ncdr(ncd(2004, 1, 1), ncd(2008, 2, 29)),
	)

	for i, tc := range ranges {
		parts := strings.Split(tc, ":")
		a, err := datetime.ParseAnyDate(2004, parts[0])
		if err != nil {
			t.Errorf("failed: %v: %v", tc, err)
		}
		b, err := datetime.ParseAnyDate(2008, parts[1])
		if err != nil {
			t.Errorf("failed: %v: %v", tc, err)
		}
		cdr := datetime.NewCalendarDateRange(a, b)

		if got, want := cdr, expected[i]; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	cdr := datetime.NewCalendarDateRange(ncd(2004, 1, 2), ncd(2006, 2, 0))
	if got, want := cdr, ncdr(ncd(2004, 1, 2), ncd(2006, 2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := cdr, ncdr(ncd(2004, 1, 2), ncd(2006, 2, 28)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cdr = datetime.NewCalendarDateRange(ncd(2004, 1, 2), ncd(2008, 2, 0))
	if got, want := cdr, ncdr(ncd(2004, 1, 2), ncd(2008, 2, 29)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// Should be Feb 28th for a leap year.
	if got, want := cdr, ncdr(ncd(2004, 1, 2), ncd(2008, 2, 28)); got == want {
		t.Errorf("unexpected value, %v is the same as %v", got, want)
	}

	for _, tc := range []string{
		"xxx",
		"feb:jan",
		"feb-20:feb-02",
		"feb-20:feb:02",
		"feb-20:feb-29-2023",
		"feb-0-29:feb-0-28",
		"feb-29-2023",
	} {
		var cdr datetime.CalendarDateRange
		if err := cdr.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}

}

func TestMergeCalendarDates(t *testing.T) {
	ncd := newCalendarDate
	ncdl := newCalendarDateList
	ncdrl := newCalendarDateRangeList
	for _, tc := range []struct {
		dates  datetime.CalendarDateList
		merged datetime.CalendarDateRangeList
	}{
		{ncdl(ncd(2024, 1, 1), ncd(2024, 1, 1)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 1))},
		{ncdl(ncd(2024, 1, 1), ncd(2024, 1, 1), ncd(2024, 1, 1)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 1))},
		{ncdl(ncd(2024, 1, 1), ncd(2024, 1, 2)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2))},
		{ncdl(ncd(2024, 1, 0), ncd(2024, 1, 2)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 2))},
		{ncdl(ncd(2024, 1, 0), ncd(2024, 1, 0)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 1))},
		{ncdl(ncd(2024, 0, 0), ncd(2024, 0, 0)), ncdrl(ncd(2024, 1, 1), ncd(2024, 1, 1))},
		{ncdl(ncd(2024, 2, 28), ncd(2024, 2, 29)), ncdrl(ncd(2024, 2, 28), ncd(2024, 2, 29))},
		{ncdl(ncd(2023, 2, 28), ncd(2023, 2, 29)), ncdrl(ncd(2023, 2, 28), ncd(2023, 2, 28))},
		{ncdl(ncd(2024, 12, 30), ncd(2024, 12, 31), ncd(2025, 1, 1)), ncdrl(ncd(2024, 12, 30), ncd(2025, 1, 1))},
		{ncdl(ncd(2024, 12, 30), ncd(2024, 12, 31), ncd(2025, 1, 1), ncd(2025, 1, 3), ncd(2025, 1, 4)), ncdrl(ncd(2024, 12, 30), ncd(2025, 1, 1), ncd(2025, 1, 3), ncd(2025, 1, 4))},
	} {
		slices.Sort(tc.dates)
		merged := tc.dates.Merge()
		if got, want := merged, tc.merged; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
