// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates

import (
	"strings"
	"time"
)

// DateConstraints represents constraints on date values such
// as weekends or custom dates. Custom dates take precedence
// over weekdays and weekends.
type DateConstraints struct {
	Weekdays      bool     // If true, include weekdays
	Weekends      bool     // If true, include weekends
	ExcludeCustom bool     // If true, exclude custom dates
	Custom        DateList // If non-empty, include these dates
}

func (dc DateConstraints) String() string {
	var out strings.Builder
	if len(dc.Custom) > 0 {
		if dc.ExcludeCustom {
			out.WriteString("excluding custom dates: ")
		} else {
			out.WriteString("including custom dates: ")
		}
		for i, d := range dc.Custom {
			if i > 0 && i < len(dc.Custom)-1 {
				out.WriteString(", ")
			}
			out.WriteString(d.String())
		}
		out.WriteString(" : ")
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
func (dc DateConstraints) Include(when time.Time) bool {
	if len(dc.Custom) > 0 {
		month, day := Month(when.Month()), when.Day()
		if !dc.ExcludeCustom {
			for _, d := range dc.Custom {
				if d.Month == month && d.Day == day {
					return true
				}
			}
		} else {
			for _, d := range dc.Custom {
				if d.Month == month && d.Day == day {
					return false
				}
			}
			return true
		}
	}
	switch {
	case dc.Weekdays && dc.Weekends:
		return true
	case dc.Weekdays:
		return when.Weekday() >= time.Monday && when.Weekday() <= time.Friday
	case dc.Weekends:
		return when.Weekday() == time.Sunday || when.Weekday() == time.Saturday
	}
	return false
}
