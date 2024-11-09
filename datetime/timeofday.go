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
type TimeOfDay uint32 //struct {
//	Hour   int
//	Minute int
//	Second int
//}

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
	hr, min, sec := t.Hour(), t.Minute(), t.Second()
	if hr > 23 {
		hr = 23
	}
	if min > 59 {
		min = 59
	}
	if sec > 59 {
		sec = 59
	}
	return NewTimeOfDay(hr, min, sec)
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

/*
// Before returns true if t is before t2.

	func (t TimeOfDay) Before(t2 TimeOfDay) bool {
		if t.Hour != t2.Hour {
			return t.Hour < t2.Hour
		}
		if t.Minute != t2.Minute {
			return t.Minute < t2.Minute
		}
		return t.Second < t2.Second
	}

// Equal returns true if t is equal to t2.

	func (t TimeOfDay) Equal(t2 TimeOfDay) bool {
		return t == t2
	}


	func (tl TimeOfDayList) Sort() {
		sort.Slice(tl, func(i, j int) bool { return tl[i].Before(tl[j]) })
	}
*/

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

func Time(yp YearAndPlace, date Date, tod TimeOfDay) time.Time {
	return time.Date(yp.Year, time.Month(date.Month()), date.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, yp.Place)
}

func TimeOfDayFromTime(t time.Time) TimeOfDay {
	return NewTimeOfDay(t.Hour(), t.Minute(), t.Second())
}
