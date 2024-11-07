// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package datetime provides support for working with dates, time of day and associated ranges.
package datetime

// TODO: multiyear date ranges.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Month as an int.
type Month time.Month

// ParseNumericMonth parses a 1 or 2 digit numeric month value in the range 1-12.
func ParseNumericMonth(val string) (Month, error) {
	n, err := strconv.Atoi(val)
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
// Day may be zero in which case the Date is refers to the entire month with the
// interpretation of the day determined by the context (eg. a start date may refer
// to the first day of the month, and an end date may refer to the last day of the month).
type Date struct {
	Month Month
	Day   int
}

func NewDate(when time.Time) Date {
	return Date{Month: Month(when.Month()), Day: when.Day()}
}

func (d Date) String() string {
	return fmt.Sprintf("%s %02d", time.Month(d.Month), d.Day)
}

var numericDateRe = regexp.MustCompile(`^([0-1]+)-([0-9]{1,2})$`)

// Parse a numeric date in formats '01/02' with error checking
// for valid month and day.
func ParseNumericDate(year int, val string) (Date, error) {
	parts := strings.Split(val, "/")
	if len(parts) != 2 {
		return Date{}, fmt.Errorf("invalid value %q, expected format 'Jan-02'", val)
	}
	month, err := ParseNumericMonth(parts[0])
	if err != nil {
		return Date{}, err
	}
	day, err := strconv.Atoi(parts[1])
	if err != nil {
		return Date{}, fmt.Errorf("invalid day: %s", parts[1])
	}
	if day < 1 || day > DaysInMonth(year, month) {
		return Date{}, fmt.Errorf("invalid day for %v %v: %d", day, month, day)
	}
	return Date{Month: month, Day: day}, nil
}

// ParseDate parses a date in the forma 'Jan-02' with error checking
// for valid month and day.
func ParseDate(year int, val string) (Date, error) {
	parts := strings.Split(val, "-")
	if len(parts) != 2 {
		return Date{}, fmt.Errorf("invalid date %q, expected format 'Jan-02'", val)
	}
	month, err := ParseMonth(parts[0])
	if err != nil {
		return Date{}, err
	}
	day, err := strconv.Atoi(parts[1])
	if err != nil {
		return Date{}, fmt.Errorf("invalid day: %s", parts[1])
	}
	if day < 1 || day > DaysInMonth(year, month) {
		return Date{}, fmt.Errorf("invalid day for %v %v: %d", day, month, day)
	}
	return Date{Month: month, Day: day}, nil
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
			return fmt.Errorf("invalid numeric date %q, expected %s", val, expectedDateFormats)
		}
		*d = date
		return nil
	}
	if strings.Contains(val, "-") {
		date, err := ParseDate(year, val)
		if err != nil {
			return fmt.Errorf("invalid date %q, expected %s", val, expectedDateFormats)
		}
		*d = date
		return nil
	}
	var month Month
	err := month.Parse(val)
	if err != nil {
		return fmt.Errorf("invalid month %q, expected %s", val, expectedDateFormats)
	}
	d.Month = month
	return nil
}

// DayOfYear returns the day of the year for the given year as
// 1-365 for non-leap years and 1-366 for leap years.
// It will silently treat days that exceed those for a given month to the last
// day of that month. A day of zero can be used to refer to the last
// day of the previous month.
func (d Date) DayOfYear(year int) int {
	return d.dayOfYear(IsLeap(year))
}

func (d Date) dayOfYear(leap bool) int {
	md := d.Day
	if leap {
		if md > daysInMonthLeap[d.Month-1] {
			md = daysInMonthLeap[d.Month-1]
		}
		return dayOfYearLeap[time.Month(d.Month)-1] + md
	}
	if md > daysInMonth[d.Month-1] {
		md = daysInMonth[d.Month-1]
	}
	return dayOfYear[time.Month(d.Month)-1] + md
}

// Tomorrow returns the date of the next day.
// It will silently treat days that exceed those for a given month as the last
// day of that month.
// 12/31 wraps to 1/1.
func (d Date) Tomorrow(year int) Date {
	return d.tomorrow(daysInMonthForYear(year))
}

func (d Date) tomorrow(daysInMonth []int) Date {
	if d.Month == 12 && d.Day == 31 {
		return Date{Month(1), 1}
	}
	if d.Day >= daysInMonth[d.Month-1] {
		d.Month++
		d.Day = 1
		return d
	}
	d.Day++
	return d
}

// Yesterday returns the date of the previous day.
// 1/1 wraps to 12/31.
func (d Date) Yesterday(year int) Date {
	return d.yesterday(daysInMonthForYear(year))
}

func (d Date) yesterday(daysInMonth []int) Date {
	if d.Month == 1 && d.Day == 1 {
		return Date{Month(12), 31}
	}
	if d.Day <= 1 {
		d.Month--
		d.Day = daysInMonth[d.Month-1]
		return d
	}
	d.Day--
	return d
}

func dateFromDay(day int, daysInMonth []int) Date {
	for month := 0; month < 12; month++ {
		if day < daysInMonth[month] {
			return Date{Month(month + 1), day}
		}
		day -= daysInMonth[month]
	}
	panic("unreachable")
}

// DateFromDay returns the Date for the given day of the year. A day of
// <= 0 is treated as Jan-01 and a day of > 365/366 is treated as Dec-31.
func DateFromDay(year, day int) Date {
	if day <= 0 {
		return Date{Month(1), 1}
	}
	if IsLeap(year) {
		if day > 366 {
			return Date{Month(12), 31}
		}
		return dateFromDay(day, daysInMonthLeap)
	}
	if day > 365 {
		return Date{Month(12), 31}
	}
	return dateFromDay(day, daysInMonth)
}

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
		out.WriteString(fmt.Sprintf("%02d-%02d", d.Month, d.Day))
	}
	return out.String()
}

func (dl DateList) Contains(d Date) bool {
	for _, dd := range dl {
		if dd == d {
			return true
		}
	}
	return false
}

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
