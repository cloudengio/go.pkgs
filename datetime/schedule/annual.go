// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package schedule provides support for scheduling events based on dates and times.
package schedule

import (
	"slices"
	"strings"

	"cloudeng.io/datetime"
)

// Dates represents a set of dates expressed as a combination of
// months, date ranges and constraints on those dates (eg. weekdays in March).
type Dates struct {
	Months       datetime.MonthList            // Whole months to include.
	MirrorMonths bool                          // Include the 'mirror' months of those in For.
	Ranges       datetime.DateRangeList        // Include specific date ranges.
	Dynamic      datetime.DynamicDateRangeList // Functions to generate dates that vary by year, such as solstices, seasons or holidays.
	Constraints  datetime.Constraints          // Constraints to be applied, such as weekdays/weekends etc.
}

func truncateCalendarDates(year int, cdrl datetime.CalendarDateRangeList) datetime.DateRangeList {
	dr := make(datetime.DateRangeList, len(cdrl))
	for i, cdr := range cdrl {
		dr[i] = cdr.Truncate(year)
	}
	return dr
}

// EvaluateDateRanges returns the list of date ranges that are represented
// by the totality of the information represented by Dates instance,
// including the evaluation of dynamic date ranges. The result is bounded
// by supplied bounds date range.
func (d Dates) EvaluateDateRanges(year int, bounds datetime.DateRange) datetime.DateRangeList {
	months := slices.Clone(d.Months)
	if d.MirrorMonths {
		for _, m := range d.Months {
			months = append(months, datetime.MirrorMonth(m))
		}
	}
	slices.Sort(months)
	merged := d.Ranges.MergeMonths(year, months)
	merged = append(merged, truncateCalendarDates(year, d.Dynamic.Evaluate(year))...)
	slices.Sort(merged)
	merged = merged.Merge(year)
	drl := make(datetime.DateRangeList, 0, len(merged))
	for _, r := range merged {
		for dr := range r.RangesConstrained(year, d.Constraints) {
			drl = append(drl, dr.DateRange(year))
		}
	}
	return drl.Bound(year, bounds)
}

func (d Dates) String() string {
	var out strings.Builder
	if len(d.Months) > 0 {
		out.WriteString("months: ")
		out.WriteString(d.Months.String())
		out.WriteRune('\n')
	}
	if d.MirrorMonths {
		out.WriteString("mirror\n")
	}
	if len(d.Ranges) > 0 {
		out.WriteString("ranges: ")
		out.WriteString(d.Ranges.String())
		out.WriteRune('\n')
	}
	if len(d.Dynamic) > 0 {
		out.WriteString("dynamic: ")
		out.WriteString(d.Dynamic.String())
		out.WriteRune('\n')
	}
	if c := d.Constraints.String(); len(c) > 0 {
		out.WriteRune('\n')
	}
	return out.String()
}
