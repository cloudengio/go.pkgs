// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"slices"
	"strings"
)

// DateRange represents a range of dates, inclusive of the start and end dates.
// NewDateRange and Parse create or initialize a DateRange. The from and to months
// are stored in the top 8 bits and the from and to days in the lower 8 bits to
// allow for sorting.
type DateRange uint32

func (dr DateRange) from() (Month, uint8) {
	return Month(dr >> 24 & 0xff), uint8(dr >> 8 & 0xff)
}

func (dr DateRange) to() (Month, uint8) {
	return Month(dr >> 16 & 0xff), uint8(dr & 0xff)
}

func (dr DateRange) fromDate() Date {
	return newDate8(Month(dr>>24&0xff), uint8(dr>>8&0xff))
}

func (dr DateRange) toDate() Date {
	return newDate8(Month(dr>>16&0xff), uint8(dr&0xff))
}

func newDateRange(from, to Date) DateRange {
	fm, fd := DateRange(from.Month()), DateRange(from.Day())
	tm, td := DateRange(to.Month()), DateRange(to.Day())
	return fm<<24 | tm<<16 | fd<<8 | td
}

// From returns the start date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (dr DateRange) From(year int) Date {
	fromMonth, fromDay := dr.from()
	if fromMonth == 2 && fromDay == 29 && !IsLeap(year) {
		return newDate8(fromMonth, 28)
	}
	return newDate8(fromMonth, fromDay)
}

// To returns the end date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (dr DateRange) To(year int) Date {
	toMonth, toDay := dr.to()
	if toMonth == 2 && toDay == 29 && !IsLeap(year) {
		return newDate8(2, 28)
	}
	return newDate8(toMonth, toDay)
}

// NewDateRange returns a DateRange for the from/to dates for the specified year.
// If the from date is later than the to date then they are swapped.
// The resulting from and to dates are then normalized using
// date.Normalize(year, true) for the from date and date.Normalize(year, false) for the to date.
func NewDateRange(year int, from, to Date) DateRange {
	if from > to {
		from, to = to, from
	}
	from, to = from.Normalize(year, true), to.Normalize(year, false)
	return newDateRange(from, to)
}

func (dr DateRange) CalendarDateRange(year int) CalendarDateRange {
	from, to := dr.From(year).CalendarDate(year), dr.To(year).CalendarDate(year)
	return NewCalendarDateRange(from, to)
}

func (dr DateRange) String() string {
	return fmt.Sprintf("%s - %s", dr.fromDate().String(), dr.toDate().String())
}

// Parse ranges in formats '01:03', 'Jan:Mar', '01/02:03-04' or 'Jan-02:Mar-04'.
// If the from day is zero then it is treated as the first day of the month.
// If the from day is 29 for a non-leap year then it is left as 29.
// If the to day is zero then it is treated as the last day of the month taking
// the year into account for Feb.
// The start date must be before the end date after normalization as per
// the above rules.
func (d *DateRange) Parse(year int, val string) error {
	parts := strings.Split(val, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, %q expected '<from>:<to>'", val)
	}
	var from, to Date
	if err := from.Parse(year, parts[0]); err != nil {
		return fmt.Errorf("invalid from: %s: %v", parts[0], err)
	}
	if err := to.Parse(year, parts[1]); err != nil {
		return fmt.Errorf("invalid to: %s: %v", parts[1], err)
	}
	from = from.Normalize(year, true)
	to = to.Normalize(year, false)
	if to < from {
		return fmt.Errorf("from is later than to: %s %s", from, to)
	}
	*d = newDateRange(from, to)
	return nil
}

// Equal returns true if the two DateRange values are equal
// for the given year.
func (dr DateRange) Equal(year int, dr2 DateRange) bool {
	af, bf := dr.From(year), dr2.From(year)
	at, bt := dr.To(year), dr2.To(year)
	return af == bf && at == bt
}

// Dates returns an iterator that yields each CalendarDate in the range for the
// given year. All of the CalendarDate values will have the same year.
func (dr DateRange) Dates(year int) func(yield func(CalendarDate) bool) {
	return dr.CalendarDateRange(year).Dates()
}

// DatesConstrained returns an iterator that yields each CalendarDate in the range for the
// given year constrained by the given DateConstraints. All of the CalendarDate values
// will have the same year.
func (dr DateRange) DatesConstrained(year int, dc Constraints) func(yield func(CalendarDate) bool) {
	return dr.CalendarDateRange(year).DatesConstrained(dc)
}

