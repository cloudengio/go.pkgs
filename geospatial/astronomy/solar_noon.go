// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy

import (
	"time"

	"cloudeng.io/datetime"
	"github.com/nathan-osman/go-sunrise"
)

func ApparentSolarNoon(date datetime.CalendarDate, place datetime.Place) time.Time {
	rise, set := sunrise.SunriseSunset(
		place.Latitude, place.Longitude, date.Year(), time.Month(date.Month()), date.Day())
	return rise.Add(set.Sub(rise) / 2).In(place.TimeLocation)
}

// SolarNoon implements datetime.DynamicTimeOfDay for the solar noon (aka Zenith).
type SolarNoon struct{}

func (s SolarNoon) Name() string {
	return "SolarNoon"
}

func (s SolarNoon) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay {
	return datetime.TimeOfDayFromTime(ApparentSolarNoon(cd, place))
}
