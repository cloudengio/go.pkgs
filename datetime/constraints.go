// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"slices"
	"strings"
	"time"
)

// Constraints represents constraints on date values such
// as weekends or custom dates to exclude. Custom dates take precedence
// over weekdays and weekends.
type Constraints struct {
	Weekdays       bool                 // If true, include weekdays
	Weekends       bool                 // If true, include weekends
	Custom         DateList             // If non-empty, exclude these dates
	CustomCalendar CalendarDateList     // If non-empty, exclude these calendar dates
	Dynamic        DynamicDateRangeList // If non-nil, exclude dates based on the evaluation of the dynamic date range functions.
}

func (dc Constraints) String() string {
	var out strings.Builder
	if len(dc.Custom) > 0 || len(dc.CustomCalendar) > 0 {
		out.WriteString("excluding custom dates: ")
		out.WriteString(dc.Custom.String())
		out.WriteString(dc.CustomCalendar.String())
		out.WriteString(": ")
	}
	if len(dc.Dynamic) > 0 {
		out.WriteString("excluding dynamic dates: ")
		out.WriteString(dc.Dynamic.String())
	}
	switch {
	case dc.Weekdays && dc.Weekends:
		out.WriteString("everyday")
	case !dc.Weekdays && !dc.Weekends:
		break
	case dc.Weekdays && !dc.Weekends:
		out.WriteString("weekdays only")
	case !dc.Weekdays && dc.Weekends:
		out.WriteString("weekends only")
	}
	return out.String()
}

// Include returns true if the given date satisfies the constraints.
// Custom dates are evaluated before weekdays and weekends.
// An empty set Constraints will return true, ie. include all dates.
func (dc Constraints) Include(when time.Time) bool {
	if len(dc.Custom) > 0 {
		return !slices.Contains(dc.Custom, DateFromTime(when))
	}
	if len(dc.CustomCalendar) > 0 {
		return !slices.Contains(dc.CustomCalendar, CalendarDateFromTime(when))
	}
	if len(dc.Dynamic) > 0 {
		cd := CalendarDateFromTime(when)
		for _, d := range dc.Dynamic {
			dd := d.Evaluate(when.Year())
			if dd.Include(cd) {
				return false
			}
		}
		return true
	}
	switch {
	case dc.Weekdays && dc.Weekends:
		return true
	case dc.Weekdays:
		return when.Weekday() >= time.Monday && when.Weekday() <= time.Friday
	case dc.Weekends:
		return when.Weekday() == time.Sunday || when.Weekday() == time.Saturday
	}
	return true
}

func (dc Constraints) Empty() bool {
	return !dc.Weekdays && !dc.Weekends && len(dc.Custom) == 0 && len(dc.CustomCalendar) == 0
}
