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
// NewDateRange and Parse create or initialize a DateRange.
type DateRange struct {
	from, to Date
}

// From returns the start date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (dr DateRange) From(year int) Date {
	if dr.from.Month == 2 && dr.from.Day == 29 && !IsLeap(year) {
		return Date{Month: dr.from.Month, Day: 28}
	}
	return dr.from
}

// To returns the end date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (dr DateRange) To(year int) Date {
	if dr.to.Month == 2 && dr.to.Day == 29 && !IsLeap(year) {
		return Date{Month: 2, Day: 28}
	}
	return dr.to
}

// NewDateRange returns a DateRange for the from/to dates for the specified year.
// If the from day is zero then it is treated as the first day of the month.
// If the from day is 29 for a non-leap year then it is left as 29.
// If the to day is zero then it is treated as the last day of the month taking
// the year into account for Feb.
func NewDateRange(year int, from, to Date) (DateRange, error) {

	if from.Day <= 0 {
		from.Day = 1
	} else if n := DaysInMonth(year, from.Month); from.Day > n {
		from.Day = n
	}

	if n := DaysInMonth(year, to.Month); to.Day == 0 {
		to.Day = n
	} else if to.Day > n {
		to.Day = n
	}

	return DateRange{from: from, to: to}, nil
}

func (dr DateRange) String() string {
	return fmt.Sprintf("%s - %s", dr.from, dr.to)
}

// Parse ranges in formats '01:03', 'Jan:Mar', '01/02:03-04' or 'Jan-02:Mar-04'.
// If the from day is zero then it is treated as the first day of the month.
// If the from day is 29 for a non-leap year then it is left as 29.
// If the to day is zero then it is treated as the last day of the month taking
// the year into account for Feb.
// The start date must be before the end date.
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
	d.from, d.to = from, to
	return nil
}

// Equal returns true if the two DateRange values are equal
// for the given year.
func (dr DateRange) Equal(year int, dr2 DateRange) bool {
	af, bf := dr.From(year), dr2.From(year)
	at, bt := dr.To(year), dr2.To(year)
	return af == bf && at == bt
}

// Dates returns an iterator that yields each date in the range for the
// given year.
func (d DateRange) Dates(year int) func(yield func(Date) bool) {
	dm := daysInMonthForYear(year)
	to := d.To(year)
	return func(yield func(Date) bool) {
		for td := d.From(year); td.BeforeOrOn(to); td = td.tomorrow(dm) {
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
		to := dr.To(year)
		for td := dr.From(year); td.BeforeOrOn(to); td = td.tomorrow(dm) {
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
		start, stop := dr.From(year), dr.To(year)
		to := stop
		inrange := dc.Include(time.Date(year, time.Month(start.Month), start.Day, 0, 0, 0, 0, time.UTC))
		for td := dr.From(year); td.BeforeOrOn(to); td = td.tomorrow(dm) {
			if !dc.Include(time.Date(year, time.Month(td.Month), td.Day, 0, 0, 0, 0, time.UTC)) {
				if inrange {
					// Range ends
					if !yield(DateRange{from: start, to: stop}) {
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
			yield(DateRange{from: start, to: stop})
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
		drl = append(drl, DateRange{from: Date{Month: m, Day: 1}, to: Date{Month: m, Day: DaysInMonth(year, m)}})
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
		drs = append(drs, DateRange{from: from, to: to})
		from = dates[i]
		to = dates[i]
	}
	drs = append(drs, DateRange{from: from, to: to})
	return drs
}

func MergeRanges(year int, ranges []DateRange) DateRangeList {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Before(ranges[j]) })
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
		merged = append(merged, DateRange{from: from, to: to})
		from = ranges[i].From(year)
		to = ranges[i].To(year)
	}
	merged = append(merged, DateRange{from: from, to: to})
	return merged
}
