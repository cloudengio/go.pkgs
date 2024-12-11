// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

// DynamicDateRange is a function that returns a DateRange for
// a given year and is intended to be evaluated once per year
// to calculate events such as solstices, seasons or holidays.
type DynamicDateRange interface {
	Name() string
	Evaluate(year int) CalendarDateRange
}

// DynamicTimeOfDay is a function that returns a TimeOfDay for
// a given date and is intended to be evaluated once per day
// to calculate events such as sunrise, sunset etc.
type DynamicTimeOfDay interface {
	Name() string
	Evaluate(cd CalendarDate, yp Place) TimeOfDay
}

type DynamicDateRangeList []DynamicDateRange

func (dl DynamicDateRangeList) String() string {
	var out string
	for i, d := range dl {
		if i > 0 {
			out += ", "
		}
		out += d.Name()
	}
	return out
}

func (dl DynamicDateRangeList) Evaluate(year int) []CalendarDateRange {
	result := make([]CalendarDateRange, len(dl))
	for i, f := range dl {
		result[i] = f.Evaluate(year)
	}
	return result
}
