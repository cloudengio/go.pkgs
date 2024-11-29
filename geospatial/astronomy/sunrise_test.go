// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy_test

import (
	"testing"

	"cloudeng.io/datetime"
	"cloudeng.io/geospatial/astronomy"
)

func TestSunrise(t *testing.T) {
	//loc, err := time.LoadLocation("America/Los_Angeles")
	lat, long := 37.3229978, -122.0321823
	rise, set := astronomy.SunRiseAndSet(datetime.NewCalendarDate(2024, 1, 1), lat, long)

	// UTC time of sunrise and sunset.
	if got, want := rise, datetime.NewTimeOfDay(7, 22, 13); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := set, datetime.NewTimeOfDay(17, 00, 33); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
