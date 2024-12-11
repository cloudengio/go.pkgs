// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"
)

// CalendarDateRange represents a range of CalendarDate values.
type CalendarDateRange uint64

func (cdr CalendarDateRange) from() (uint16, Month, uint8) {
	return uint16(cdr >> 48 & 0xffff), Month(cdr >> 24 & 0xff), uint8(cdr >> 8 & 0xff)
}

func (cdr CalendarDateRange) to() (uint16, Month, uint8) {
	return uint16(cdr >> 32 & 0xffff), Month(cdr >> 16 & 0xff), uint8(cdr & 0xff)
}

func (cdr CalendarDateRange) fromDate() CalendarDate {
	return newCalendarDate8(uint16(cdr>>48&0xffff), Month(cdr>>24&0xff), uint8(cdr>>8&0xff))
}

func (cdr CalendarDateRange) toDate() CalendarDate {
	return newCalendarDate8(uint16(cdr>>32&0xffff), Month(cdr>>16&0xff), uint8(cdr&0xff))
}

func newCalendarDateRange(from, to CalendarDate) CalendarDateRange {
	fy, fm, fd := CalendarDateRange(from.Year()), CalendarDateRange(from.Month()), CalendarDateRange(from.Day())
	ty, tm, td := CalendarDateRange(to.Year()), CalendarDateRange(to.Month()), CalendarDateRange(to.Day())
	return fy<<48 | ty<<32 | fm<<24 | tm<<16 | fd<<8 | td
}

// From returns the start date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (cdr CalendarDateRange) From() CalendarDate {
	fromYear, fromMonth, fromDay := cdr.from()
	if fromMonth == 2 && fromDay == 29 && !IsLeap(int(fromYear)) {
		return newCalendarDate8(fromYear, fromMonth, 28)
	}
	return newCalendarDate8(fromYear, fromMonth, fromDay)
}

// To returns the end date of the range for the specified year.
// Feb 29 is returned as Feb 28 for non-leap years.
func (cdr CalendarDateRange) To() CalendarDate {
	toYear, toMonth, toDay := cdr.to()
	if toMonth == 2 && toDay == 29 && !IsLeap(int(toYear)) {
		return newCalendarDate8(toYear, 2, 28)
	}
	return newCalendarDate8(toYear, toMonth, toDay)
}

// OnOrAfter returns a new DateRange with the from date set to on
// or after the specified date.
func (cdr CalendarDateRange) OnOrAfter(start CalendarDate) CalendarDateRange {
	if cdr.fromDate() >= start {
		return cdr
	}
	if start > cdr.toDate() {
		return CalendarDateRange(0)
	}
	return newCalendarDateRange(start, cdr.toDate())
}

// OnOrBefore returns a new DateRange with the to date set to on
// or before the specified date.
func (cdr CalendarDateRange) OnOrBefore(end CalendarDate) CalendarDateRange {
	if cdr.toDate() <= end {
		return cdr
	}
	if end < cdr.fromDate() {
		return CalendarDateRange(0)
	}
	return newCalendarDateRange(cdr.fromDate(), end)
}

// NewCalendarDateRange returns a CalendarDateRange for the from/to dates.
// If the from date is later than the to date then they are swapped.
// The resulting from and to dates are then normalized using
// calendardate.Normalize(year, true) for the from date and calendardate.Normalize(year, false) for the to date.
func NewCalendarDateRange(from, to CalendarDate) CalendarDateRange {
	if from > to {
		from, to = to, from
	}
	from, to = from.Normalize(true), to.Normalize(false)
	return newCalendarDateRange(from, to)
}

func (cdr CalendarDateRange) String() string {
	return fmt.Sprintf("%s - %s", cdr.fromDate().String(), cdr.toDate().String())
}

// Dates returns an iterator that yields each Date in the range for the
// given year.
func (cdr CalendarDateRange) Dates() iter.Seq[CalendarDate] {
	to := cdr.To()
	return func(yield func(CalendarDate) bool) {
		for td := cdr.From(); td <= to; td = td.Tomorrow() {
			if !yield(td) {
				return
			}
		}
	}
}

