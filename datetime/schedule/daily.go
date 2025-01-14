// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule

import (
	"iter"
	"slices"
	"sort"
	"time"

	"cloudeng.io/datetime"
)

// DynamicTimeOfDaySpec represents a time of day that is dynamically evaluated
// and offset with a fixed duration.
type DynamicTimeOfDaySpec struct {
	Due    datetime.DynamicTimeOfDay
	Offset time.Duration
}

// RepeatSpec represents a repeat interval and an optional number of times
// for an action to be repeated.
type RepeatSpec struct {
	Interval time.Duration
	Repeats  int
}

// ActionSpec represents a specification of an action to be taken at specific
// times of that day. The specification may refer to a dynamically
// evaluated time of day (eg. sunrise/sunset) with an offset.
// In addition it may specify a repeat interval and an optional
// number of times for the action to be repeated.
type ActionSpec[T any] struct {
	Name    string
	Due     datetime.TimeOfDay
	Dynamic DynamicTimeOfDaySpec
	Repeat  RepeatSpec
	T       T
}

type ActionSpecs[T any] []ActionSpec[T]

// Evaluate returns a new ActionSpec with the Due field set to the result of
// evaluating the DynamicDue field if it is non-nil (with the DynamicOffset applied).
func (a ActionSpec[T]) Evaluate(cd datetime.CalendarDate, place datetime.Place) ActionSpec[T] {
	r := a
	if dyn := a.Dynamic.Due; dyn != nil {
		r.Due = dyn.Evaluate(cd, place).Add(a.Dynamic.Offset)
		r.Dynamic = DynamicTimeOfDaySpec{}
	}
	return r
}

// Evaluate returns a new ActionSpecs with each of the ActionSpecs evaluated,
// that is, with any Dynamic functions evaluated and the results stored in
// the Due field. The returned ActionSpecs have their Dynamic fields zeroed out.
func (a ActionSpecs[T]) Evaluate(cd datetime.CalendarDate, place datetime.Place) ActionSpecs[T] {
	result := make(ActionSpecs[T], len(a))
	for i, as := range a {
		result[i] = as.Evaluate(cd, place)
	}
	return result
}

// Sort by due time and then by name.
func (a ActionSpecs[T]) Sort() {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Due == a[j].Due {
			return a[i].Name < a[j].Name
		}
		return a[i].Due < a[j].Due
	})
}

// Sort by due time, but preserve the order of actions with
// the same due time.
func (a ActionSpecs[T]) SortStable() {
	slices.SortStableFunc(a, func(a, b ActionSpec[T]) int {
		if a.Due < b.Due {
			return -1
		} else if a.Due > b.Due {
			return 1
		}
		return 0
	})
}

// Active represents the next scheduled action, ie. the one to be 'active'
// at time 'When'.
type Active[T any] struct {
	Name string
	When time.Time
	T    T
}

// Scheduled specifies the set of actions scheduled for a given date.
type Scheduled[T any] struct {
	Date  datetime.CalendarDate
	Specs ActionSpecs[T]
}

// Active is an iterator that returns the next scheduled action for the
// given year and place.
func (s Scheduled[T]) Active(place datetime.Place) iter.Seq[Active[T]] {
	rm := newRepeatManager(s.Specs, s.Date, place)
	return func(yield func(Active[T]) bool) {
		for rm.hasActions() {
			when, he := rm.manage(place.TimeLocation)
			active := Active[T]{
				Name: he.name,
				When: when,
				T:    he.t,
			}
			if !yield(active) {
				return
			}
		}
	}
}
