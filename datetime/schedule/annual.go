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
	For          datetime.MonthList     // Whole months to include.
	MirrorMonths bool                   // Include the 'mirror' months of those in For.
	Ranges       datetime.DateRangeList // Include specific date ranges.
	Constraints  datetime.Constraints   // Constraints to be applied, such as weekdays/weekends etc.
}

func (d Dates) clone() Dates {
	return Dates{
		For:          slices.Clone(d.For),
		MirrorMonths: d.MirrorMonths,
		Ranges:       slices.Clone(d.Ranges),
		Constraints:  d.Constraints,
	}
}

// EvaluateDateRanges returns the list of date ranges that are represented
// by the totality of the information represented by Dates instance.
func (d Dates) EvaluateDateRanges(year int) datetime.DateRangeList {
	months := slices.Clone(d.For)
	if d.MirrorMonths {
		for _, m := range d.For {
			months = append(months, datetime.MirrorMonth(m))
		}
	}
	slices.Sort(months)
	merged := d.Ranges.MergeMonths(year, months)
	drl := make(datetime.DateRangeList, 0, len(merged))
	for _, r := range merged {
		for dr := range r.RangesConstrained(year, d.Constraints) {
			drl = append(drl, dr.DateRange(year))
		}
	}
	return drl
}

// Active represents a set of actions that are 'active', ie. which are due
// to be executed according to the schedule.
type Active[T any] struct {
	Date    datetime.Date
	Actions Actions[T] // Ordered by name.
}

// Action represents an event that the schedule should trigger at a specific day and time.
type Action[T any] struct {
	Due    datetime.TimeOfDay
	Name   string
	Action T
}

type Actions[T any] []Action[T]

// Sort by due time and then by name.
func (a Actions[T]) Sort() {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Due == a[j].Due {
			return a[i].Name < a[j].Name
		}
		return a[i].Due < a[j].Due
	})
}

// Sort by due time, but preserve the order of actions with
// the same due time.
func (a Actions[T]) SortStable() {
	slices.SortStableFunc(a, func(a, b Action[T]) int {
		if a.Due < b.Due {
			return -1
		} else if a.Due > b.Due {
			return 1
		}
		return 0
	})
}

// Annual represents a schedule of actions to be taken at specific dates
// and times each year. Each action results in one or more events at a specific
// time on a specific date within any given year.
type Annual[T any] struct {
	Name    string
	Dates   Dates
	Actions Actions[T]
}

func (a Annual[T]) clone() Annual[T] {
	return Annual[T]{
		Name:    a.Name,
		Dates:   a.Dates.clone(),
		Actions: slices.Clone(a.Actions),
	}
}
