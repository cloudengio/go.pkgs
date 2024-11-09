// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"slices"
	"strings"
	"time"
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
	return DateRange(fm<<24 | tm<<16 | fd<<8 | td)
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
		return fmt.Errorf("invalid from: %s", parts[0])
	}
	if err := to.Parse(year, parts[1]); err != nil {
		return fmt.Errorf("invalid to: %s", parts[1])
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

// Dates returns an iterator that yields each Date in the range for the
// given year.
func (d DateRange) Dates(year int) func(yield func(Date) bool) {
	dm := daysInMonthForYear(year)
	to := d.To(year)
	return func(yield func(Date) bool) {
		for td := d.From(year); td <= to; td = td.tomorrow(dm) {
			if !yield(td) {
				return
			}
		}
	}
}

// DatesConstrained returns an iterator that yields each Date in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) DatesConstrained(year int, dc Constraints) func(yield func(Date) bool) {
	return func(yield func(Date) bool) {
		dm := daysInMonthForYear(year)
		to := dr.To(year)
		for td := dr.From(year); td <= to; td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month()), td.Day(), 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(td) {
				return
			}
		}
	}
}

// Ranges returns an iterator that yields each DateRange in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) RangesConstrained(year int, dc Constraints) func(yield func(DateRange) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(DateRange) bool) {
		start, stop := dr.From(year), dr.To(year)
		to := stop
		inrange := dc.Include(time.Date(year, time.Month(start.Month()), start.Day(), 0, 0, 0, 0, time.UTC))
		for td := dr.From(year); td <= to; td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month()), td.Day(), 0, 0, 0, 0, time.UTC)) {
				if inrange {
					// Range ends
					if !yield(newDateRange(start, stop)) {
						return
					}
				}
				inrange = false
				continue
			}
			if !inrange {
				start = td
				inrange = true
			}
			stop = td
		}
		if inrange {
			yield(newDateRange(start, stop))
		}
	}
}

// Days returns an iterator that yields each day in the range for the
// given year.
func (dr DateRange) Days(year int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		last := dr.To(year).DayOfYear(year)
		for yd := dr.From(year).DayOfYear(year); yd <= last; yd++ {
			if !yield(yd) {
				return
			}
		}
	}
}

// DaysConstrained returns an iterator that yields each day in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) DaysConstrained(year int, dc Constraints) func(yield func(int) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(int) bool) {
		to := dr.To(year).DayOfYear(year)
		for d := dr.From(year).DayOfYear(year); d <= to; d++ {
			tm := dateFromDay(d, dm)
			if !dc.Include(time.Date(year, time.Month(tm.Month()), tm.Day(), 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(d) {
				return
			}
		}
	}
}

// DateRangeList represents a list of DateRange values, it can be sorted and searched
// using the slices package.
type DateRangeList []DateRange

// MergeMonthsAndRanges creates an ordered list of DateRange values from the given
// MonthList and DateRangeList.
func MergeMonthsAndRanges(year int, months MonthList, ranges DateRangeList) DateRangeList {
	drl := make(DateRangeList, 0, len(months)+len(ranges))
	for _, m := range months {
		drl = append(drl, newDateRange(newDate8(m, 1), newDate8(m, DaysInMonth(year, m))))
	}
	drl = append(drl, ranges...)
	slices.Sort(drl)
	return MergeRanges(year, drl)
}

// Parse ranges in formats '01:03', 'Jan:Mar', '01-02:03-04' or 'Jan-02:Mar-04'.
// The parsed list is sorted and without duplicates. If the start date is
// identical then the end date is used to determine the order.
func (dr *DateRangeList) Parse(year int, ranges []string) error {
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
	*dr = drs
	return nil
}

// Equal returns true if the two DateRangeList values are equal for the given year.
func (dr DateRangeList) Equal(year int, dr2 DateRangeList) bool {
	if len(dr) != len(dr2) {
		return false
	}
	for i, d := range dr {
		if !d.Equal(year, dr2[i]) {
			return false
		}
	}
	return true
}

func MergeDates(year int, dates []Date) DateRangeList {
	slices.Sort(dates)
	dm := daysInMonthForYear(year)
	var drs []DateRange
	from := dates[0].Normalize(year, true)
	to := dates[0].Normalize(year, false)
	for i := 1; i < len(dates); i++ {
		if dates[i-1] == dates[i] {
			continue
		}
		if dates[i-1] == dates[i].yesterday(dm) {
			to = dates[i]
			continue
		}
		drs = append(drs, newDateRange(from, to))
		from = dates[i].Normalize(year, false)
		to = from
	}
	drs = append(drs, newDateRange(from, to))
	return drs
}

func MergeRanges(year int, ranges []DateRange) DateRangeList {
	if len(ranges) == 0 {
		return nil
	}
	slices.Sort(ranges)
	leap := IsLeap(year)
	var merged []DateRange

	from := ranges[0].From(year)
	to := ranges[0].To(year)
	for i := 1; i < len(ranges); i++ {
		fd := ranges[i-1].To(year).dayOfYear(leap)
		td := ranges[i].From(year).dayOfYear(leap)
		if fd >= (td - 1) {
			to = ranges[i].To(year)
			continue
		}
		merged = append(merged, newDateRange(from, to))
		from = ranges[i].From(year)
		to = ranges[i].To(year)
	}
	merged = append(merged, newDateRange(from, to))
	return merged
}
