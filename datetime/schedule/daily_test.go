// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"cloudeng.io/datetime"
	"cloudeng.io/datetime/schedule"
)

func TestDaily(t *testing.T) {
	nt := datetime.NewTimeOfDay
	action1 := newSpec("action1", 10, 2, 3, testAction{"action1"})
	action2 := newSpec("action2", 4, 5, 6, testAction{"action2"})

	cd := datetime.NewCalendarDate(2024, 1, 1)

	scheduled := schedule.Scheduled[testAction]{
		Date:  cd,
		Specs: schedule.ActionSpecs[testAction]{action1, action2},
	}

	active := []schedule.Active[testAction]{}
	for a := range scheduled.Active(datetime.Place{TimeLocation: time.UTC}) {
		active = append(active, a)
	}
	if got, want := active, ([]schedule.Active[testAction]{
		{Name: "action2", When: cd.Time(nt(4, 5, 6), time.UTC), T: action2.T},
		{Name: "action1", When: cd.Time(nt(10, 2, 3), time.UTC), T: action1.T}}); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func expectedRepeats(start time.Time, repeat time.Duration) int {
	avail := (time.Hour * 24) -
		time.Duration(start.Hour())*time.Hour -
		time.Duration(start.Minute())*time.Minute -
		time.Duration(start.Second())*time.Second
	expected := avail / repeat
	if (expected * repeat) < avail {
		expected++
	}
	return int(expected)
}

func findDelta(start time.Time, repeat time.Duration) int {
	fnd := 0
	n := start
	for {
		fnd++
		n = n.Add(repeat)
		if start.Day() != n.Day() {
			break
		}
	}
	return fnd
}

func TestDailyRepeat(t *testing.T) {
	nt := datetime.NewTimeOfDay
	ncd := datetime.NewCalendarDate

	// Note that time.Date does not guarantee which timezone it
	// returns for times in the 1AM..2AM window across locations.
	// For example 3/10 1AM in LA is reported as DST, whereas
	// as in London it is GMT. The docs for time.Date make
	// this inconistency clear. However, repeats that start
	// before the transition will gain/lose events. In all
	// cases the interval between events is maintained.

	ca, _ := time.LoadLocation("America/Los_Angeles")
	uk, _ := time.LoadLocation("Europe/London")
	for _, tc := range []struct {
		loc  *time.Location
		date datetime.CalendarDate
		dst  bool
	}{
		{ca, ncd(2024, 1, 1), false},
		{ca, ncd(2024, 3, 10), true}, // Lose an hour.
		{ca, ncd(2024, 3, 11), false},
		{ca, ncd(2024, 11, 03), true}, // Repeat an hour.
		{ca, ncd(2024, 11, 04), false},

		{uk, ncd(2024, 1, 1), false},
		{uk, ncd(2024, 3, 31), true}, // Lose an hour.
		{uk, ncd(2024, 4, 1), false},
		{uk, ncd(2024, 10, 27), true}, // Repeat an hour.
		{uk, ncd(2024, 10, 28), false},
	} {
		for _, start := range []struct {
			tod         datetime.TimeOfDay
			expectDelta bool
		}{
			{nt(0, 0, 0), true},
			{nt(0, 59, 59), true},
			{nt(1, 0, 0), false},   // can't reliably determine if the time is repeated.
			{nt(2, 0, 0), false},   // can't reliably determine if the time is repeated.
			{nt(23, 13, 0), false}, // doesn't cross a transition.
		} {

			expectedNumOps := map[string]int{}
			repeats := []time.Duration{time.Hour,
				time.Hour * 5,
				time.Minute,
				time.Minute * 13,
			}
			// Run the repeats one at a time.
			for i, repeat := range repeats {
				scheduled := createActions(tc.date, start.tod, []time.Duration{repeat})
				active, _, _ := runDaily(scheduled, tc.loc)

				expectDelta := start.expectDelta && tc.dst && repeat < time.Hour
				expected := expectedRepeats(ncd(2024, 1, 1).Time(start.tod, tc.loc), repeat)
				actual := findDelta(tc.date.Time(start.tod, tc.loc), repeat)
				if (expected-actual) == 0 && expectDelta {
					t.Errorf("expected delta for %v %v %v %v", tc.loc, tc.date, start.tod, repeat)
					continue
				}
				if got, want := len(active), actual; got != want {
					t.Errorf("%v: %v: %v: %v: got %v, want %v", tc.loc, tc.date, start, repeat, got, want)
				}

				if got, ok := compareIntervals(active, repeat); !ok {
					t.Errorf("%v: %v: %v: %v: got %v, want %v", tc.loc, tc.date, start.tod, repeat, got, repeat)
				}

				expectedNumOps[fmt.Sprintf("action%d", i)] = actual
			}

			// Run the repeats together.
			scheduled := createActions(tc.date, start.tod, repeats)
			_, numOps, times := runDaily(scheduled, tc.loc)

			for i := 0; i < len(repeats); i++ {
				name := fmt.Sprintf("action%d", i)
				prev := times[name][0]
				for _, now := range times[name][1:] {
					if got, want := now.Sub(prev), repeats[i]; got != want {
						t.Errorf("got %v, want %v", got, want)
					}
					prev = now
				}
			}

			if got, want := numOps, expectedNumOps; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}

func createActions(date datetime.CalendarDate, tod datetime.TimeOfDay, repeats []time.Duration) schedule.Scheduled[testAction] {
	scheduled := schedule.Scheduled[testAction]{
		Date: date,
	}
	for i, repeat := range repeats {
		name := fmt.Sprintf("action%d", i)
		action := newSpec(name, tod.Hour(), tod.Minute(), tod.Second(), testAction{name})
		action.Repeat.Interval = repeat
		scheduled.Specs = append(scheduled.Specs, action)
	}
	return scheduled
}

func runDaily(scheduled schedule.Scheduled[testAction], loc *time.Location) ([]schedule.Active[testAction], map[string]int, map[string][]time.Time) {
	active := []schedule.Active[testAction]{}
	numOps := map[string]int{}
	times := map[string][]time.Time{}
	for a := range scheduled.Active(datetime.Place{TimeLocation: loc}) {
		active = append(active, a)
		times[a.Name] = append(times[a.Name], a.When)
		numOps[a.Name]++
	}
	return active, numOps, times
}

func compareIntervals(active []schedule.Active[testAction], repeat time.Duration) (time.Duration, bool) {
	prev := active[0].When
	for _, a := range active[1:] {
		if got, want := a.When.Sub(prev), repeat; got != want {
			return got, false
		}
		prev = a.When
	}
	return 0, true
}
