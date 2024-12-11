// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cloudeng.io/datetime"
	"cloudeng.io/datetime/schedule"
)

func parseDateRangeList(s ...string) datetime.DateRangeList {
	if len(s) == 0 || len(s[0]) == 0 {
		return datetime.DateRangeList{}
	}
	var dr datetime.DateRangeList
	if err := dr.Parse(s); err != nil {
		panic(err)
	}
	return dr
}

func parseMontList(s string) datetime.MonthList {
	var ml datetime.MonthList
	if err := ml.Parse(s); err != nil {
		panic(err)
	}
	return ml
}

func TestDates(t *testing.T) {
	pdrl := parseDateRangeList
	pml := parseMontList
	unconstrained := datetime.Constraints{}
	noFeb29 := datetime.Constraints{Custom: []datetime.Date{datetime.NewDate(2, 29)}}
	for _, tc := range []struct {
		monthsFor   string
		mirror      bool
		ranges      string
		constraints datetime.Constraints
		year        int
		expected    string
	}{
		{"1,2", false, "", unconstrained, 2024, "1:2"},
		{"1,2", true, "", unconstrained, 2024, "1:2,10:11"},
		{"1,2", false, "", unconstrained, 2023, "1:2"},
		{"1", false, "12/01:12/06", unconstrained, 2024, "1:1,12/01:12/06"},
		{"2", false, "", noFeb29, 2024, "2/1:2/28"},
		{"2", true, "", noFeb29, 2024, "2/1:2/28,10/1:10/31"},
		{"2", false, "", noFeb29, 2023, "2/1:2/28"},
	} {
		sd := schedule.Dates{
			For:          pml(tc.monthsFor),
			Ranges:       pdrl(strings.Split(tc.ranges, ",")...),
			MirrorMonths: tc.mirror,
			Constraints:  tc.constraints,
		}
		expected := pdrl(strings.Split(tc.expected, ",")...)
		for i := range expected {
			expected[i] = expected[i].Normalize(tc.year)
		}
		if got, want := sd.EvaluateDateRanges(tc.year), expected; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}

func TestActions(t *testing.T) {
	a := schedule.ActionSpecs[int]{
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "g", T: 1},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "f", T: 2},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "e", T: 3},
		{Due: datetime.NewTimeOfDay(12, 1, 3), Name: "d", T: 4},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 5},
		{Due: datetime.NewTimeOfDay(12, 50, 2), Name: "b", T: 6},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "a", T: 7},
	}
	b := slices.Clone(a)

	a.Sort()
	if got, want := a, []schedule.ActionSpec[int]{
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "a", T: 7},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 5},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "e", T: 3},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "f", T: 2},
		{Due: datetime.NewTimeOfDay(12, 1, 3), Name: "d", T: 4},
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "g", T: 1},
		{Due: datetime.NewTimeOfDay(12, 50, 2), Name: "b", T: 6},
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	b.SortStable()
	if got, want := b, []schedule.ActionSpec[int]{
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "e", T: 3},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 5},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "a", T: 7},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "f", T: 2},
		{Due: datetime.NewTimeOfDay(12, 1, 3), Name: "d", T: 4},
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "g", T: 1},
		{Due: datetime.NewTimeOfDay(12, 50, 2), Name: "b", T: 6},
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type DynamicTimeOfDay struct {
	name string
	val  datetime.TimeOfDay
}

func (d DynamicTimeOfDay) Name() string {
	return d.name
}

func (d DynamicTimeOfDay) Evaluate(_ datetime.CalendarDate, _ datetime.Place) datetime.TimeOfDay {
	return d.val
}

func TestDynamic(t *testing.T) {
	breakfast := DynamicTimeOfDay{name: "breakfast", val: datetime.NewTimeOfDay(8, 0, 0)}
	a := schedule.ActionSpecs[int]{
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "a", T: 1},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "b", T: 2},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 3},
		{Dynamic: schedule.DynamicTimeOfDaySpec{Due: breakfast, Offset: time.Minute * 30}, Name: "d", T: 4},
	}
	b := a.Evaluate(datetime.NewCalendarDate(2024, 1, 1), datetime.Place{TZ: time.Local})
	b.Sort()

	if got, want := b, []schedule.ActionSpec[int]{
		{Due: datetime.NewTimeOfDay(8, 30, 0), Name: "d", T: 4},
		{Due: datetime.NewTimeOfDay(12, 0, 2), Name: "c", T: 3},
		{Due: datetime.NewTimeOfDay(12, 1, 1), Name: "b", T: 2},
		{Due: datetime.NewTimeOfDay(12, 3, 0), Name: "a", T: 1},
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
