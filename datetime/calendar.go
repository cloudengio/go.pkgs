// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CalendarDate represents a date with a year, month and day.
// Year is represented in the top 16 bits and Date in the lower 16 bits.
type CalendarDate uint32

// NewCalendarDate creates a new CalendarDate from the specified year, month and day.
// Year must be in the range 0..65535, month in the range 0..12 and day in the range 0..31.
func NewCalendarDate(year int, month Month, day int) CalendarDate {
	return CalendarDate(uint32(year&0xffff)<<16 | uint32(month)<<8 | uint32(day))
}

func newCalendarDate8(year uint16, month Month, day uint8) CalendarDate {
	return CalendarDate(uint32(year&0xffff)<<16 | uint32(month)<<8 | uint32(day))
}

func parseYearAndMonth(y, m string, numeric bool) (uint16, Month, error) {
	tmp, err := strconv.ParseUint(y, 10, 16)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %s: %v", y, err)
	}
	year := uint16(tmp & 0xffff)
	var month Month
	if numeric {
		month, err = ParseNumericMonth(m)
	} else {
		month, err = ParseMonth(m)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month: %s: %v", m, err)
	}
	return year, month, nil
}

func parseDay(y uint16, m Month, d string) (uint8, error) {
	tmp, err := strconv.ParseUint(d, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid day: %s: %v", d, err)
	}
	day := uint8(tmp & 0xff)
	if day < 1 || day > DaysInMonth(int(y), m) {
		return 0, fmt.Errorf("invalid day for %v %v: %d", day, m, day)
	}
	return day, nil
}

// ParseNumericCalendarDate a numeric calendar date in formats '01/02/2006' with error checking
// for valid month and day.
func ParseNumericCalendarDate(val string) (CalendarDate, error) {
	parts := strings.Split(val, "/")
	if len(parts) != 3 {
		return CalendarDate(0), fmt.Errorf("invalid value %q, expected format '01/02/2006'", val)
	}
	year, month, err := parseYearAndMonth(parts[2], parts[0], true)
	if err != nil {
		return CalendarDate(0), err
	}
	day, err := parseDay(year, month, parts[1])
	if err != nil {
		return CalendarDate(0), err
	}
	return newCalendarDate8(year, month, day), nil
}

// ParseCalendarDate a numeric calendar date in formats 'Jan-02-2006' with error checking
// for valid month and day.
func ParseCalendarDate(val string) (CalendarDate, error) {
	parts := strings.Split(val, "-")
	if len(parts) != 3 {
		return CalendarDate(0), fmt.Errorf("invalid value %q, expected format 'Jan-02-2006'", val)
	}
	year, month, err := parseYearAndMonth(parts[2], parts[0], false)
	if err != nil {
		return CalendarDate(0), err
	}
	day, err := parseDay(year, month, parts[1])
	if err != nil {
		return CalendarDate(0), err
	}
	return newCalendarDate8(year, month, day), nil
}

const expectedCalendarDateFormats = "01/02/2006, 02/2006, Jan-02-2024, Jan-2006"

func (cd *CalendarDate) Parse(val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value, expected %s", expectedCalendarDateFormats)
	}
	parts := strings.Split(val, "/")
	switch len(parts) {
	case 2:
		year, month, err := parseYearAndMonth(parts[1], parts[0], true)
		if err != nil {
			return err
		}
		*cd = newCalendarDate8(year, month, 0)
		return nil
	case 3:
		ncd, err := ParseNumericCalendarDate(val)
		if err != nil {
			return fmt.Errorf("invalid numeric calendar date: %v", err)
		}
		*cd = ncd
		return nil
	}
	parts = strings.Split(val, "-")
	switch len(parts) {
	case 2:
		year, month, err := parseYearAndMonth(parts[1], parts[0], false)
		if err != nil {
			return err
		}
		*cd = newCalendarDate8(year, month, 0)
		return nil
	case 3:
		ncd, err := ParseCalendarDate(val)
		if err != nil {
			return fmt.Errorf("invalid calendar date: %v", err)
		}
		*cd = ncd
		return nil
	}
	return fmt.Errorf("invalid input %q expected %s", val, expectedCalendarDateFormats)
}

