// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package schedule provides support for scheduling events based on dates and times.
package schedule

import (
	"slices"
	"sort"

	"cloudeng.io/datetime"
)

// Dates represents a set of dates expressed as a combination of
// months, date ranges and constraints on those dates (eg. weekdays in March).
type Dates struct {
	For          datetime.MonthList            // Whole months to include.
	MirrorMonths bool                          // Include the 'mirror' months of those in For.
	Ranges       datetime.DateRangeList        // Include specific date ranges.
	Dynamic      datetime.DynamicDateRangeList // Functions to generate dates that vary by year, such as solstices, seasons or holidays.
	Constraints  datetime.Constraints          // Constraints to be applied, such as weekdays/weekends etc.
}

func (d Dates) clone() Dates {
	return Dates{
		For:          slices.Clone(d.For),
		MirrorMonths: d.MirrorMonths,
		Ranges:       slices.Clone(d.Ranges),
		Dynamic:      slices.Clone(d.Dynamic),
		Constraints:  d.Constraints,
	}
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
// including the evaluation of any dynamic date ranges.
func (d Dates) EvaluateDateRanges(year int) datetime.DateRangeList {
	months := slices.Clone(d.For)
	if d.MirrorMonths {
		for _, m := range d.For {
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
	return drl
}

// Sort by due time and then by name.
func (a ActionSpecs[T]) Sort() {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Due == a[j].Due {
			return a[i].Name < a[j].Name
		}
		return a[i].Due < a[j].Due
	})
}

// Sort by due time, but preserve the order of actions with
// the same due time.
func (a ActionSpecs[T]) SortStable() {
	slices.SortStableFunc(a, func(a, b ActionSpec[T]) int {
		if a.Due < b.Due {
			return -1
		} else if a.Due > b.Due {
			return 1
		}
		return 0
	})
}

// Annual represents a schedule of actions to be taken on specific dates
// of that year. The actions are specified on a per-day basis in the form
// as a specification of the times of the day the action is to be taken.
type Annual[T any] struct {
	Name  string
	Dates Dates
	Specs ActionSpecs[T]
}

func (a Annual[T]) clone() Annual[T] {
	return Annual[T]{
		Name:  a.Name,
		Dates: a.Dates.clone(),
		Specs: slices.Clone(a.Specs),
	}
}
