// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"fmt"
	"strings"

	"cloudeng.io/datetime"
)

func daysFromDatesString(year int, datelist string) []datetime.YearDay {
	parts := strings.Split(datelist, ",")
	days := make([]datetime.YearDay, 0, len(parts))
	for _, p := range parts {
		var date datetime.Date
		if len(p) == 0 {
			continue
		}
		if err := date.Parse(p); err != nil {
			panic(err)
		}
		days = append(days, datetime.NewYearDay(year, date.DayOfYear(year)))
	}
	return days
}

func daysFromCalendarDatesString(datelist string) []datetime.YearDay {
	parts := strings.Split(datelist, ",")
	days := make([]datetime.YearDay, 0, len(parts))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		var date datetime.CalendarDate
		if err := date.Parse(p); err != nil {
			panic(err)
		}
		days = append(days, date.YearDay())
	}
	return days
}

func newCalendarDateRange(a, b datetime.CalendarDate) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(a, b)
}

func newDateList(d ...datetime.Date) datetime.DateList {
	r := make(datetime.DateList, len(d))
	copy(r, d)
	return r
}

func newCalendarDateList(d ...datetime.CalendarDate) datetime.CalendarDateList {
	r := make([]datetime.CalendarDate, len(d))
	copy(r, d)
	return r
}

func newDateRangeList(d ...datetime.Date) datetime.DateRangeList {
	r := make([]datetime.DateRange, 0, len(d)/2)
	for i := 0; i < len(d); i += 2 {
		r = append(r, datetime.NewDateRange(d[i], d[i+1]))
	}
	return r
}

func newCalendarDateRangeList(d ...datetime.CalendarDate) datetime.CalendarDateRangeList {
	r := make([]datetime.CalendarDateRange, 0, len(d)/2)
	for i := 0; i < len(d); i += 2 {
		r = append(r, datetime.NewCalendarDateRange(d[i], d[i+1]))
	}
	return r
}

func newDate(m, d int) datetime.Date {
	return datetime.NewDate(datetime.Month(m), d)
}

func newCalendarDate(y, m, d int) datetime.CalendarDate {
	return datetime.NewCalendarDate(y, datetime.Month(m), d)
}

func newTimeOfDayList(tods ...datetime.TimeOfDay) datetime.TimeOfDayList {
	r := make([]datetime.TimeOfDay, len(tods))
	copy(r, tods)
	return r
}

type dateList []datetime.CalendarDate

func (dr *dateList) String() string {
	var out strings.Builder
	for _, d := range *dr {
		fmt.Fprintf(&out, "%02d/%02d/%04d,", d.Month(), d.Day(), d.Year())
	}
	if out.Len() == 0 {
		return ""
	}
	return out.String()[:out.Len()-1]
}

func appendYearToDates(year int, val string) string {
	y := fmt.Sprintf("/%d", year)
	parts := strings.Split(val, ",")
	for i, part := range parts {
		parts[i] = part + y
	}
	return strings.Join(parts, ",")
}

func appendYearToRanges(year int, val string) string {
	ranges := strings.Split(val, ",")
	y := fmt.Sprintf("/%d", year)
	out := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts := strings.Split(r, ":")
		for i, part := range parts {
			parts[i] = part + y
		}
		out = append(out, strings.Join(parts, ":"))
	}
	return strings.Join(out, ",")
}

func datesAsString(m, d int) string {
	var s strings.Builder
	for i := 1; i <= d; i++ {
		fmt.Fprintf(&s, "%02d/%02d,", m, i)
	}
	return s.String()
}

func calendarDatesAsString(y, m, d int) string {
	var s strings.Builder
	for i := 1; i <= d; i++ {
		fmt.Fprintf(&s, "%02d/%02d/%04d,", m, i, y)
	}
	return s.String()
}

func calendarMonthsAsString(y int, months ...int) string {
	var s strings.Builder
	for _, m := range months {
		for d := 1; d <= int(datetime.DaysInMonth(y, datetime.Month(m))); d++ {
			fmt.Fprintf(&s, "%02d/%02d/%04d,", m, d, y)
		}
	}
	return s.String()
}
