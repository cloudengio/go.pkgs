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
// date, latitude and longitude. The returned times are in UTC and must
// adjusted for the local timezone that lat/long correspond to.
func SunRiseAndSet(date datetime.CalendarDate, place datetime.Place) (time.Time, time.Time) {
	rise, set := sunrise.SunriseSunset(
		place.Latitude, place.Longitude, date.Year(), time.Month(date.Month()), date.Day())
	return rise.In(place.Location), set.In(place.Location)
}

// SunRise implements datetime.DynamicTimeOfDay for sunrise.
type SunRise struct{}

func (s SunRise) Name() string {
	return "Sunrise"
}

func (s SunRise) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay {
	rise, _ := SunRiseAndSet(cd, place)
	return datetime.TimeOfDayFromTime(rise)
}

// SunSet implements datetime.DynamicTimeOfDay for sunset.
type SunSet struct{}

func (s SunSet) Name() string {
	return "Sunset"
}

func (s SunSet) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay {
	_, set := SunRiseAndSet(cd, place)
	return datetime.TimeOfDayFromTime(set)
}
