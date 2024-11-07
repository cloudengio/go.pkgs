// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DateRange represents a range of dates, inclusive of the start and end dates.
// NewDateRange and Parse create or initialize a DateRange. If the
// From Day is zero, it is treated as 1. If the To Day is zero, it is treated
// as the last day of the month,
type DateRange struct {
	From, To Date
}

// ToForYear returns the end date of the range for the specified year.
// Feb 29 is treated as Feb
func (dr DateRange) ToForYear(year int) Date {
	if dr.To.YearSpecific() {
		if dr.To.Day == 29 {
			return Date{Month: 2, Day: DaysInFeb(year)}
		}
	}
	return dr.To
}

// NewDateRange returns a DateRange for the from/to dates for the
// specified year.
// If the from date is after the to date then the dates are swapped.
func NewDateRange(year int, from, to Date) DateRange {
	ofrom, oto := from, to
	if from.Day == 0 {
		from.Day = 1
	}
	if to.Day == 0 {
		to.Day = daysInMonthForYear(year)[to.Month-1]
	}
	if to.Before(from) {
		return NewDateRange(year, oto, ofrom)
	}
	return DateRange{from: from, to: to}
}

func (dr DateRange) String() string {
	if dr.LeapYearSpecific() {
		return fmt.Sprintf("%s - %s*", dr.from, dr.to)
	}
	return fmt.Sprintf("%s - %s", dr.from, dr.to)
}

// Parse ranges in formats '01:03', 'Jan:Mar', '01/02:03-04' or 'Jan-02:Mar-04'.
// If the start date has a day of zero then it is interpreted as the first day
// of the month, similary if the end date has a day of zero then it is interpreted
// as the last day of the month. The start date must be before the end date.
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
	if from.Day == 0 {
		from.Day = 1
	}
	if to.Day == 0 {
		to.Day = DaysInMonth(year, to.Month)
		d.leapYearSpecific = to.Month == 2 && (to.Day == 0 || to.Day == 29)
	}
	if to.Before(from) {
		return fmt.Errorf("from is later than to: %s %s", from, to)
	}
	d.from, d.to = from, to
	return nil
}

func (dr DateRange) Equal(dr2 DateRange) bool {
	return dr.from == dr2.from && dr.to == dr2.to
}

func (dr DateRange) EqualForYear(dr2 DateRange) bool {
	return dr.from == dr2.from && dr.to == dr2.to
}

// Dates returns an iterator that yields each date in the range for the
// given year.
func (d DateRange) Dates(year int) func(yield func(Date) bool) {
	dm := daysInMonthForYear(year)
	to := d.ToForYear(year)
	return func(yield func(Date) bool) {
		for td := d.From(); td.BeforeOrOn(to); td = td.tomorrow(dm) {
			if !yield(td) {
				return
			}
		}
	}
}

// DatesConstrained returns an iterator that yields each date in the range for the
// given year constrained by the given DateConstraints.
func (dr DateRange) DatesConstrained(year int, dc Constraints) func(yield func(Date) bool) {
	return func(yield func(Date) bool) {
		dm := daysInMonthForYear(year)
		to := dr.ToForYear(year)
		for td := dr.From(); td.BeforeOrOn(to); td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month), td.Day, 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(td) {
				return
			}
		}
	}
}

func (dr DateRange) RangesConstrained(year int, dc Constraints) func(yield func(DateRange) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(DateRange) bool) {
		start, stop := dr.From(), dr.ToForYear(year)
		to := stop
		inrange := dc.Include(time.Date(year, time.Month(start.Month), start.Day, 0, 0, 0, 0, time.UTC))
		for td := dr.From(); td.BeforeOrOn(to); td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month), td.Day, 0, 0, 0, 0, time.UTC)) {
				if inrange {
					// Range ends
					if !yield(NewDateRange(year, start, stop)) {
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
			yield(NewDateRange(year, start, stop))
		}
	}
}

// Days returns an iterator that yields each day in the range for the
// given year.
func (dr DateRange) Days(year int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		last := dr.ToForYear(year).DayOfYear(year)
		for yd := dr.From().DayOfYear(year); yd <= last; yd++ {
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
		to := dr.ToForYear(year).DayOfYear(year)
		for d := dr.From().DayOfYear(year); d <= to; d++ {
			tm := dateFromDay(d, dm)
			if !dc.Include(time.Date(year, time.Month(tm.Month), tm.Day, 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(d) {
				return
			}
		}
	}
}

// Before returns true if d is before a. If the months and start days are identical
// then the end days are used to determine the order, ie. which range ends first.
func (dr DateRange) Before(a DateRange) bool {
	if dr.from.Month != a.from.Month {
		return dr.from.Month < a.from.Month
	}
	// From.Day may be zero, but that doesn't affect the ordering.
	if dr.from.Day != a.from.Day {
		return dr.from.Day < a.from.Day
	}
	// If the start dates are identical then use the end date to determine the order.
	return dr.to.Day < a.to.Day
}

type DateRangeList []DateRange

// MergeMonthsAndRanges creates an ordered list of DateRange values from the given
// MonthList and DateRangeList.
func MergeMonthsAndRanges(year int, months MonthList, ranges DateRangeList) DateRangeList {
	drl := make(DateRangeList, 0, len(months)+len(ranges))
	for _, m := range months {
		drl = append(drl, NewDateRange(year, Date{Month: m, Day: 1}, Date{Month: m, Day: DaysInMonth(year, m)}))
	}
	drl = append(drl, ranges...)
	sort.Slice(drl, func(i, j int) bool { return drl[i].Before(drl[j]) })
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
	drs.Sort()
	*dr = drs
	return nil
}

func (dr DateRangeList) Sort() {
	sort.Slice(dr, func(i, j int) bool { return dr[i].Before(dr[j]) })
}

func (dr DateRangeList) Equal(dr2 DateRangeList) bool {
	if len(dr) != len(dr2) {
		return false
	}
	for i, d := range dr {
		if !d.Equal(dr2[i]) {
			return false
		}
	}
	return true
}

func MergeDates(year int, dates []Date) DateRangeList {
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	dm := daysInMonthForYear(year)
	var drs []DateRange
	from := dates[0]
	to := dates[0]
	for i := 1; i < len(dates); i++ {
		if dates[i-1] == dates[i] {
			continue
		}
		if dates[i-1] == dates[i].yesterday(dm) {
			to = dates[i]
			continue
		}
		drs = append(drs, NewDateRange(year, from, to))
		from = dates[i]
		to = dates[i]
	}
	drs = append(drs, NewDateRange(year, from, to))
	return drs
}

func MergeRanges(year int, ranges []DateRange) DateRangeList {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Before(ranges[j]) })
	leap := IsLeap(year)
	var merged []DateRange

	from := ranges[0].From()
	to := ranges[0].ToForYear(year)
	for i := 1; i < len(ranges); i++ {
		fd := ranges[i-1].ToForYear(year).dayOfYear(leap)
		td := ranges[i].From().dayOfYear(leap)
		if fd >= (td - 1) {
			to = ranges[i].ToForYear(year)
			continue
		}
		merged = append(merged, NewDateRange(year, from, to))
		from = ranges[i].From()
		to = ranges[i].ToForYear(year)
	}
	merged = append(merged, NewDateRange(year, from, to))
	return merged
}
