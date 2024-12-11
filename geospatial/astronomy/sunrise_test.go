// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy_test

import (
	"testing"
	"time"

	"cloudeng.io/datetime"
	"cloudeng.io/geospatial/astronomy"
)

func TestSunrise(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	place := datetime.Place{
		TZ:        loc,
		Latitude:  37.3229978,
		Longitude: -122.0321823}
	cd := datetime.NewCalendarDate(2024, 1, 1)
	rise, set := astronomy.SunRiseAndSet(cd, place)

	// UTC time of sunrise and sunset.
	if got, want := rise, cd.Time(datetime.NewTimeOfDay(7, 22, 13), place.TZ); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := set, cd.Time(datetime.NewTimeOfDay(17, 00, 33), place.TZ); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
