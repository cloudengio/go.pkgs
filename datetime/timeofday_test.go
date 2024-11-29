// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cloudeng.io/datetime"
)

func TestTimeOfDayParse(t *testing.T) {
	for _, tc := range []struct {
		val  string
		when datetime.TimeOfDay
	}{
		{"08:12", datetime.NewTimeOfDay(8, 12, 0)},
		{"08-12", datetime.NewTimeOfDay(8, 12, 0)},
		{"20:01", datetime.NewTimeOfDay(20, 01, 0)},
		{"21-01", datetime.NewTimeOfDay(21, 01, 0)},
		{"08:12:13", datetime.NewTimeOfDay(8, 12, 13)},
		{"08-12-13", datetime.NewTimeOfDay(8, 12, 13)},
		{"20:01:13", datetime.NewTimeOfDay(20, 01, 13)},
		{"21-01-13", datetime.NewTimeOfDay(21, 01, 13)},
	} {
		var tod datetime.TimeOfDay
		if err := tod.Parse(tc.val); err != nil {
			t.Errorf("failed: %v: %v", tc.val, err)
		}
		if !reflect.DeepEqual(tod, tc.when) {
			t.Errorf("got %v, want %v", tod, tc.when)
		}
	}

	for _, tc := range []string{
		"",
		"08:61",
		"08 16",
		"08:61-15",
		"08-61:15",
	} {
		var tod datetime.TimeOfDay
		if err := tod.Parse(tc); err == nil {
			t.Errorf("failed to return an error: %v", tc)
		}
	}

	tods := datetime.TimeOfDayList{}
	examples := []string{"08:13", "07:13", "09:14:12", "09:14:9", "09:14"}
	for _, s := range examples {
		var tod datetime.TimeOfDay
		if err := tod.Parse(s); err != nil {
			t.Errorf("failed: %v", err)
		}
		tods = append(tods, tod)
	}
	slices.Sort(tods)

	nt := datetime.NewTimeOfDay
	expected := newTimeOfDayList(
		nt(7, 13, 0),
		nt(8, 13, 0),
		nt(9, 14, 0),
		nt(9, 14, 9),
		nt(9, 14, 12))

	if got, want := tods, expected; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	tods = datetime.TimeOfDayList{}
	if err := tods.Parse(strings.Join(examples, ",")); err != nil {
		t.Errorf("failed: %v", err)
	}

	if got, want := tods, expected; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTimeOfDayAdd(t *testing.T) {
	nt := datetime.NewTimeOfDay
	tod := nt(8, 1, 2)
	if got, want := tod.Add(time.Hour), nt(9, 1, 2); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tod.Add(time.Hour*11), nt(19, 1, 2); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tod.Add(time.Hour*23), nt(23, 59, 59); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tod.Add(-time.Hour), nt(7, 1, 2); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tod.Add(-time.Hour*24), nt(0, 0, 0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