// Parse ranges in formats '01/2006:03/2007', 'Jan-2006:Mar-2007',
// '01/02/2006:03/04/2007' or 'Jan-02-2006:Mar-04-2007',
// If the from day is zero then it is treated as the first day of the month.
// If the from day is 29 for a non-leap year then it is left as 29.
// If the to day is zero then it is treated as the last day of the month taking
// the year into account for Feb.
// The start date must be before the end date after normalization as per
// the above rules.
func (cdr *CalendarDateRange) Parse(val string) error {
	parts := strings.Split(val, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, %q expected '<from>:<to>'", val)
	}
	var from, to CalendarDate
	if err := from.Parse(parts[0]); err != nil {
		return fmt.Errorf("invalid from: %s: %v", parts[0], err)
	}
	if err := to.Parse(parts[1]); err != nil {
		return fmt.Errorf("invalid to: %s: %v", parts[1], err)
	}
	from = from.Normalize(true)
	to = to.Normalize(false)
	if to < from {
		return fmt.Errorf("from is later than to: %s %s", from, to)
	}
	*cdr = newCalendarDateRange(from, to)
	return nil
}

func (cdr CalendarDateRange) DateRange(year int) DateRange {
	if cdr.From().Year() != year || cdr.To().Year() != year {
		return DateRange(0)
	}
	return NewDateRange(cdr.From().Date(), cdr.To().Date())
}

// Truncate returns a DateRange that is truncated to
// the start or end of specified year iff the range spans
// consecutive years, otherwise it returns DateRange(0).
func (cdr CalendarDateRange) Truncate(year int) DateRange {
	fy, ty := cdr.From().Year(), cdr.To().Year()
	if fy == year && ty == year {
		return NewDateRange(cdr.From().Date(), cdr.To().Date())
	}
	if fy == year && ty == year+1 {
		return NewDateRange(cdr.From().Date(), NewCalendarDate(year, 12, 31).Date())
	}
	if fy == year-1 && ty == year {
		return NewDateRange(NewCalendarDate(year, 1, 1).Date(), cdr.To().Date())
	}
	return DateRange(0)
}

