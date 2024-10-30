// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package dates provides support for working with dates and date ranges.
package dates

// TODO: multiyear date ranges.

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
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

func (d Date) String() string {
	return fmt.Sprintf("%s/%02d", time.Month(d.Month), d.Day)
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
		var date Date
		if err := date.Parse(year, part); err != nil {
			return err
		}
		d = append(d, date)
	}
	*dl = d
	return nil
}

type MonthList []Month

type DateRangeList []DateRange

// MergeMonthsAndRanges creates an ordered list of DateRange values from the given
// MonthList and DateRangeList.
func MergeMonthsAndRanges(year int, months MonthList, ranges DateRangeList) DateRangeList {
	drl := make(DateRangeList, 0, len(months)+len(ranges))
	for _, m := range months {
		drl = append(drl, NewDateRange(year, Date{Month: m, Day: 1}, Date{Month: m, Day: DaysInMonth(year, m)}))
	}
	drl = append(drl, ranges...)
	sort.Slice(drl, func(i, j int) bool { return drl[i].Before(drl[j]) })
	return drl
}

// Parse ranges in formats '01:03', 'Jan:Mar', '01-02:03-04' or 'Jan-02:Mar-04'.
// The parsed list is sorted and without duplicates. If the start date is
// identical then the end date is used to determine the order.
func (dr *DateRangeList) Parse(year int, ranges []string) error {
	if len(ranges) == 0 {
		return nil
	}
	drs := make(DateRangeList, 0, len(ranges))
	seen := map[DateRange]struct{}{}
	for _, rg := range ranges {
		var dr DateRange
		if err := dr.Parse(year, rg); err != nil {
			return err
		}
		if _, ok := seen[dr]; ok {
			continue
		}
		drs = append(drs, dr)
		seen[dr] = struct{}{}
	}
	//sort.Slice(drs, func(i, j int) bool { return drs[i].Before(drs[j]) })
	drs.Sort()

	*dr = drs
	return nil
}

func (dr DateRangeList) Sort() {
	sort.Slice(dr, func(i, j int) bool { return dr[i].Before(dr[j]) })
}

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

//	d := time.Date(year, time.Month(dr.From.Month), dr.From.Day, 0, 0, 0, 0, time.UTC)

//	fromOk, toOk := dc.Eval(fromTime), dc.Eval(toTime)
//		if fromOk && toOk {
//			allowed = append(allowed, dr)
//			continue
//		}
//	}
//	return allowed
//}

/*
func (dc DateConstraints) FirstLastDay(yp YearAndPlace, month Month) (first, last int) {
	daysInMonth := DaysInMonth(yp.Year, month)
	for i := 1; i <= daysInMonth; i++ {
		today := time.Date(yp.Year, time.Month(month), i, 0, 0, 0, 0, yp.Place)
		if first == 0 && dc.Eval(today) {
			first = i
		}
		if dc.Eval(today) {
			last = i
		}
	}
	return
}
*/