// ParseAnyDate parses a date in the format '01/02/2006', 'Jan-02-2006', '01/02', 'Jan-02' or '01'.
// The year argument is ignored for the '01/02/2006' and 'Jan-02-2006' formats.
// Jan-02, 01/02 are treated as month and day and the year argument is used to set the year.
func ParseAnyDate(year int, val string) (CalendarDate, error) {
	switch strings.Count(val, "/") {
	case 0:
	case 1:
		d, err := ParseNumericDate(val)
		if err != nil {
			return CalendarDate(0), err
		}
		return NewCalendarDate(year, d.Month(), d.Day()), nil
	case 2:
		return ParseNumericCalendarDate(val)
	default:
		return CalendarDate(0), fmt.Errorf("invalid date: %q, expected formats %s or %s", val, expectedCalendarDateFormats, expectedDateFormats)
	}
	switch strings.Count(val, "-") {
	case 0:
	case 1:
		d, err := ParseDate(val)
		if err != nil {
			return CalendarDate(0), err
		}
		return NewCalendarDate(year, d.Month(), d.Day()), nil
	case 2:
		return ParseCalendarDate(val)
	default:
		return CalendarDate(0), fmt.Errorf("invalid date: %q, expected formats %s or %s", val, expectedCalendarDateFormats, expectedDateFormats)
	}
	var m Month
	if err := m.Parse(val); err == nil {
		return NewCalendarDate(year, m, 0), nil
	}
	return CalendarDate(0), fmt.Errorf("invalid date: %q, expected formats %s or %s", val, expectedCalendarDateFormats, expectedDateFormats)
}

// Normalize adjusts the date for the given year. If the day is zero
// then firstOfMonth is used to determine the interpretation of the day.
// If firstOfMonth is true then the day is set to the first day of the month,
// otherwise it is set to the last day of the month.
// Month is normalized to be in the range 1-12.
func (cd CalendarDate) Normalize(firstOfMonth bool) CalendarDate {
	year := cd.Year()
	month := max(cd.Month(), 1)
	month = min(month, 12)
	dm := daysInMonthForYear(year)[month-1]
	day := cd.day8()
	if day == 0 {
		if firstOfMonth {
			day = 1
		} else {
			day = dm
		}
	}
	day = min(day, dm)
	return NewCalendarDate(year, month, int(day))
}

// CalendarDateFromTime creates a new CalendarDate from the specified time.Time.
func CalendarDateFromTime(t time.Time) CalendarDate {
	return NewCalendarDate(t.Year(), Month(t.Month()), t.Day())
}

func (cd CalendarDate) Year() int {
	return int(cd >> 16 & 0xffff)
}

func (cd CalendarDate) year16() uint16 {
	return uint16(cd >> 16 & 0xffff)
}

func (cd CalendarDate) Month() Month {
	return Month(cd >> 8 & 0xff)
}

func (cd CalendarDate) Day() int {
	return int(cd & 0xff)
}

func (cd CalendarDate) day8() uint8 {
	return uint8(cd & 0xff)
}

// Date returns the Date for the CalendarDate.
func (cd CalendarDate) Date() Date {
	return Date(cd & 0xffff)
}

func (cd CalendarDate) DayOfYear() int {
	return calcDayOfYear(uint8(cd>>8&0xff), uint8(cd&0xff), IsLeap(cd.Year()))
}

func (cd CalendarDate) YearDay() YearDay {
	return NewYearDay(cd.Year(), cd.DayOfYear())
}

func (cd CalendarDate) String() string {
	return fmt.Sprintf("%02d %02d %04d", cd.Month(), cd.Day(), cd.Year())
}

// IsDST returns true if the date is within daylight savings time for the specified
// location assuming that the time is 12:00 hours. DST generally starts at 2am and ends
// at 3am.
func (cd CalendarDate) IsDST(loc *time.Location) bool {
	return time.Date(cd.Year(), time.Month(cd.Month()), cd.Day(), 12, 0, 0, 0, loc).IsDST()
}

// Include returns true if the specified date is within the range.
func (cd CalendarDateRange) Include(d CalendarDate) bool {
	return cd.fromDate() <= d && cd.toDate() >= d
}

// Tomorrow returns the CalendarDate for the day after the specified date, wrapping
// to the next month or year as needed.
func (cd CalendarDate) Tomorrow() CalendarDate {
	year := cd.Year()
	month := cd.Month()
	day := cd.Day()
	if month == December && day == 31 {
		return NewCalendarDate(year+1, January, 1)
	}
	if day == int(DaysInMonth(year, month)) {
		return NewCalendarDate(year, month+1, 1)
	}
	return NewCalendarDate(year, month, day+1)
}

// Yesterday returns the CalendarDate for the day before the specified date, wrapping
// to the previous month or year as needed.
func (cd CalendarDate) Yesterday() CalendarDate {
	year := cd.year16()
	month := cd.Month()
	day := cd.Day()
	if month == January && day == 1 {
		return newCalendarDate8(year-1, December, 31)
	}
	if day == 1 {
		return newCalendarDate8(year, month-1, DaysInMonth(int(year), month))
	}
	return newCalendarDate8(year, month, uint8(day&0xff)-1)
}

// CalendarDateList represents a list of CalendarDate values, it can sorted and
// searched using the slices package.
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
