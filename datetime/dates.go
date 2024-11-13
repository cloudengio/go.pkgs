// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package datetime provides support for working with dates, the time of day
// and associated ranges.
package datetime

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Month as a uint8
type Month uint8

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

func (m Month) String() string {
	return time.Month(m).String()
}

// ParseNumericMonth parses a 1 or 2 digit numeric month value in the range 1-12.
func ParseNumericMonth(val string) (Month, error) {
	n, err := strconv.ParseUint(val, 10, 8)
	if err != nil {
		return 0, err
	}
	if n < 1 || n > 12 {
		return 0, fmt.Errorf("invalid month: %d", n)
	}
	return Month(n), nil
}

// ParseMonth parses a month name of the form "Jan" to "Dec" or any other longer
// prefixes of "January" to "December" in either lower or upper case.
func ParseMonth(val string) (Month, error) {
	lc := strings.ToLower(val)
	for i := range months {
		if strings.HasPrefix(months[i], lc) {
			return Month(i + 1), nil
		}
	}
	return 0, fmt.Errorf("invalid month: %s", val)
}

// Parse parses a month in either numeric or month name format.
func (m *Month) Parse(val string) error {
	if n, err := ParseNumericMonth(val); err == nil {
		*m = n
		return nil
	}
	n, err := ParseMonth(val)
	if err != nil {
		return err
	}
	*m = n
	return nil
}

// Date as Month and Day. Use CalendarDate to specify a year.
// Day may be zero in which case the Date refers to the entire month with the
// interpretation of the day determined by the context (eg. a start date may refer
// to the first day of the month, and an end date may refer to the last day of the month).

// Date as a uint16 with the month in the high byte and the day in the low byte.
type Date uint16

// NewDate returns a Date for the given month and day. Both are assumed to
// valid for the context in which they are used. Normalize should be used to
// to adjust for a given year and interpretation of zero day value.
func NewDate(month Month, day int) Date {
	return Date(month)&0xff<<8 | Date(uint8(day&0xff))
}

func newDate8(month Month, day uint8) Date {
	return Date(month)&0xff<<8 | Date(day&0xff)
}

// Month returns the month for the date.
func (d Date) Month() Month {
	return Month(d >> 8)
}

// Day returns the day for the date.
func (d Date) Day() int {
	return int(d & 0xff)
}

// DateFromTime returns the Date for the given time.Time.
func DateFromTime(when time.Time) Date {
	return NewDate(Month(when.Month()), when.Day())
}

func (d Date) String() string {
	return fmt.Sprintf("%s %02d", d.Month().String(), d.Day())
}

func (d Date) CalendarDate(year int) CalendarDate {
	return NewCalendarDate(year, d.Month(), d.Day())
}

// Normalize adjusts the date for the given year. If the day is zero
// then firstOfMonth is used to determine the interpretation of the day.
// If firstOfMonth is true then the day is set to the first day of the month,
// otherwise it is set to the last day of the month.
// Month is normalized to be in the range 1-12.
func (d Date) Normalize(year int, firstOfMonth bool) Date {
	month := max(Month(d>>8), 1)
	month = min(month, 12)
	dm := daysInMonthForYear(year)[month-1]
	day := uint8(d & 0xff)
	if day == 0 {
		if firstOfMonth {
			day = 1
		} else {
			day = dm
		}
	}
	day = min(day, dm)
	return newDate8(month, day)
}

// Parse a numeric date in the format '01/02' with error checking
// for valid month and day.
func ParseNumericDate(year int, val string) (Date, error) {
	parts := strings.Split(val, "/")
	if len(parts) != 2 {
		return Date(0), fmt.Errorf("invalid value %q, expected format '01/02'", val)
	}
	tmp, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return Date(0), err
	}
	month := Month(tmp & 0xff)
	tmp, err = strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return Date(0), fmt.Errorf("invalid day: %s", parts[1])
	}
	day := uint8(tmp & 0xff)
	if day < 1 || day > DaysInMonth(year, month) {
		return Date(0), fmt.Errorf("invalid day for %v %v: %d", day, month, day)
	}
	return newDate8(month, day), nil
}

// ParseDate parses a date in the forma 'Jan-02' with error checking
// for valid month and day.
func ParseDate(year int, val string) (Date, error) {
	parts := strings.Split(val, "-")
	if len(parts) != 2 {
		return Date(0), fmt.Errorf("invalid date %q, expected format 'Jan-02'", val)
	}
	month, err := ParseMonth(parts[0])
	if err != nil {
		return Date(0), fmt.Errorf("invalid month: %s: %v", parts[0], err)
	}
	tmp, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil {
		return Date(0), fmt.Errorf("invalid day: %s: %v", parts[1], err)
	}
	day := uint8(tmp)
	if day < 1 || day > DaysInMonth(year, month) {
		return Date(0), fmt.Errorf("invalid day %v for %s in %d", day, month, year)
	}
	return newDate8(month, day), nil
}