// Ranges returns an iterator that yields each DateRange in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) RangesConstrained(year int, dc Constraints) func(yield func(CalendarDateRange) bool) {
	return dr.CalendarDateRange(year).RangesConstrained(dc)
}

// Days returns an iterator that yields each day in the range for the
// given year.
func (dr DateRange) Days(year int) func(yield func(YearDay) bool) {
	return dr.CalendarDateRange(year).Days()
}

// DaysConstrained returns an iterator that yields each day in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) DaysConstrained(year int, dc Constraints) func(yield func(YearDay) bool) {
	return dr.CalendarDateRange(year).DaysConstrained(dc)
}

// DateRangeList represents a list of DateRange values, it can be sorted and searched
// using the slices package.
type DateRangeList []DateRange

// Parse ranges in formats '01:03', 'Jan:Mar', '01-02:03-04' or 'Jan-02:Mar-04'.
// The parsed list is sorted and without duplicates. If the start date is
// identical then the end date is used to determine the order.
func (drl *DateRangeList) Parse(year int, ranges []string) error {
	if len(ranges) == 0 {
		return nil
	}
	drs := make(DateRangeList, 0, len(ranges))
	seen := map[DateRange]struct{}{}
	for _, rg := range ranges {
		var dr DateRange
		if err := dr.Parse(year, rg); err != nil {
			return err
		}
		if _, ok := seen[dr]; ok {
			continue
		}
		drs = append(drs, dr)
		seen[dr] = struct{}{}
	}
	slices.Sort(drs)
	*drl = drs
	return nil
}

// Equal returns true if the two DateRangeList values are equal for the given year.
func (drl DateRangeList) Equal(year int, dr2 DateRangeList) bool {
	if len(drl) != len(dr2) {
		return false
	}
	for i, d := range drl {
		if !d.Equal(year, dr2[i]) {
			return false
		}
	}
	return true
}

func (dl DateList) ExpandMonths(year int) DateRangeList {
	drl := make(DateRangeList, 0, len(dl))
	for _, d := range dl {
		drl = append(drl, newDateRange(d, d))
	}
	return drl.Merge(year)
}

// Merge returns a new list of date ranges that containes merged consecutive
// dates into ranges. All dates are normalized using date.Normalize(year, true).
// The date list is assumed to be sorted.
func (dl DateList) Merge(year int) DateRangeList {
	dm := daysInMonthForYear(year)
	merged := make(DateRangeList, 0, len(dl))

	from := dl[0].Normalize(year, true)
	to := dl[0].Normalize(year, true)
	for i := 1; i < len(dl); i++ {
		prev, cur := dl[i-1].Normalize(year, true), dl[i].Normalize(year, true)
		if prev == cur {
			// duplicate
			continue
		}
		if prev == cur.yesterday(dm) {
			to = cur
			continue
		}
		merged = append(merged, newDateRange(from, to))
		from = cur
		to = cur
	}
	return slices.Clip(append(merged, newDateRange(from, to)))
}

// Merge returns a new list of date ranges that contains merged consecutive
// overlapping ranges.
// The date list is assumed to be sorted.
func (drl DateRangeList) Merge(year int) DateRangeList {
	if len(drl) == 0 {
		return drl
	}
	leap := IsLeap(year)
	merged := make(DateRangeList, 0, len(drl))
	from := drl[0].From(year)
	to := drl[0].To(year)
	for i := 1; i < len(drl); i++ {
		prevTo, curFrom := drl[i-1].To(year), drl[i].From(year)
		if prevTo.calcDayOfYear(leap) >= (curFrom.calcDayOfYear(leap) - 1) {
			to = drl[i].To(year)
			if prevTo > to {
				to = prevTo
			}
			continue
		}
		merged = append(merged, newDateRange(from, to))
		from = curFrom
		to = drl[i].To(year)
		if prevTo > to {
			to = prevTo
		}
	}
	return slices.Clip(append(merged, newDateRange(from, to)))
}

// MergeMonths returns a merged list of date ranges that contains the specified
// months for the given year.
func (drl DateRangeList) MergeMonths(year int, months MonthList) DateRangeList {
	ndrl := make(DateRangeList, 0, len(months)+len(drl))
	for _, m := range months {
		ndrl = append(ndrl, newDateRange(newDate8(m, 1), newDate8(m, DaysInMonth(year, m))))
	}
	ndrl = append(ndrl, drl...)
	slices.Sort(ndrl)
	return ndrl.Merge(year)
}
