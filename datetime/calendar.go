// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"strings"
	"time"
)

// CalendarDate represents a date with a year, month and day.
// Year is represented in the top 16 bits and Date in the lower 16 bits.
type CalendarDate uint32

func NewCalendarDate(year int, month Month, day int) CalendarDate {
	return CalendarDate(uint32(year)<<16 | uint32(month)<<8 | uint32(day))
}

func newCalendarDate8(year int, month Month, day uint8) CalendarDate {
	return CalendarDate(uint32(year)<<16 | uint32(month)<<8 | uint32(day))
}

func CalendarDateFromTime(t time.Time) CalendarDate {
	return NewCalendarDate(t.Year(), Month(t.Month()), t.Day())
}

func (cd CalendarDate) Year() int {
	return int(cd >> 16)
}

func (cd CalendarDate) Month() Month {
	return Month(cd >> 8 & 0xff)
}

func (cd CalendarDate) Day() uint8 {
	return uint8(cd & 0xff)
}

// Date returns the Date for the CalendarDate.
func (cd CalendarDate) Date() Date {
	return Date(cd & 0xffff)
}

func (cd CalendarDate) String() string {
	return fmt.Sprintf("%02d %02d %04d", Month(cd.Month()), cd.Day(), cd.Year())
}

func (cd CalendarDate) Tomorrow() CalendarDate {
	year := cd.Year()
	month := cd.Month()
	day := cd.Day()
	if month == December && day == 31 {
		return NewCalendarDate(year+1, January, 1)
	}
	if day == DaysInMonth(year, month) {
		return NewCalendarDate(year, month+1, 1)
	}
	return NewCalendarDate(year, month, int(day)+1)
}

func (cd CalendarDate) Yesterday() CalendarDate {
	year := cd.Year()
	month := cd.Month()
	day := cd.Day()
	if month == January && day == 1 {
		return NewCalendarDate(year-1, December, 31)
	}
	if day == 1 {
		return newCalendarDate8(year, month-1, DaysInMonth(year, month))
	}
	return newCalendarDate8(year, month, day-1)
}

type CalendarDateList []CalendarDate

func (cdl CalendarDateList) String() string {
	var out strings.Builder
	for i, d := range cdl {
		if i > 0 && i < len(cdl)-1 {
			out.WriteString(", ")
		}
		out.WriteString(fmt.Sprintf("%04d-%02d-%02d", d.Year(), d.Month(), d.Day()))
	}
	return out.String()
}

func (cdl CalendarDateList) Dates() DateList {
	dl := make(DateList, len(cdl))
	for i, cd := range cdl {
		dl[i] = cd.Date()
	}
	return dl
}

type CalendarDateRange uint64