const expectedDateFormats = "01, Jan, 01/02 or Jan-02"

// Parse date in formats '01', 'Jan','01/02' or 'Jan-02'. The year
// is required for correct validation of Feb.
func (d *Date) Parse(year int, val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value, expected %s", expectedDateFormats)
	}
	if strings.Contains(val, "/") {
		date, err := ParseNumericDate(year, val)
		if err != nil {
			return fmt.Errorf("invalid numeric date: %v", err)
		}
		*d = date
		return nil
	}
	if strings.Contains(val, "-") {
		date, err := ParseDate(year, val)
		if err != nil {
			return fmt.Errorf("invalid date: %v", err)
		}
		*d = date
		return nil
	}
	var month Month
	err := month.Parse(val)
	if err != nil {
		return fmt.Errorf("invalid month %q, expected %s", val, expectedDateFormats)
	}
	*d = NewDate(month, 0)
	return nil
}

// DayOfYear returns the day of the year for the given year as
// 1-365 for non-leap years and 1-366 for leap years.
// It will silently treat days that exceed those for a given month to the last
// day of that month. A day of zero can be used to refer to the last
// day of the previous month.
func (d Date) DayOfYear(year int) int {
	return calcDayOfYear(uint8(d>>8&0xff), uint8(d&0xff), IsLeap(year))
}

func (d Date) calcDayOfYear(leap bool) int {
	return calcDayOfYear(uint8(d>>8&0xff), uint8(d&0xff), leap)
}

func calcDayOfYear(month, day uint8, leap bool) int {
	month--
	if leap {
		if day > daysInMonthLeap[month] {
			day = daysInMonthLeap[month]
		}
		return dayOfYearLeap[month] + int(day)
	}
	if day > daysInMonth[month] {
		day = daysInMonth[month]
	}
	return dayOfYear[month] + int(day)
}

func (d Date) YearDay(year int) YearDay {
	return NewYearDay(year, d.DayOfYear(year))
}

// Tomorrow returns the date of the next day.
// It will silently treat days that exceed those for a given month as the last
// day of that month. 12/31 wraps to 1/1.
func (d Date) Tomorrow(year int) Date {
	return d.tomorrow(daysInMonthForYear(year))
}

func (d Date) tomorrow(daysInMonth []uint8) Date {
	day := uint8(d & 0xff)
	month := Month(d >> 8)
	if month == 12 && day == 31 {
		return NewDate(1, 1)
	}
	if day >= daysInMonth[month-1] {
		return NewDate(month+1, 1)
	}
	return NewDate(month, int(day+1))
}

// Yesterday returns the date of the previous day.
// 1/1 wraps to 12/31.
func (d Date) Yesterday(year int) Date {
	return d.yesterday(daysInMonthForYear(year))
}

func (d Date) yesterday(daysInMonth []uint8) Date {
	day := uint8(d & 0xff)
	month := Month(d >> 8)
	if month == 1 && day == 1 {
		return NewDate(12, 31)
	}
	if day <= 1 {
		return newDate8(month-1, daysInMonth[month-2])
	}
	return newDate8(month, day-1)
}

// DateList represents a list of Dates, it can be sorted and searched the slices package.
type DateList []Date

// Parse a comma separated list of Dates.
func (dl *DateList) Parse(year int, val string) error {
	if len(val) == 0 {
		return nil
	}
	parts := strings.Split(val, ",")
	d := make(DateList, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var date Date
		if err := date.Parse(year, part); err != nil {
			return err
		}
		d = append(d, date)
	}
	*dl = d
	return nil
}

func (dl DateList) String() string {
	var out strings.Builder
	for i, d := range dl {
		if i > 0 && i < len(dl)-1 {
			out.WriteString(", ")
		}
		out.WriteString(fmt.Sprintf("%02d-%02d", d.Month(), d.Day()))
	}
	return out.String()
}

// MonthList represents a list of Months, it can be sorted and searched the slices package.
type MonthList []Month

// Parse val in formats 'Jan,12,Nov'. The parsed list is sorted
// and without duplicates.
func (ml *MonthList) Parse(val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value")
	}
	parts := strings.Split(strings.ReplaceAll(val, " ", ""), ",")
	drs := make([]Month, 0, len(parts))
	seen := map[Month]struct{}{}
	for _, p := range parts {
		var m Month
		if err := m.Parse(p); err != nil {
			return fmt.Errorf("invalid month: %s", p)
		}
		if _, ok := seen[m]; ok {
			continue
		}
		drs = append(drs, m)
		seen[m] = struct{}{}
	}
	slices.Sort[MonthList](drs)
	*ml = drs
	return nil
}
