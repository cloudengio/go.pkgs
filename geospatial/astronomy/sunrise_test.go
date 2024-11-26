// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy

import (
	"testing"
	"time"

	"cloudeng.io/datetime"
)

func TestSunrise(t *testing.T) {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	lat, long := 37.3229978, -122.0321823
	rise, set := SunRise(datetime.NewCalendarDate(2024, 1, 1), lat, long)

	if got, want := rise.In(loc), time.Date(2024, 1, 1, 7, 22, 13, 0, loc); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := set.In(loc), time.Date(2024, 1, 1, 17, 00, 33, 0, loc); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
