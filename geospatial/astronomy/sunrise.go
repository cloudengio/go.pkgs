// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy

import (
	"time"

	"cloudeng.io/datetime"
	"github.com/nathan-osman/go-sunrise"
)

// SunRise returns the time of sunrise and sunset for the specified
// date, latitude and longitude. The returned time is in UTC.
func SunRise(date datetime.CalendarDate, lat, long float64) (rise, set time.Time) {
	rise, set = sunrise.SunriseSunset(
		lat, long,
		date.Year(), time.Month(date.Month()), date.Day())
	return
}
