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

func newAction[T any](name string, hour, min, sec int, action T) schedule.Action[T] {
	return schedule.Action[T]{
		Name:   name,
		Due:    datetime.NewTimeOfDay(hour, min, sec),
		Action: action,
	}
}

func expectedActiveForMonth[T any](year, mnth int, actions ...schedule.Action[T]) []schedule.Active[T] {
	days := datetime.DaysInMonth(year, datetime.Month(mnth))
	r := make([]schedule.Active[T], days)
	for i := range days {
		r[i] = schedule.Active[T]{
			Date:    datetime.NewDate(datetime.Month(mnth), int(i+1)),
			Actions: actions,
		}
	}
	return r
}

func sortActive[T any](active []schedule.Active[T]) {
	sort.Slice(active, func(i, j int) bool {
		if active[i].Date != active[j].Date {
			return active[i].Date < active[j].Date
		}
		// assume # actions is the same.
		for n := range active[i].Actions {
			if active[i].Actions[n].Name != active[j].Actions[n].Name {
				return active[i].Actions[n].Name < active[j].Actions[n].Name
			}
		}
		return true
	})
}

func TestScheduler(t *testing.T) {
	action1 := newAction("action1", 1, 2, 3, testAction{"action1"})
	action2 := newAction("action2", 4, 5, 6, testAction{"action2"})
	sched := schedule.Annual[testAction]{
		Name: "test",
		Dates: schedule.Dates{
			For: datetime.MonthList{1, 2},
		},
		Actions: []schedule.Action[testAction]{action1, action2},
	}
	scheduler := schedule.NewAnnualScheduler(sched)
	yp := datetime.YearAndPlace{Year: 2024, Place: time.UTC}
	active := []schedule.Active[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected := expectedActiveForMonth(yp.Year, 1, action1)
	expected = append(expected, expectedActiveForMonth(yp.Year, 1, action2)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action2)...)
	sortActive(expected)

	if got, want := len(active), (31+29)*2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	action3 := newAction("action3", 1, 2, 3, testAction{"action3"})
	sched.Actions = append(sched.Actions, action3)
	scheduler = schedule.NewAnnualScheduler(sched)

	active = []schedule.Active[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected = expectedActiveForMonth(yp.Year, 1, action1, action3)
	expected = append(expected, expectedActiveForMonth(yp.Year, 1, action2)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1, action3)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action2)...)
	sortActive(expected)

	if got, want := len(active), (31+29)*2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", len(got), len(want))
		t.Errorf("got %v, want %v", got, want)
	}

	yp = datetime.YearAndPlace{Year: 2023, Place: time.UTC}
	active = []schedule.Active[testAction]{}
	for scheduled := range scheduler.Scheduled(yp) {
		active = append(active, scheduled)
	}

	expected = expectedActiveForMonth(yp.Year, 1, action1, action3)
	expected = append(expected, expectedActiveForMonth(yp.Year, 1, action2)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action1, action3)...)
	expected = append(expected, expectedActiveForMonth(yp.Year, 2, action2)...)
	sortActive(expected)

	if got, want := len(active), (31+28)*2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := active, expected; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", len(got), len(want))
		t.Errorf("got %v, want %v", got, want)
	}
}
