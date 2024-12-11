// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"cloudeng.io/datetime"
	"cloudeng.io/datetime/schedule"
)

type testAction struct {
	action string
}

func newSpec[T any](name string, h, m, s int, action T) schedule.ActionSpec[T] {
	return schedule.ActionSpec[T]{
		Name: name,
		Due:  datetime.NewTimeOfDay(h, m, s),
		T:    action,
	}
}

func expectedActiveForMonth[T any](year, mnth int, specs ...schedule.ActionSpec[T]) []schedule.Scheduled[T] {
	days := datetime.DaysInMonth(year, datetime.Month(mnth))
	r := make([]schedule.Scheduled[T], days)
	for i := range days {
		r[i] = schedule.Scheduled[T]{
			Date:  datetime.NewCalendarDate(year, datetime.Month(mnth), int(i+1)),
			Specs: specs,
		}
	}
	return r
}

func sortForDate[T any](active []schedule.Scheduled[T]) {
	sort.Slice(active, func(i, j int) bool {
		if active[i].Date != active[j].Date {
			return active[i].Date < active[j].Date
		}
		// assume # actions is the same.
		for n := range active[i].Specs {
			if active[i].Specs[n].Name != active[j].Specs[n].Name {
				return active[i].Specs[n].Name < active[j].Specs[n].Name
			}
		}
		return true
	})
}

func TestScheduler(t *testing.T) {
	action1 := newSpec("action1", 1, 2, 3, testAction{"action1"})
	action2 := newSpec("action2", 4, 5, 6, testAction{"action2"})
	sched := schedule.Annual[testAction]{
		Name: "test",
		Dates: schedule.Dates{
			For: datetime.MonthList{1, 2},
		},
		Specs: []schedule.ActionSpec[testAction]{action1, action2},
	}
	scheduler := schedule.NewAnnualScheduler(sched)
	yp := datetime.NewYearTZ(2024, time.UTC)
	active := []schedule.Scheduled[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected := expectedActiveForMonth(yp.Year, 1, action1, action2)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1, action2)...)
	sortForDate(expected)

	if got, want := len(active), (31 + 29); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Add a third action, test that sorting by due time works.
	action3 := newSpec("action3", 1, 2, 3, testAction{"action3"})
	sched.Specs = append(sched.Specs, action3)
	scheduler = schedule.NewAnnualScheduler(sched)

	active = []schedule.Scheduled[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected = expectedActiveForMonth(yp.Year, 1, action1, action3, action2)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1, action3, action2)...)
	sortForDate(expected)

	if got, want := len(active), (31 + 29); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Logf(" got: %v", got[0])
		t.Logf("want: %v", want[0])

		t.Errorf("got %v, want %v", got, want)
	}

	// Check non-leap year.
	yp = datetime.NewYearTZ(2023, time.UTC)
	active = []schedule.Scheduled[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected = expectedActiveForMonth(yp.Year, 1, action1, action3, action2)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1, action3, action2)...)
	sortForDate(expected)

	if got, want := len(active), (31 + 28); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSchedulerDifferentYear(t *testing.T) {
	cd := datetime.NewCalendarDate
	action1 := newSpec("action1", 1, 2, 3, testAction{"action1"})
	schedMonth := schedule.Annual[testAction]{
		Name: "testMonth",
		Dates: schedule.Dates{
			For: datetime.MonthList{2},
		},
		Specs: []schedule.ActionSpec[testAction]{action1},
	}
	schedRange := schedule.Annual[testAction]{
		Name: "testRange",
		Dates: schedule.Dates{
			Ranges: datetime.DateRangeList{
				datetime.NewDateRange(datetime.NewDate(2, 1), datetime.NewDate(2, 29)),
			},
		},
		Specs: []schedule.ActionSpec[testAction]{action1},
	}
	for _, sched := range []schedule.Annual[testAction]{
		schedMonth, schedRange,
	} {
		scheduler := schedule.NewAnnualScheduler(sched)
		yp := datetime.NewYearTZ(2023, time.UTC)
		active := []schedule.Scheduled[testAction]{}
		for scheduled := range scheduler.Scheduled(yp) {
			active = append(active, scheduled)
		}
		if got, want := len(active), 28; got != want {
			t.Errorf("got %v, want %v", got, want)
			continue
		}
		if got, want := active[27].Date, cd(yp.Year, 2, 28); got != want {
			t.Errorf("got %v, want %v", got, want)
		}

		yp.Year = 2024
		active = []schedule.Scheduled[testAction]{}
		for scheduled := range scheduler.Scheduled(yp) {
			active = append(active, scheduled)
		}
		if got, want := len(active), 29; got != want {
			t.Errorf("got %v, want %v", got, want)
			continue
		}
		if got, want := active[27].Date, cd(yp.Year, 2, 28); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := active[28].Date, cd(yp.Year, 2, 29); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
