// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"context"
	"fmt"
	"time"
)

var (
	dayOfYear       []int   // per month cumulative days in year so [0, 31, 28 etc]
	dayOfYearLeap   []int   // per month cumulative days in leap year [0, 31, 29 etc]
	daysInMonth     []uint8 // days in each month
	daysInMonthLeap []uint8
	months          = []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}

	mirrorMonths = []uint8{
		10, // jan, nov
		9,  // feb, oct
		8,  // mar, sep
		7,  // apr, aug
		6,  // may, jul
		5,  // jun
		4,  // jul, may
		3,  // aug, apr
		2,  // sep, mar
		1,  // oct, feb
		0,  // nov, jan
		11, // dec
	}
)

// MirrorMonth returns the month that is equidistant from the summer or winter
// solstice for the specified month. For example, the mirror month for January
// is November, and the mirror month for February is October.
func MirrorMonth(month Month) Month {
	return Month(mirrorMonths[month-1] + 1)
}

func daysInMonthForYearInit(year int, month uint8) uint8 {
	switch month {
	case 2:
		return DaysInFeb(year)
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

func init() {
	daysInMonth = make([]uint8, 12)
	daysInMonthLeap = make([]uint8, 12)
	dayOfYear = make([]int, 12)
	dayOfYearLeap = make([]int, 12)

	for i := uint8(0); i < 12; i++ {
		daysInMonth[i] = daysInMonthForYearInit(2023, i+1)
		daysInMonthLeap[i] = daysInMonthForYearInit(2024, i+1)
	}
	for i := uint8(0); i < 11; i++ {
		dayOfYear[i+1] += dayOfYear[i] + int(daysInMonth[i])
		dayOfYearLeap[i+1] += dayOfYearLeap[i] + int(daysInMonthLeap[i])
	}
}

// DaysInMonth returns the number of days in the given month for the given year.
func DaysInMonth(year int, month Month) uint8 {
	if IsLeap(year) {
		return daysInMonthLeap[month-1]
	}
	return daysInMonth[month-1]
}

func daysInMonthForYear(year int) []uint8 {
	if IsLeap(year) {
		return daysInMonthLeap
	}
	return daysInMonth
}

// IsLeap returns true if the given year is a leap year.
func IsLeap(year int) bool {
	return year%4 == 0 && year%100 != 0 || year%400 == 0
}

// DaysInFeb returns the number of days in February for the given year.
func DaysInFeb(year int) uint8 {
	if IsLeap(year) {
		return 29
	}
	return 28
}

// DaysInYear returns the number of days in the given year.dc
func DaysInYear(year int) int {
	if IsLeap(year) {
		return 366
	}
	return 365
}

// YearAndPlace represents a year and a location.
type YearAndPlace struct {
	Year  int
	Place *time.Location
}

// IsSet returns true if the Year and Place fields are both set.
func (yp YearAndPlace) IsSet() bool {
	return yp.Year != 0 && yp.Place != nil
}

// YearAndPlaceFromTime returns a YearAndPlace value from the given time.
func YearAndPlaceFromTime(t time.Time) YearAndPlace {
	return YearAndPlace{
		Year:  t.Year(),
		Place: t.Location(),
	}
}

// NewYearAndPlace returns a YearAndPlace value from the given year and location.
func NewYearAndPlace(year int, place *time.Location) YearAndPlace {
	return YearAndPlace{
		Year:  year,
		Place: place,
	}
}

type ypKey struct{}

// ContextWithYearAndPlace returns a new context with the given YearAndPlace value
// stored in it.
func ContextWithYearAndPlace(ctx context.Context, yp YearAndPlace) context.Context {
	return context.WithValue(ctx, ypKey{}, yp)
}

// YearAndPlaceFromContext returns the YearAndPlace value stored in the given context,
// if there is no value stored then an empty YearAndPlace is returned for
// which is IsNotset will be true.
func YearAndPlaceFromContext(ctx context.Context) YearAndPlace {
	yp, ok := ctx.Value(ypKey{}).(YearAndPlace)
	if !ok {
		return YearAndPlace{}
	}
	return yp
}

// YearDay represents a year and the day in that year as 1-365/366.
type YearDay uint32

func (yd YearDay) Year() int {
	return int(yd >> 16)
}

func (yd YearDay) Day() int {
	return int(yd & 0xffff)
}

func (yd YearDay) String() string {
	return fmt.Sprintf("%04d(%03d)", yd.Year(), yd.Day())
}

func dateFromDay(day int, daysInMonth []uint8) Date {
	for month := uint8(0); month < 12; month++ {
		dm := int(daysInMonth[month])
		if day <= dm {
			return NewDate(Month(month+1), day)
		}
		day -= dm
	}
	panic("unreachable")
}

// Date returns the Date for the given day of the year. A day of
// <= 0 is treated as Jan-01 and a day of > 365/366 is treated as Dec-31.
func (yd YearDay) Date() Date {
	day := yd.Day()
	if day <= 0 {
		return NewDate(1, 1)
	}
	year := yd.Year()
	if IsLeap(year) {
		if day > 366 {
			return NewDate(12, 31)
		}
		return dateFromDay(day, daysInMonthLeap)
	}
	if day > 365 {
		return NewDate(12, 31)
	}
	return dateFromDay(day, daysInMonth)
}

func (yd YearDay) CalendarDate() CalendarDate {
	date := yd.Date()
	return NewCalendarDate(yd.Year(), date.Month(), date.Day())
}

// NewYearDay returns a YearDay for the given year and day. If the day is greater
// than the number of days in the year then the last day of the year is used.
func NewYearDay(year, day int) YearDay {
	n := DaysInYear(year)
	if day > n {
		day = n
	}
	return YearDay(uint32(year)<<16 | uint32(day))
}
