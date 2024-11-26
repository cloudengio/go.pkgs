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
	return datetime.NewCalendarDate(int(y), datetime.Month(m), int(d))
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