// DatesConstrained returns an iterator that yields each Date in the range for the
// given year constrained by the given DateConstraints.
func (cdr CalendarDateRange) DatesConstrained(dc Constraints) iter.Seq[CalendarDate] {
	return func(yield func(CalendarDate) bool) {
		to := cdr.To()
		for td := cdr.From(); td <= to; td = td.Tomorrow() {
			if !dc.Include(time.Date(td.Year(), time.Month(td.Month()), td.Day(), 0, 0, 0, 0, time.UTC)) {
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
func (cdr CalendarDateRange) RangesConstrained(dc Constraints) iter.Seq[CalendarDateRange] {
	return func(yield func(CalendarDateRange) bool) {
		start, stop := cdr.From(), cdr.To()
		to := stop
		inrange := dc.Include(time.Date(start.Year(), time.Month(start.Month()), start.Day(), 0, 0, 0, 0, time.UTC))
		for td := cdr.From(); td <= to; td = td.Tomorrow() {
			if !dc.Include(time.Date(td.Year(), time.Month(td.Month()), td.Day(), 0, 0, 0, 0, time.UTC)) {
				if inrange {
					// Range ends
					if !yield(newCalendarDateRange(start, stop)) {
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
			yield(newCalendarDateRange(start, stop))
		}
	}
}

// Days returns an iterator that yields each day in the range for the
// given year.
func (cdr CalendarDateRange) Days() iter.Seq[YearDay] {
	to := cdr.To()
	return func(yield func(YearDay) bool) {
		for td := cdr.From(); td <= to; td = td.Tomorrow() {
			year := td.Year()
			if !yield(NewYearDay(year, td.Date().DayOfYear(year))) {
				return
			}
		}
	}
}

// DaysConstrained returns an iterator that yields each day in the range for the
// given year constrained by the given DateConstraints.
func (cdr CalendarDateRange) DaysConstrained(dc Constraints) iter.Seq[YearDay] {
	to := cdr.To()
	return func(yield func(YearDay) bool) {
		for td := cdr.From(); td <= to; td = td.Tomorrow() {
			year := td.Year()
			if !dc.Include(time.Date(year, time.Month(td.Month()), td.Day(), 0, 0, 0, 0, time.UTC)) {
				continue
			}
			if !yield(NewYearDay(year, td.Date().DayOfYear(year))) {
				return
			}
		}
	}
}

type CalendarDateRangeList []CalendarDateRange

// Parse parses a list of ranges in the format expected by CalendarDateRange.Parse.
func (cdrl *CalendarDateRangeList) Parse(ranges []string) error {
	if len(ranges) == 0 {
		return nil
	}
	drs := make(CalendarDateRangeList, 0, len(ranges))
	seen := map[CalendarDateRange]struct{}{}
	for _, rg := range ranges {
		var cdr CalendarDateRange
		if err := cdr.Parse(rg); err != nil {
			return err
		}
		if _, ok := seen[cdr]; ok {
			continue
		}
		drs = append(drs, cdr)
		seen[cdr] = struct{}{}
	}
	slices.Sort(drs)
	*cdrl = drs
	return nil
}

// Merge returns a new list of date ranges that contains merged consecutive
// calendar dates into ranges. The dates are normalized using date.Normalize(true).
// The date list is assumed to be sorted.
func (cdl CalendarDateList) Merge() CalendarDateRangeList {
	if len(cdl) == 0 {
		return nil
	}
	var cdrl CalendarDateRangeList
	from := cdl[0].Normalize(true)
	to := cdl[0].Normalize(true)
	for i := 1; i < len(cdl); i++ {
		prev, cur := cdl[i-1].Normalize(true), cdl[i].Normalize(true)
		if prev == cur {
			continue
		}
		if prev == cur.Yesterday() {
			to = cur
			continue
		}
		cdrl = append(cdrl, newCalendarDateRange(from, to))
		from = cur
		to = cur
	}
	return slices.Clip(append(cdrl, newCalendarDateRange(from, to)))
}

// Merge returns a new list of date ranges that contains merged consecutive
// overlapping ranges.
// The date list is assumed to be sorted.
func (cdrl CalendarDateRangeList) Merge() CalendarDateRangeList {
	if len(cdrl) == 0 {
		return cdrl
	}
	merged := make(CalendarDateRangeList, 0, len(cdrl))
	from := cdrl[0].From()
	to := cdrl[0].To()
	for i := 1; i < len(cdrl); i++ {
		prevTo, curFrom := cdrl[i-1].To(), cdrl[i].From()
		if prevTo.YearDay() >= (curFrom.YearDay()-1) || curFrom.Yesterday() == prevTo {
			to = max(cdrl[i].To(), prevTo)
			continue
		}
		merged = append(merged, newCalendarDateRange(from, to))
		from = curFrom
		to = max(cdrl[i].To(), prevTo)
	}
	return slices.Clip(append(merged, newCalendarDateRange(from, to)))
}

// MergeMonths returns a merged list of date ranges that contains the specified
// months for the given year.
func (cdrl CalendarDateRangeList) MergeMonths(year int, months MonthList) CalendarDateRangeList {
	ncdrl := make(CalendarDateRangeList, 0, len(months)+len(cdrl))
	for _, m := range months {
		ncdrl = append(ncdrl, newCalendarDateRange(NewCalendarDate(year, m, 1), NewCalendarDate(year, m, int(DaysInMonth(year, m)))))
	}
	ncdrl = append(ncdrl, cdrl...)
	slices.Sort(ncdrl)
	return ncdrl.Merge()
}

func (cdrl CalendarDateRangeList) OnOrAfter(start CalendarDate) CalendarDateRangeList {
	if len(cdrl) == 0 {
		return cdrl
	}
	ndr := make(CalendarDateRangeList, len(cdrl))
	for i, dr := range cdrl {
		ndr[i] = dr.OnOrAfter(start)
	}
	return ndr
}

func (cdrl CalendarDateRangeList) OnOrBefore(end CalendarDate) CalendarDateRangeList {
	if len(cdrl) == 0 {
		return cdrl
	}
	ncdr := make(CalendarDateRangeList, len(cdrl))
	for i, dr := range cdrl {
		ncdr[i] = dr.OnOrBefore(end)
	}
	return ncdr
}
