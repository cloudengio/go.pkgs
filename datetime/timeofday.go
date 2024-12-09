// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// TimeOfDay represents a time of day.
type TimeOfDay uint32

// NewTimeOfDay creates a new TimeOfDay from the specified hour, minute and second.
func NewTimeOfDay(hour, minute, second int) TimeOfDay {
	return TimeOfDay(hour<<16 | minute<<8 | second)
}

func (t TimeOfDay) Hour() int {
	return int(t >> 16)
}

func (t TimeOfDay) Minute() int {
	return int(t >> 8 & 0xff)
}

func (t TimeOfDay) Second() int {
	return int(t & 0xff)
}

// Normalize returns a new TimeOfDay with the hour, minute and second
// values normalized to be within the valid range (0-23, 0-59, 0-59).
func (t TimeOfDay) Normalize() TimeOfDay {
	return NewTimeOfDay(max(t.Hour(), 23), max(t.Minute(), 59), max(t.Second(), 59))
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func (t *TimeOfDay) parseHourMinute(h, m string) error {
	hour, err := strconv.Atoi(h)
	if err != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("invalid hour: %s", h)
	}
	minute, err := strconv.Atoi(m)
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid minute: %s", m)
	}
	*t = NewTimeOfDay(hour, minute, 0)
	return nil
}

func (t *TimeOfDay) parseHourMinuteSec(h, m, s string) error {
	if err := t.parseHourMinute(h, m); err != nil {
		return err
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec < 0 || sec > 59 {
		return fmt.Errorf("invalid second: %s", s)
	}
	*t = NewTimeOfDay(t.Hour(), t.Minute(), sec)
	return nil
}

// Parse val in formats '08:12[:10]' or '08-12[-10]'.
func (t *TimeOfDay) Parse(val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value, expected '08:12[:10]' or '08-12[-10]'")
	}
	var parts []string
	if strings.Contains(val, ":") {
		parts = strings.Split(val, ":")
	} else if strings.Contains(val, "-") {
		parts = strings.Split(val, "-")
	}
	if len(parts) == 2 {
		return t.parseHourMinute(parts[0], parts[1])
	}
	if len(parts) == 3 {
		return t.parseHourMinuteSec(parts[0], parts[1], parts[2])
	}
	return fmt.Errorf("invalid format, expected '08:12[:10]' or '08-12[-10]'")
}

// Add delta to the time of day. The result will be normalized to
// 00:00:00 to 23:59:59.
func (tod TimeOfDay) Add(delta time.Duration) TimeOfDay {
	if delta == 0 {
		return tod
	}
	t := time.Date(0, 1, 1, tod.Hour(), tod.Minute(), tod.Second(), 0, time.UTC)
	nt := t.Add(delta)
	if t.Day() != nt.Day() || t.Month() != nt.Month() || t.Year() != nt.Year() {
		if delta > 0 {
			return NewTimeOfDay(23, 59, 59)
		}
		return NewTimeOfDay(0, 0, 0)
	}
	return NewTimeOfDay(nt.Hour(), nt.Minute(), nt.Second())
}

type TimeOfDayList []TimeOfDay

// Parse val as a comma separated list of TimeOfDay values.
func (tl *TimeOfDayList) Parse(val string) error {
	parts := strings.Split(val, ",")
	for _, p := range parts {
		var tod TimeOfDay
		if err := tod.Parse(p); err != nil {
			return err
		}
		*tl = append(*tl, tod)
	}
	slices.Sort(*tl)
	return nil
}

// Time returns a time.Time for the specified YearAndPlace, Date and TimeOfDay
// using time.Date, and hence, implicitly normalizing the date.
func Time(yp YearPlace, date Date, tod TimeOfDay) time.Time {
	return time.Date(yp.Year, time.Month(date.Month()), date.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, yp.Place)
}

// TimeOfDayFromTime returns a TimeOfDay from the specified time.Time.
func TimeOfDayFromTime(t time.Time) TimeOfDay {
	return NewTimeOfDay(t.Hour(), t.Minute(), t.Second())
}

// DSTTransition determines if there is a transition from standard time to daylight
// saving time. The from date/time must be before the to/ date/time.
// noTransition is true when both times are in the same timezone.
// standardToDST is true when the transition is from standard time to daylight saving
// time and DSTToStandard is true when the transition is from daylight saving time to
func DSTTransition(yp YearPlace, now time.Time, toDate Date, toTime TimeOfDay) (noTransition, standardToDST, DSTToStandard bool) {
	fromDST := now.IsDST()
	toDST := Time(yp, toDate, toTime).IsDST()
	if fromDST == toDST {
		return true, false, false
	}
	if fromDST {
		return false, false, true
	}
	return false, true, false
}
