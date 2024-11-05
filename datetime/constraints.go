// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"strings"
	"time"
)

// Constraints represents constraints on date values such
// as weekends or custom dates to exclude. Custom dates take precedence
// over weekdays and weekends.
type Constraints struct {
	Weekdays       bool             // If true, include weekdays
	Weekends       bool             // If true, include weekends
	Custom         DateList         // If non-empty, exclude these dates
	CustomCalendar CalendarDateList // If non-empty, exclude these calendar dates
}

func (dc Constraints) String() string {
	var out strings.Builder
	if len(dc.Custom) > 0 || len(dc.CustomCalendar) > 0 {
		out.WriteString("excluding custom dates: ")
		out.WriteString(dc.Custom.String())
		out.WriteString(dc.CustomCalendar.String())
		out.WriteString(": ")
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
		month, day := Month(when.Month()), when.Day()
		contains := dc.Custom.Contains(Date{month, day})
		return !contains
	}
	if len(dc.CustomCalendar) > 0 {
		year, month, day := when.Year(), Month(when.Month()), when.Day()
		contains := dc.CustomCalendar.Contains(CalendarDate{year, month, day})
		return !contains
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
