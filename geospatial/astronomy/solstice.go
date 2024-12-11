// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy

import (
	"cloudeng.io/datetime"
	"github.com/mooncaker816/learnmeeus/v3/julian"
	"github.com/mooncaker816/learnmeeus/v3/solstice"
)

func JDEToCalendar(jde float64) datetime.CalendarDate {
	y, m, d := julian.JDToCalendar(jde)
	return datetime.NewCalendarDate(y, datetime.Month(m), int(d))
}

// December returns the winter solstice.
func December(year int) datetime.CalendarDate {
	return JDEToCalendar(solstice.December(year))
}

// March returns the vernal/spring equinox.
func March(year int) datetime.CalendarDate {
	return JDEToCalendar(solstice.March(year))
}

// June returns the summer solstice.
func June(year int) datetime.CalendarDate {
	return JDEToCalendar(solstice.June(year))
}

// September returns the autumnal equinox.
func September(year int) datetime.CalendarDate {
	return JDEToCalendar(solstice.September(year))
}

// SummerSolstice implements datetime.DynamicDateRange for the summer solstice.
type SummerSolstice struct{}

func (s SummerSolstice) Name() string {
	return "SummerSolstice"
}

func (s SummerSolstice) Evaluate(year int) datetime.CalendarDateRange {
	cd := June(year)
	return datetime.NewCalendarDateRange(cd, cd)
}

// WinterSolstice implements datetime.DynamicDateRange for the winter solstice.
type WinterSolstice struct{}

func (s WinterSolstice) Name() string {
	return "WinterSolstice"
}

func (s WinterSolstice) Evaluate(year int) datetime.CalendarDateRange {
	cd := December(year)
	return datetime.NewCalendarDateRange(cd, cd)
}

// SpringEquinox implements datetime.DynamicDateRange for the spring equinox.
type SpringEquinox struct{}

func (s SpringEquinox) Name() string {
	return "SpringEquinox"
}

func (s SpringEquinox) Evaluate(year int) datetime.CalendarDateRange {
	cd := March(year)
	return datetime.NewCalendarDateRange(cd, cd)
}

// AutumnEquinox implements datetime.DynamicDateRange for the autumn equinox.
type AutumnEquinox struct{}

func (s AutumnEquinox) Name() string {
	return "AutumnEquinox"
}

func (s AutumnEquinox) Evaluate(year int) datetime.CalendarDateRange {
	cd := September(year)
	return datetime.NewCalendarDateRange(cd, cd)
}

// Winter implements datetime.DynamicDateRange for the winter season.
type Winter struct{}

func (w Winter) Name() string {
	return "Winter"
}

func (w Winter) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(December(year), March(year+1))
}

// Spring implements datetime.DynamicDateRange for the spring season.
type Spring struct{}

func (s Spring) Name() string {
	return "Spring"
}

func (s Spring) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(March(year), June(year))
}

// Summer implements datetime.DynamicDateRange for the summer season.
type Summer struct{}

func (s Summer) Name() string {
	return "Summer"
}

func (s Summer) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(June(year), September(year))
}

// Autumn implements datetime.DynamicDateRange for the autumn season.
type Autumn struct{ LocalName string }

func (a Autumn) Name() string {
	if a.LocalName != "" {
		return a.LocalName
	}
	return "Autumn"
}

func (a Autumn) Evaluate(year int) datetime.CalendarDateRange {
	return datetime.NewCalendarDateRange(September(year), December(year))
}
