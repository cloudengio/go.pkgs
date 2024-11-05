// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule_test

import (
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/datetime/dates"
	"cloudeng.io/datetime/schedule"
)

func parseDateRangeList(year int, s ...string) dates.DateRangeList {
	if len(s) == 0 || len(s[0]) == 0 {
		return dates.DateRangeList{}
	}
	var dr dates.DateRangeList
	if err := dr.Parse(year, s); err != nil {
		panic(err)
	}
	return dr
}

func parseMontList(s string) dates.MonthList {
	var ml dates.MonthList
	if err := ml.Parse(s); err != nil {
		panic(err)
	}
	return ml
}

func TestDates(t *testing.T) {
	pdrl := parseDateRangeList
	pml := parseMontList
	unconstrained := dates.Constraints{}
	noFeb29 := dates.Constraints{Custom: []dates.Date{{Month: 2, Day: 29}}}
	for _, tc := range []struct {
		monthsFor   string
		mirror      bool
		ranges      string
		constraints dates.Constraints
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
			Ranges:       pdrl(tc.year, strings.Split(tc.ranges, ",")...),
			MirrorMonths: tc.mirror,
			Constraints:  tc.constraints,
		}
		if got, want := sd.EvaluateDateRanges(tc.year), pdrl(tc.year, strings.Split(tc.expected, ",")...); !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
