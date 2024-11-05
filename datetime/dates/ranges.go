// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// DateRange represents a range of dates, inclusive of From and To.
type DateRange struct {
	From, To Date
}

// NewDateRange returns a DateRange for the from/to dates for the
// specified year.
// If the from date has a day of zero then it is interpreted as the first
// day of the month, similary if the to date has a day of zero then it is
// interpreted as the last day of the month.
// If the from date is after the to date then the dates are swapped.
func NewDateRange(year int, from, to Date) DateRange {
	if to.Before(from) {
		return NewDateRange(year, to, from)
	}
	if from.Day == 0 {
		from.Day = 1
	}
	if to.Day == 0 {
		to.Day = daysInMonthForYear(year)[to.Month-1]
	}
	return DateRange{from, to}
}

// Before returns true if date is before d. It returns false if the
// dates are equal.
func (d Date) Before(date Date) bool {
	return d.Month < date.Month || (d.Month == date.Month && d.Day < date.Day)
}

// BeforeOrOn returns true if date is before or on d. It returns true if the
func (d Date) BeforeOrOn(date Date) bool {
	return d.Month < date.Month || (d.Month == date.Month && d.Day <= date.Day)
}

func (d DateRange) String() string {
	return fmt.Sprintf("%s - %s", d.From, d.To)
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
	}
	if to.Before(from) {
		return fmt.Errorf("from is later than to: %s %s", from, to)
	}
	d.From, d.To = from, to
	return nil
}

// Dates returns an iterator that yields each date in the range for the
// given year.
func (d DateRange) Dates(year int) func(yield func(Date) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(Date) bool) {
		for td := d.From; td.BeforeOrOn(d.To); td = td.tomorrow(dm) {
			if !yield(td) {
				return
			}
		}
	}
}

// DatesConstrained returns an iterator that yields each date in the range for the
// given year constrained by the given DateConstraints.
func (d DateRange) DatesConstrained(year int, dc Constraints) func(yield func(Date) bool) {
	return func(yield func(Date) bool) {
		dm := daysInMonthForYear(year)
		for td := d.From; td.BeforeOrOn(d.To); td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month), td.Day, 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(td) {
				return
			}
		}
	}
}

func (d DateRange) RangesConstrained(year int, dc Constraints) func(yield func(DateRange) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(DateRange) bool) {
		start, stop := d.From, d.To
		inrange := dc.Include(time.Date(year, time.Month(start.Month), start.Day, 0, 0, 0, 0, time.UTC))
		for td := d.From; td.BeforeOrOn(d.To); td = td.tomorrow(dm) {
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
func (d DateRange) Days(year int) func(yield func(int) bool) {
	return func(yield func(int) bool) {
		last := d.To.DayOfYear(year)
		for yd := d.From.DayOfYear(year); yd <= last; yd++ {
			if !yield(yd) {
				return
			}
		}
	}
}

// DaysConstrained returns an iterator that yields each day in the range for the
// given year constrained by the given DateConstraints.
func (d DateRange) DaysConstrained(year int, dc Constraints) func(yield func(int) bool) {
	dm := daysInMonthForYear(year)
	return func(yield func(int) bool) {
		to := d.To.DayOfYear(year)
		for d := d.From.DayOfYear(year); d <= to; d++ {
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
func (d DateRange) Before(a DateRange) bool {
	if d.From.Month != a.From.Month {
		return d.From.Month < a.From.Month
	}
	// From.Day may be zero, but that doesn't affect the ordering.
	if d.From.Day != a.From.Day {
		return d.From.Day < a.From.Day
	}
	// If the start dates are identical then use the end date to determine the order.
	return d.To.Day < a.To.Day
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
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Before(ranges[j]) })
	leap := IsLeap(year)
	var merged []DateRange

	from := ranges[0].From
	to := ranges[0].To
	for i := 1; i < len(ranges); i++ {
		fd := ranges[i-1].To.dayOfYear(leap)
		td := ranges[i].From.dayOfYear(leap)
		if fd >= (td - 1) {
			to = ranges[i].To
			continue
		}
		merged = append(merged, NewDateRange(year, from, to))
		from = ranges[i].From
		to = ranges[i].To
	}
	merged = append(merged, NewDateRange(year, from, to))
	return merged
}
