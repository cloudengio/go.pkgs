// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package astronomy_test

import (
	"testing"

	"cloudeng.io/datetime"
	"cloudeng.io/geospatial/astronomy"
)

func TestSolstice(t *testing.T) {

	if got, want := astronomy.December(2024), datetime.NewCalendarDate(2024, 12, 21); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := astronomy.March(1900), datetime.NewCalendarDate(1900, 03, 21); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := astronomy.June(2022), datetime.NewCalendarDate(2022, 06, 21); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := astronomy.September(2023), datetime.NewCalendarDate(2023, 9, 23); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
