// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"context"
	"time"
)

var (
	dayOfYear       []int // per month cumulative days in year so [0, 31, 28 etc]
	dayOfYearLeap   []int // per month cumulative days in leap year [0, 31, 29 etc]
	daysInMonth     []int // days in each month
	daysInMonthLeap []int
	months          = []string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}

	mirrorMonths = []int{
		11 - 1, // jan, nov
		10 - 1, // feb, oct
		9 - 1,  // mar, sep
		8 - 1,  // apr, aug
		7 - 1,  // may, jul
		6 - 1,  // jun
		5 - 1,  // jul, may
		4 - 1,  // aug, apr
		3 - 1,  // sep, mar
		2 - 1,  // oct, feb
		1 - 1,  // nov, jan
		12 - 1, // dec
	}
)

// MirrorMonth returns the month that is equidistant from the summer or winter
// solstice for the specified month. For example, the mirror month for January
// is November, and the mirror month for February is October.
func MirrorMonth(month Month) Month {
	return Month(mirrorMonths[month-1] + 1)
}

func daysInMonthForYearInit(year int, month int) int {
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
	daysInMonth = make([]int, 12)
	daysInMonthLeap = make([]int, 12)
	dayOfYear = make([]int, 12)
	dayOfYearLeap = make([]int, 12)

	for i := 0; i < 12; i++ {
		daysInMonth[i] = daysInMonthForYearInit(2023, i+1)
		daysInMonthLeap[i] = daysInMonthForYearInit(2024, i+1)
	}
	for i := 0; i < 11; i++ {
		dayOfYear[i+1] += dayOfYear[i] + daysInMonth[i]
		dayOfYearLeap[i+1] += dayOfYearLeap[i] + daysInMonthLeap[i]
	}
}

// DaysInMonth returns the number of days in the given month for the given year.
func DaysInMonth(year int, month Month) int {
	if IsLeap(year) {
		return daysInMonthLeap[month-1]
	}
	return daysInMonth[month-1]
}

func daysInMonthForYear(year int) []int {
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
func DaysInFeb(year int) int {
	if IsLeap(year) {
		return 29
	}
	return 28
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
