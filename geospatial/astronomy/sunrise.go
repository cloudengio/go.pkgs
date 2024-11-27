// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy

import (
	"time"

	"cloudeng.io/datetime"
	"github.com/nathan-osman/go-sunrise"
)

// SunRiseAndSet returns the time of sunrise and sunset for the specified
// date, latitude and longitude. The returned time is in UTC.
func SunRiseAndSet(date datetime.CalendarDate, lat, long float64) (rise, set time.Time) {
	rise, set = sunrise.SunriseSunset(
		lat, long,
		date.Year(), time.Month(date.Month()), date.Day())
	return
}

// SunRise implements datetime.DynamicTimeOfDay for sunrise.
type SunRise struct{}

func (s SunRise) Name() string {
	return "Sunrise"
}

func (s SunRise) Evaluate(year int, month datetime.Month, day int) time.Time {
	rise, _ := SunRiseAndSet(datetime.NewCalendarDate(year, month, day), 0, 0)
	return rise
}

// SunSet implements datetime.DynamicTimeOfDay for sunset.
type SunSet struct{}

func (s SunSet) Name() string {
	return "Sunset"
}

func (s SunSet) Evaluate(year int, month datetime.Month, day int) time.Time {
	_, set := SunRiseAndSet(datetime.NewCalendarDate(year, month, day), 0, 0)
	return set
}
