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
	for a := range scheduled.Active(datetime.Place{TZ: time.UTC}) {
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

func findDelta(expected int, start time.Time, repeat time.Duration) int {
	if !(start.Month() == 3 && start.Day() == 10) && !(start.Month() == 11 && start.Day() == 3) {
		return 0
	}
	fnd := 0
	n := start
	for {
		fnd++
		n = n.Add(repeat)
		if start.Day() != n.Day() {
			break
		}
	}
	return fnd - expected
}

func TestDailyRepeat(t *testing.T) {
	nt := datetime.NewTimeOfDay
	ncd := datetime.NewCalendarDate
	loc, _ := time.LoadLocation("America/Los_Angeles")
	for _, tc := range []struct {
		date datetime.CalendarDate
	}{
		{ncd(2024, 1, 1)},
		{ncd(2024, 3, 10)}, // Lose an hour.
		{ncd(2024, 3, 11)},
		{ncd(2024, 11, 03)}, // Repeat an hour.
		{ncd(2024, 11, 04)},
	} {
		for _, start := range []datetime.TimeOfDay{
			nt(0, 0, 0), nt(1, 0, 0), nt(1, 30, 0), nt(23, 13, 0),
		} {
			for _, repeat := range []time.Duration{time.Hour, time.Hour * 5, time.Minute, time.Minute * 13} {
				action1 := newSpec("action1", start.Hour(), start.Minute(), start.Second(), testAction{"action1"})
				action1.Repeat.Interval = repeat
				scheduled := schedule.Scheduled[testAction]{
					Date:  tc.date,
					Specs: schedule.ActionSpecs[testAction]{action1},
				}
				active := []schedule.Active[testAction]{}
				for a := range scheduled.Active(datetime.Place{TZ: loc}) {
					active = append(active, a)
				}
				expected := expectedRepeats(ncd(2024, 1, 1).Time(start, loc), repeat)
				delta := findDelta(expected, tc.date.Time(start, loc), repeat)
				if got, want := len(active), expected+delta; got != want {
					if repeat == time.Minute*13 && tc.date.Day() == 10 {
						fmt.Printf("%v: %v: %v: got %v, want %v", tc.date, start, repeat, got, want)
					}
					t.Errorf("%v: %v: %v: got %v, want %v", tc.date, start, repeat, got, want)
				}
				prev := active[0].When
				for _, a := range active[1:] {
					if got, want := a.When.Sub(prev), repeat; got != want {
						t.Errorf("got %v, want %v", got, want)
					}
					prev = a.When
				}
			}
		}
	}
}

func TestDailyRepeatOverlapping(t *testing.T) {
	nt := datetime.NewTimeOfDay
	ncd := datetime.NewCalendarDate
	loc, _ := time.LoadLocation("America/Los_Angeles")
	for _, tc := range []struct {
		date datetime.CalendarDate
	}{
		{ncd(2024, 1, 1)},
		{ncd(2024, 3, 10)}, // Lose an hour.
		{ncd(2024, 3, 11)},
		{ncd(2024, 11, 03)}, // Repeat an hour.
		{ncd(2024, 11, 04)},
	} {

		for _, start := range []datetime.TimeOfDay{
			nt(0, 0, 0), nt(1, 0, 0), nt(1, 30, 0), nt(23, 13, 0),
		} {
			scheduled := schedule.Scheduled[testAction]{
				Date: tc.date,
			}

			numOps := map[string]int{}
			times := map[string][]time.Time{}
			expectedNumOps := map[string]int{}

			repeats := []time.Duration{time.Hour, time.Hour * 5, time.Minute, time.Minute * 13}
			for i, repeat := range repeats {
				name := fmt.Sprintf("action%d", i)
				action := newSpec(name, start.Hour(), start.Minute(), start.Second(), testAction{name})
				action.Repeat.Interval = repeat
				scheduled.Specs = append(scheduled.Specs, action)

				expected := expectedRepeats(ncd(2024, 1, 1).Time(start, loc), repeat)
				delta := findDelta(expected, tc.date.Time(start, loc), repeat)
				expectedNumOps[name] = expected + delta
			}

			for a := range scheduled.Active(datetime.Place{TZ: loc}) {
				times[a.Name] = append(times[a.Name], a.When)
				numOps[a.Name]++
			}

			if got, want := numOps, expectedNumOps; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}

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
		}
	}
}
